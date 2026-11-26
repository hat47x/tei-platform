package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// decideTypedRowScopedOperator decides a rule whose operator reads every row of
// a relation in a type other than number (decision section 8.4).
//
// Every input's column is read in the type its declaration states, and a
// constant is one value for the whole relation: it belongs to no record, so the
// reader is not asked about it. A value the reader did not read leaves the rule
// undecided rather than being read as the zero of its type.
func decideTypedRowScopedOperator(rule semantic.Validation, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf(
			"semantic validation %s uses operator %s, which reads every row, but the rule names no relation; write \"per <relation>\"",
			rule.ID, rule.Operator)
	}

	rows := make(map[string][]RowValueReading, len(inputs))
	var reason *Undecided
	constantColumns := map[string][]RowValueReading{}

	for index, input := range inputs {
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				value, parseErr := semantic.ParseOperatorInputValue(input.Type, assignment.Value)
				if parseErr != nil {
					return Decision{}, fmt.Errorf("semantic validation %s cannot read operator %s input %q as %s: %w", rule.ID, rule.Operator, input.Name, input.Type, parseErr)
				}
				constantColumns[input.Name] = append(constantColumns[input.Name], RowValueReading{
					State: ValuePresent,
					Value: value,
				})
				continue
			}
			attribute = core.CanonicalID(assignment.Value)
		}
		reading, readErr := readTypedRows(rule, input, source, core.CanonicalID(rule.Relation), attribute)
		if readErr != nil {
			return Decision{}, readErr
		}
		column, stopped, stateErr := typedRowColumn(rule, attribute, reading)
		if stateErr != nil {
			return Decision{}, stateErr
		}
		if stopped != nil {
			if reason == nil {
				reason = stopped
			}
			continue
		}
		rows[input.Name] = column
	}

	// A constant states one value for the whole relation, so its column holds
	// that value for every row the relation was read into.
	for name, value := range constantColumns {
		rows[name] = value
	}

	if reason != nil {
		return Decision{Undecided: reason}, nil
	}

	result, err := implementation.ValueRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decided(relationResult(result))
}

// typedRowColumn returns the rows one input's column was read into, or the
// reason the rule cannot be decided. Exactly one of the first two results is set
// when err is nil.
func typedRowColumn(rule semantic.Validation, attribute core.CanonicalID, reading TypedRowReading) ([]RowValueReading, *Undecided, error) {
	switch reading.State {
	case ValuePresent:
		return reading.Values, nil, nil
	case ValueNull, ValueNotFetched:
		reason := UndecidedValueNull
		if reading.State == ValueNotFetched {
			reason = UndecidedValueNotFetched
		}
		return nil, &Undecided{Reason: reason, Attribute: attribute}, nil
	default:
		return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
	}
}

// relationResult maps a row-scoped answer to the Result a rule carries.
func relationResult(tri Tri) Result {
	if tri == TriYes {
		return ResultSatisfied
	}
	if tri == TriNo {
		return ResultNotSatisfied
	}
	return Result("")
}
