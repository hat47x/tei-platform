package validationeval

import (
	"fmt"
	"math/big"

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
//
// The three-valued answer, the read Coverage and the operator's refutation
// statement are composed in the one place that decides it (decideRowScoped,
// row_scope.go), exactly as they are for the numeric row path: a counterexample
// that was seen settles the rule, and a relation that was only partly read
// cannot be answered satisfied.
func decideTypedRowScopedOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf(
			"semantic validation %s uses operator %s, which reads every row, but the rule names no relation; write \"per <relation>\"",
			rule.ID, rule.Operator)
	}

	rows := make(map[string][]RowValueReading, len(inputs))
	var coverage Coverage
	var reason *Undecided
	// A constant states one value for the whole relation. Its column is as long
	// as the rows that were read, so the value is written into every row once
	// the row count is known.
	var constants map[string]semantic.OperatorValue
	var width []RowValueReading

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
				if constants == nil {
					constants = make(map[string]semantic.OperatorValue, len(inputs))
				}
				constants[input.Name] = value
				continue
			}
			attribute = core.CanonicalID(assignment.Value)
		}
		reading, readErr := readTypedRows(rule, input, source, core.CanonicalID(rule.Relation), attribute)
		if readErr != nil {
			return Decision{}, readErr
		}
		// Every input of this one rule is read from this one relation, so all of
		// them report the same coverage. Two coverages would mean two reads, and
		// one answer must not mix them.
		if coverage != "" && reading.Coverage != coverage {
			return Decision{}, fmt.Errorf(
				"semantic validation %s read relation %s with two coverages (%q and %q); one answer cannot mix two reads",
				rule.ID, rule.Relation, coverage, reading.Coverage)
		}
		coverage = reading.Coverage
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
		if index == 0 {
			width = column
		}
		// A row that holds no value, or was not read, keeps its own reason. The
		// first such row in read order is reported; the implementation is still
		// asked, so a counterexample found in another row is not thrown away.
		if reason == nil {
			reason = firstTypedRowReason(attribute, column)
		}
	}

	for name, value := range constants {
		rows[name] = fillTypedConstant(width, value)
	}

	answer, err := implementation.ValueRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, reason)
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

// fillTypedConstant returns one column of the relation in which every row holds
// the same value. A constant input states one value for the whole relation, so
// the operator sees that value once per row rather than a column of one row.
func fillTypedConstant(width []RowValueReading, value semantic.OperatorValue) []RowValueReading {
	rows := make([]RowValueReading, len(width))
	for index := range rows {
		held := value
		if held.Number != nil {
			held.Number = new(big.Rat).Set(held.Number)
		}
		rows[index] = RowValueReading{State: ValuePresent, Value: held}
	}
	return rows
}

// firstTypedRowReason reports the first row that holds no value or was not read,
// in read order. It is the reason a row-scoped answer may be undecided; a
// decided answer must not be reported together with it (decideRowScoped checks
// that).
func firstTypedRowReason(attribute core.CanonicalID, rows []RowValueReading) *Undecided {
	for _, row := range rows {
		switch row.State {
		case ValueNull:
			return &Undecided{Reason: UndecidedValueNull, Attribute: attribute}
		case ValueNotFetched:
			return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}
		}
	}
	return nil
}
