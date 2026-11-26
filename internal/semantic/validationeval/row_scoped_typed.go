package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides a rule whose operator reads every row of a relation and
// declares an input of a type other than number.
//
// A column of days, texts or truths is read as that type. The numeric row reader
// would answer a question about numbers the rule never asked, so the rows are
// read by the reader that understands the declared type and handed to the
// implementation with the state of each row next to its value (decision section
// 8.4). What a rule reads stays decided by the declaration, so the
// implementation sees no Source.

// declaresTypedInput reports whether some declared input is read as a value of a
// type other than number. Such an input is read by the reader that understands
// its type, and its rows are decided by the typed row field (decision section
// 8.4).
func declaresTypedInput(inputs []semantic.OperatorInputDeclaration) bool {
	for _, input := range inputs {
		if input.Type != semantic.OperatorInputNumber {
			return true
		}
	}
	return false
}

// decideRowScopedTypedOperator decides a rule whose operator reads every row of
// a relation and declares at least one input of a type other than number.
func decideRowScopedTypedOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	reader, ok := source.(StatefulRowValueSource)
	if !ok {
		return Decision{}, fmt.Errorf(
			"semantic validation %s reads every row of relation %s as a value that is not a number, but this reader cannot read such a column",
			rule.ID, rule.Relation)
	}
	if implementation.ValueRows == nil {
		return Decision{}, fmt.Errorf(
			"operator %s reads every row of a relation as a value that is not a number, but its selected implementation does not decide such rows",
			rule.Operator)
	}

	rows := make(map[string][]RowValue, len(inputs))
	var coverage Coverage
	var reason *Undecided
	// A constant states one value for the whole relation. Its column is as long
	// as the rows that were read, so the value is written into every row once
	// the row count is known.
	var constantValues map[string]semantic.OperatorValue
	var width []RowValue
	for index, input := range inputs {
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf(
					"semantic validation %s does not assign operator %s input %q; every input after the first must be assigned",
					rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				value, err := semantic.ParseOperatorInputValue(input.Type, assignment.Value)
				if err != nil {
					return Decision{}, fmt.Errorf(
						"semantic validation %s assigns operator %s input %q the value %q, which is neither an attribute nor a value of type %s: %w",
						rule.ID, rule.Operator, input.Name, assignment.Value, input.Type, err)
				}
				if constantValues == nil {
					constantValues = make(map[string]semantic.OperatorValue, len(inputs))
				}
				constantValues[input.Name] = value
				continue
			}
			named, err := core.ParseCanonicalID(assignment.Value)
			if err != nil {
				return Decision{}, fmt.Errorf(
					"semantic validation %s assigns operator %s input %q an unreadable attribute %q",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			attribute = named
		}
		reading, err := reader.ReadRelationValues(rule.Relation, attribute)
		if err != nil {
			return Decision{}, fmt.Errorf(
				"semantic validation %s cannot read attribute %s over relation %s: %w",
				rule.ID, attribute, rule.Relation, err)
		}
		// Every input of this one rule is read from this one relation, so all of
		// them report the same coverage. Two coverages would mean two reads, and
		// one answer must not mix them (section 14.2 B-10).
		if coverage != "" && reading.Coverage != coverage {
			return Decision{}, fmt.Errorf(
				"semantic validation %s read relation %s with two coverages (%q and %q); one answer cannot mix two reads",
				rule.ID, rule.Relation, coverage, reading.Coverage)
		}
		coverage = reading.Coverage
		rows[input.Name] = reading.Rows
		if index == 0 {
			width = reading.Rows
		}
		// A row that holds no value, or was not read, keeps its own reason. The
		// first such row in read order is reported; the implementation is still
		// asked, so a counterexample found in another row is not thrown away.
		if reason == nil {
			reason = firstRowValueReason(attribute, rows[input.Name])
		}
	}
	for name, value := range constantValues {
		rows[name] = fillConstantRowValues(width, value)
	}

	answer, err := implementation.ValueRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, reason)
}

// firstRowValueReason reports the reason the first row that holds no value, or
// was not read, carries. It is the typed counterpart of firstRowReason, and it
// reads the state next to the value rather than the value itself.
func firstRowValueReason(attribute core.CanonicalID, rows []RowValue) *Undecided {
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

// fillConstantRowValues returns one column of the relation in which every row
// holds the same value. A constant input states one value for the whole
// relation, so the operator sees that value once per row rather than a column
// that changes.
func fillConstantRowValues(width []RowValue, value semantic.OperatorValue) []RowValue {
	rows := make([]RowValue, len(width))
	for i := range rows {
		rows[i] = RowValue{State: ValuePresent, Value: value}
	}
	return rows
}
