package validationread

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// Decisions is the answer of one rule for the rows of a Detail.
//
// A rule about all the rows of the relation has one answer, in Relation. A
// rule about one record has one answer for every row, in Rows, in the order
// of the rows. Exactly one of the two is set; which one is decided by the
// kind of the rule and never by what the rows happen to hold.
type Decisions struct {
	Rule     core.CanonicalID
	Relation *validationeval.Decision
	Rows     []validationeval.Decision
}

// Decide asks rule of the rows.
//
// A rule about one record is asked of every row that was read. When the rows
// are not all the rows of the relation, its answers still speak only of the
// rows read: nothing is said about a row that was not returned, and no row
// means no answer, not a satisfied rule.
//
// A kind this reader does not know is refused. It is not asked of the rows in
// either way by default: a new kind that is about all the rows would
// otherwise be answered row by row, and that answer would look decided.
func (d *Detail) Decide(rule semantic.Validation, operators validationeval.Operators) (Decisions, error) {
	decisions := Decisions{Rule: rule.ID}
	// A rule whose operator declares an input read over all rows is asked of
	// the rows as a collection (decision section 15.4). Answering it row by row
	// would look decided while the declaration says otherwise.
	rowScoped, operatorKnown := false, true
	if rule.Kind == semantic.ValidationOperator && rule.Operator != "" {
		rowScoped, operatorKnown = declaresRowScopedOperator(rule, operators)
	}
	switch {
	case rule.Kind == semantic.ValidationRelationUnique,
		rule.Kind == semantic.ValidationNoOverlap,
		rule.Kind == semantic.ValidationOperator && rowScoped:
		decision, err := validationeval.Decide(rule, d, operators)
		if err != nil {
			return Decisions{}, err
		}
		decisions.Relation = &decision
		return decisions, nil

	case rule.Kind == semantic.ValidationPositive,
		rule.Kind == semantic.ValidationNotAfter,
		rule.Kind == semantic.ValidationRequiredIf:
		decisions.Rows = make([]validationeval.Decision, 0, len(d.rows))
		for index := range d.rows {
			decision, err := validationeval.Decide(rule, Row{detail: d, index: index}, operators)
			if err != nil {
				// A value that cannot be read names its row itself.
				return Decisions{}, err
			}
			decisions.Rows = append(decisions.Rows, decision)
		}
		return decisions, nil

	case rule.Kind == semantic.ValidationOperator && !operatorKnown:
		// The rule names an operator this reader was not given a declaration
		// for, so how it is asked is unknown. Refusing says that; answering the
		// rows one by one would report an unrelated reader complaint instead.
		return Decisions{}, fmt.Errorf("semantic validation %s uses operator %s, which has no loaded declaration; this reader cannot tell whether the rule is asked of one row or of all the rows", rule.ID, rule.Operator)

	case rule.Kind == semantic.ValidationOperator:
		decisions.Rows = make([]validationeval.Decision, 0, len(d.rows))
		for index := range d.rows {
			decision, err := validationeval.Decide(rule, Row{detail: d, index: index}, operators)
			if err != nil {
				return Decisions{}, err
			}
			decisions.Rows = append(decisions.Rows, decision)
		}
		return decisions, nil

	default:
		return Decisions{}, fmt.Errorf("semantic validation %s is of kind %q; this reader does not know whether that is asked of one row or of all the rows", rule.ID, rule.Kind)
	}
}

func (d *Detail) readPresence(index int, attribute core.CanonicalID) (validationeval.PresenceReading, error) {
	if _, err := d.kindOf(attribute); err != nil {
		return validationeval.PresenceReading{}, err
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.PresenceReading{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.PresenceReading{State: validationeval.ValueNull}, nil
	}
	return validationeval.PresenceReading{State: validationeval.ValuePresent}, nil
}

// ReadPresence reads whether the row holds a value for attribute. The value
// is not parsed: an empty text is a value, and only SQL NULL is none.
func (r Row) ReadPresence(attribute core.CanonicalID) (validationeval.PresenceReading, error) {
	return r.detail.readPresence(r.index, attribute)
}

// declaresRowScopedOperator reports whether the operator this rule names
// declares an input read over all rows of a relation.
//
// The second result is false when the named operator is not among the loaded
// declarations. That is a different fact from "the operator reads one record":
// the rule cannot be asked row by row either, because its signature is unknown,
// so the caller reports the missing declaration instead of falling through to
// the row path and letting the reader blame itself.
func declaresRowScopedOperator(rule semantic.Validation, operators validationeval.Operators) (bool, bool) {
	for _, contract := range operators.Declared {
		if contract.ID != rule.Operator {
			continue
		}
		for _, input := range contract.Inputs {
			if input.OfEachRow {
				return true, true
			}
		}
		return false, true
	}
	return false, false
}
