package numbering

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// RoleRule is the optional role that names the attribute identifying a numbering
// rule inside a shared Data. X-TEA identified a rule by the IDNUMBER column
// (D:Session.java L1985–1994) and kept all rules in one table.
//
// Without the role, a Data holds one rule, and every row belongs to it. With the
// role, a Data can hold several rules, and each row names the rule it belongs to,
// which is what a caller passes as ruleID to Reserve.
const RoleRule = "numbering-rule"

// RuleBinding is the attribute that identifies a numbering rule in a Data.
type RuleBinding struct {
	DataID    core.CanonicalID
	Attribute core.CanonicalID
}

// BindRule resolves the rule identity attribute of a numbering Data.
//
// A Data that does not declare the role holds one rule; that is not an error, and
// the binding is empty. A Data that declares the role must carry a rule name in
// every row, which ValidateRows reads.
func (d Declaration) BindRule(world ir.World) (RuleBinding, error) {
	data, ok := numberingDataByID(world, d.DataID)
	if !ok {
		return RuleBinding{}, fmt.Errorf("numbering data %s is not declared", d.DataID)
	}
	for _, attribute := range data.Attributes {
		if attribute.Role == RoleRule {
			return RuleBinding{DataID: d.DataID, Attribute: attribute.ID}, nil
		}
	}
	return RuleBinding{DataID: d.DataID}, nil
}

// RuleFor returns the rule name the row belongs to, and whether the row names
// one. A Data without the identity role has one rule, so the caller's ruleID is
// the only name and no row needs to repeat it.
func (b RuleBinding) RuleFor(row map[core.CanonicalID]string) (string, bool) {
	if b.Attribute == "" {
		return "", false
	}
	value := row[b.Attribute]
	return value, value != ""
}

// RowsFor returns the rows of one rule. A Data without the identity role returns
// every row, because the Data holds one rule. A shared Data returns only the rows
// that name the rule, so two rules in one table do not read each other's rows.
func (b RuleBinding) RowsFor(ruleID string, rows []map[core.CanonicalID]string) ([]map[core.CanonicalID]string, error) {
	if b.Attribute == "" {
		return rows, nil
	}
	if ruleID == "" {
		return nil, fmt.Errorf("%w: the rule is not named", ErrNumbering)
	}
	out := make([]map[core.CanonicalID]string, 0, len(rows))
	for index, row := range rows {
		value := row[b.Attribute]
		if value == "" {
			// A row without a rule cannot be issued from a shared table: guessing
			// which rule it belongs to would issue the wrong rule's number.
			return nil, fmt.Errorf("numbering row %d: %s is empty", index+1, b.Attribute)
		}
		if value == ruleID {
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no row names rule %s", ErrNumbering, ruleID)
	}
	return out, nil
}

func numberingDataByID(world ir.World, dataID core.CanonicalID) (semantic.Data, bool) {
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID == dataID {
				return data, true
			}
		}
	}
	return semantic.Data{}, false
}
