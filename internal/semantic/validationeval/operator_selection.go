package validationeval

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file connects a declared operator to the implementation a composition
// selected for it.
//
// Three things stay separate. A document declares the operator (what it is
// called and what it reads). A composition, such as a Workspace manifest,
// selects an implementation for it by name. The running program carries
// implementations under those names. An operator rule is decided only when
// the selection names exactly one implementation the program carries.
//
// The implementations compiled into this binary are generic checks on one
// number. They are not a plugin registry; an implementation supplied by a
// plugin has to arrive through the existing plugin selection and lock, and
// then enters ResolveImplementations as one more NamedImplementation.

// NamedImplementation is an implementation the running program carries, under
// the name a composition selects it by.
type NamedImplementation struct {
	ID     string
	Input  semantic.OperatorInput
	Number func(value *big.Rat) (bool, error)
	// Numbers decides for a multi-input operator. Exactly one of Number and
	// Numbers is non-nil.
	Numbers func(values map[string]*big.Rat) (bool, error)
	// NumberRows decides for an operator whose input reads every row of a
	// relation. At most one of Number, Numbers and NumberRows is non-nil.
	NumberRows func(rows map[string][]NumberReading) (Tri, error)
}

// ImplementationSelection is one choice of a composition: this operator is
// decided by that implementation.
type ImplementationSelection struct {
	Operator       core.CanonicalID `json:"operator"`
	Implementation string           `json:"implementation"`
}

// ImplementationCall records that a selected implementation was actually
// asked to decide. Being selected and being called are different facts.
type ImplementationCall struct {
	Operator       core.CanonicalID
	Implementation string
}

// BuiltinImplementations lists the implementations compiled into this binary.
// It is a function so no caller can mutate a shared table.
func BuiltinImplementations() []NamedImplementation {
	return []NamedImplementation{
		{
			ID:    "tei.number.non-negative",
			Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) {
				return value.Sign() >= 0, nil
			},
			// The same check for an operator that reads every row of a
			// relation: one column, and a negative value in any row of it is a
			// counterexample. It is added here rather than as a second
			// implementation because the selection names one implementation
			// per operator, and which of the two functions decides is the
			// declaration's `of each row`, not a second choice a Workspace
			// makes.
			NumberRows: func(rows map[string][]NumberReading) (Tri, error) {
				if len(rows) != 1 {
					return TriMaybe, fmt.Errorf("non-negative over all rows reads one input; %d were supplied", len(rows))
				}
				for _, column := range rows {
					for _, row := range column {
						switch row.State {
						case ValuePresent:
							if row.Value == nil {
								return TriMaybe, fmt.Errorf("a row was reported as present without a value")
							}
							if row.Value.Sign() < 0 {
								return TriNo, nil
							}
						case ValueNull, ValueNotFetched:
							// A row that holds no value, or was not read, keeps
							// the answer from being certain.
							return TriMaybe, nil
						default:
							return TriMaybe, fmt.Errorf("a row was reported in the unknown state %q", row.State)
						}
					}
				}
				return TriYes, nil
			},
		},
		{
			ID:    "tei.number.non-zero",
			Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) {
				return value.Sign() != 0, nil
			},
		},
		{
			// tei.number.within decides "low <= value <= high" by input name.
			// It reads inputs named value, low, and high; a declaration that
			// names them otherwise is not this operator and is reported.
			ID:    "tei.number.within",
			Input: semantic.OperatorInputNumber,
			Numbers: func(values map[string]*big.Rat) (bool, error) {
				v, ok := values["value"]
				if !ok {
					return false, fmt.Errorf("within reads an input named \"value\"; the rule supplied no value input")
				}
				low, ok := values["low"]
				if !ok {
					return false, fmt.Errorf("within reads an input named \"low\"; the rule supplied no low input")
				}
				high, ok := values["high"]
				if !ok {
					return false, fmt.Errorf("within reads an input named \"high\"; the rule supplied no high input")
				}
				// A range whose lower bound is above its upper bound is a
				// wrong range specification, not an out-of-range value. It is
				// reported and never answered as not-satisfied; low and high
				// are not swapped to make it look decidable.
				if low.Cmp(high) > 0 {
					return false, fmt.Errorf("within: low (%s) is greater than high (%s); the range specification is wrong", low.RatString(), high.RatString())
				}
				return v.Cmp(low) >= 0 && v.Cmp(high) <= 0, nil
			},
		},
	}
}

// ResolveImplementations turns the selections of a composition into operator
// implementations. Every selection must name exactly one available
// implementation, and one operator may be selected only once. Nothing is
// chosen by position, and an implementation nobody selected is not returned.
//
// called, when not nil, receives one entry each time a returned implementation
// is asked to decide. It observes the call; it cannot change the result.
func ResolveImplementations(selected []ImplementationSelection, available []NamedImplementation, called func(ImplementationCall)) ([]OperatorImplementation, error) {
	seen := map[core.CanonicalID]bool{}
	out := make([]OperatorImplementation, 0, len(selected))
	for _, selection := range selected {
		if selection.Operator == "" || selection.Implementation == "" {
			return nil, fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_INVALID: a selection needs an operator and an implementation")
		}
		if seen[selection.Operator] {
			return nil, fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS: operator %s has more than one selected implementation", selection.Operator)
		}
		seen[selection.Operator] = true

		var matches []NamedImplementation
		for _, candidate := range available {
			if candidate.ID == selection.Implementation {
				matches = append(matches, candidate)
			}
		}
		switch {
		case len(matches) == 0:
			return nil, fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE: operator %s selects implementation %q, which this tei does not carry (available: %s)",
				selection.Operator, selection.Implementation, availableImplementationNames(available))
		case len(matches) > 1:
			return nil, fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS: implementation %q exists %d times; exactly one must be available",
				selection.Implementation, len(matches))
		}
		match := matches[0]
		implementation := OperatorImplementation{Operator: selection.Operator, Input: match.Input}
		if match.Number != nil {
			call := ImplementationCall{Operator: selection.Operator, Implementation: match.ID}
			decide := match.Number
			implementation.Number = func(value *big.Rat) (bool, error) {
				if called != nil {
					called(call)
				}
				return decide(value)
			}
		}
		if match.Numbers != nil {
			call := ImplementationCall{Operator: selection.Operator, Implementation: match.ID}
			decide := match.Numbers
			implementation.Numbers = func(values map[string]*big.Rat) (bool, error) {
				if called != nil {
					called(call)
				}
				return decide(values)
			}
		}
		if match.NumberRows != nil {
			call := ImplementationCall{Operator: selection.Operator, Implementation: match.ID}
			decide := match.NumberRows
			implementation.NumberRows = func(rows map[string][]NumberReading) (Tri, error) {
				if called != nil {
					called(call)
				}
				return decide(rows)
			}
		}
		out = append(out, implementation)
	}
	return out, nil
}

func availableImplementationNames(available []NamedImplementation) string {
	seen := map[string]bool{}
	var names []string
	for _, candidate := range available {
		if !seen[candidate.ID] {
			seen[candidate.ID] = true
			names = append(names, candidate.ID)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
