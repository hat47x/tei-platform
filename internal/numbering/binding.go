// Package numbering binds a declared numbering Data to the counter and the
// formatting rules the contract uses.
//
// X-TEA read a numbering table by row ID (IDNUMBER) and took the counter, the
// digit count, the prefix, and the check-digit flag from columns (NRCURRENT,
// NRNUMDIGIT, TXPREFIX, FGWITHCD) that came from a table named by a system
// variable (D:Session.java L1980–2009). The row was written back with an
// optimistic check on UPDCOUNTER whose result was never inspected, so a
// concurrent update lost the issuance silently. Here the attributes are declared
// by role, and the contract refuses to issue when the declaration cannot say
// what the number looks like.
package numbering

import (
	"fmt"
	"strconv"

	"tei/internal/core"
	"tei/internal/ir"
)

// Attribute roles a numbering Data must declare.
const (
	RoleCounter  = "numbering-counter"
	RoleDigits   = "numbering-digits"
	RolePrefix   = "numbering-prefix"
	RoleCheck    = "numbering-check"
	RoleOverflow = "numbering-overflow"
)

// Declaration is the attributes a numbering Data declares, plus the contract
// rules that are not per-row.
type Declaration struct {
	DataID   core.CanonicalID
	Counter  core.CanonicalID
	Digits   core.CanonicalID
	Prefix   core.CanonicalID
	Check    core.CanonicalID
	Overflow core.CanonicalID
}

// Resolve reads the numbering declaration from the world.
//
// The counter and the digit count are required: without them the contract cannot
// say where the sequence is or how wide it is. The prefix, the check digit, and
// the overflow rule are optional; an empty column keeps the contract default,
// which is no prefix, no check digit, and stop at the limit.
func Resolve(world ir.World, dataID core.CanonicalID) (Declaration, error) {
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			declaration := Declaration{DataID: data.ID}
			for _, attribute := range data.Attributes {
				switch attribute.Role {
				case RoleCounter:
					declaration.Counter = attribute.ID
				case RoleDigits:
					declaration.Digits = attribute.ID
				case RolePrefix:
					declaration.Prefix = attribute.ID
				case RoleCheck:
					declaration.Check = attribute.ID
				case RoleOverflow:
					declaration.Overflow = attribute.ID
				}
			}
			var missing []string
			if declaration.Counter == "" {
				missing = append(missing, RoleCounter)
			}
			if declaration.Digits == "" {
				missing = append(missing, RoleDigits)
			}
			if len(missing) > 0 {
				return Declaration{}, fmt.Errorf("numbering data %s does not declare %v", dataID, missing)
			}
			return declaration, nil
		}
	}
	return Declaration{}, fmt.Errorf("numbering data %s is not declared", dataID)
}

// Bind turns one stored row into a contract and the state the counter is at.
//
// A row is one numbering rule. The digits are read per row because a table can
// hold several rules with different widths; the prefix and the overflow rule are
// read the same way, so a caller does not have to repeat them.
func (d Declaration) Bind(row map[core.CanonicalID]string) (Contract, State, error) {
	digits, err := parseInt(row[d.Digits], "digit width")
	if err != nil {
		return Contract{}, State{}, fmt.Errorf("numbering row: %s: %w", d.Digits, err)
	}
	current, err := parseInt(row[d.Counter], "counter")
	if err != nil {
		return Contract{}, State{}, fmt.Errorf("numbering row: %s: %w", d.Counter, err)
	}
	contract := Contract{Digits: digits}
	if d.Prefix != "" {
		contract.Prefix = row[d.Prefix]
	}
	if d.Check != "" {
		switch raw := row[d.Check]; raw {
		case "", "F":
		case "T":
			contract.CheckDigit = CheckDigitModulus10Weighted
		default:
			return Contract{}, State{}, fmt.Errorf("numbering row: %s: %q is neither T nor F", d.Check, raw)
		}
	}
	if d.Overflow != "" {
		if raw := row[d.Overflow]; raw != "" {
			contract.Overflow = Overflow(raw)
		}
	}
	if err := contract.validate(); err != nil {
		return Contract{}, State{}, fmt.Errorf("numbering row: %w", err)
	}
	if current < 0 {
		return Contract{}, State{}, fmt.Errorf("numbering row: counter %d is negative", current)
	}
	return contract, State{Current: current}, nil
}

func parseInt(value, label string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("%s is empty", label)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not a whole number", label, value)
	}
	return parsed, nil
}
