package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides a rule whose operator reads every row of a relation,
// written `of each row` on an input of the declaration.
//
// The declaration decides that such a rule is asked once for the whole
// relation rather than once per row, so every input of it is read over the
// rows: the subject attribute supplies the first input's column, and every
// later input names the attribute its column comes from. One call to
// ReadRelationNumbers is one reading, so all the columns of one rule come from
// one read Attempt, which is what section 14.2 B-10 asks for.
//
// The implementation is a pure function: it receives the states of the rows
// that were read and answers TriYes, TriNo or TriMaybe. It sees no Source, so
// what a rule reads stays decided by the declaration and never by the code
// that implements it. Whether a partly read relation may still be answered
// not-satisfied is the operator's own `refutes on some rows` statement, and
// that composition happens in decideRowScoped and nowhere else.

// decideOperatorRows decides one operator rule that reads every row of a
// relation.
//
// The reader is asked through StatefulRowNumberSource, the same capability
// shape StatefulPeriodSource already uses: a reader that cannot read the rows
// of a relation is refused here rather than asked row by row, because a rule
// about all the rows cannot be answered from one row.
func decideOperatorRows(rule semantic.Validation, contract semantic.Operator, source StatefulSource, implementation OperatorImplementation) (Decision, error) {
	rows, ok := source.(StatefulRowNumberSource)
	if !ok {
		return Decision{}, fmt.Errorf("semantic validation %s uses operator %s, which reads every row of relation %s, but this reader cannot read the rows of a relation", rule.ID, rule.Operator, rule.Relation)
	}
	if implementation.NumberRows == nil {
		return Decision{}, fmt.Errorf("the implementation of operator %s cannot decide a rule that reads every row of a relation", rule.Operator)
	}
	inputs := contractInputs(contract)
	columns := make(map[string][]NumberReading, len(inputs))
	attributes := make([]core.CanonicalID, len(inputs))
	coverage := Coverage("")
	for index, input := range inputs {
		// The first input is the attribute the rule constrains.
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
			}
			// An input read over the rows names the attribute its column comes
			// from. A constant has no column: reading it as one value among
			// the rows would answer the whole relation from one record.
			if !isAttributeReference(assignment.Value) {
				return Decision{}, fmt.Errorf("semantic validation %s assigns the constant %q to operator %s input %q, which is read over all rows; an input read over the rows names the attribute its column comes from", rule.ID, assignment.Value, rule.Operator, input.Name)
			}
			attribute = core.CanonicalID(assignment.Value)
		}
		reading, err := rows.ReadRelationNumbers(rule.Relation, attribute)
		if err != nil {
			return Decision{}, fmt.Errorf("semantic validation %s cannot evaluate relation %s / attribute %s: %w", rule.ID, rule.Relation, attribute, err)
		}
		// A reading that contradicts itself is the reader's error, not a value
		// that was absent: it is reported instead of being recorded as "no
		// value", which would answer a question about the data with a
		// statement about the read.
		for _, row := range reading.Rows {
			switch row.State {
			case ValuePresent:
				if row.Value == nil {
					return Decision{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s as present without a value", rule.ID, attribute)
				}
			case ValueNull, ValueNotFetched:
				if row.Value != nil {
					return Decision{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s as %s together with a value", rule.ID, attribute, row.State)
				}
			default:
				return Decision{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, row.State)
			}
		}
		if index == 0 {
			coverage = reading.Coverage
		} else if reading.Coverage != coverage {
			return Decision{}, fmt.Errorf("semantic validation %s: the reader reports relation %s with the coverage %q for %s and %q for %s; one rule is decided from one reading", rule.ID, rule.Relation, coverage, attributes[0], reading.Coverage, attribute)
		}
		attributes[index] = attribute
		columns[input.Name] = reading.Rows
	}
	answer, err := implementation.NumberRows(columns)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, rowReason(rule, inputs, attributes, columns))
}

// rowReason names the reason a row that was read gives no value, or nil when
// every row of every column was read with a value.
//
// It is reported before the coverage, because reading more rows cannot settle a
// value that is absent, or was not read, in a row that was read. An
// implementation that answered certainly around such a row is refused by
// decideRowScoped: it cannot have decided around a value it never saw.
func rowReason(rule semantic.Validation, inputs []semantic.OperatorInputDeclaration, attributes []core.CanonicalID, columns map[string][]NumberReading) *Undecided {
	for index, input := range inputs {
		for _, row := range columns[input.Name] {
			switch row.State {
			case ValueNull:
				return &Undecided{Reason: UndecidedValueNull, Attribute: attributes[index], Relation: rule.Relation}
			case ValueNotFetched:
				return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attributes[index], Relation: rule.Relation}
			}
		}
	}
	return nil
}
