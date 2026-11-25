package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A row-set example states a table of rows and one expected answer for the
// whole table. An implementation is chosen only if it answers every such
// example as the declaration says, exactly as it is checked against one-record
// examples today.
//
// The expectations below are written from the meaning of every-at-most over the
// rows stated here, not read back from the implementation.

func rowSetOperator(cases ...semantic.OperatorCase) semantic.Operator {
	return semantic.Operator{
		ID: "sales.every_line_at_most", Symbol: "every_line_at_most",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
		Cases:             cases,
	}
}

func rowSetExample(answer string, rows ...map[string]string) semantic.OperatorCase {
	return semantic.OperatorCase{Rows: rows, RowExpect: answer}
}

func rowOf(value, limit string) map[string]string {
	return map[string]string{"value": value, "limit": limit}
}

func TestRowSetExamplesCheckTheSelectedImplementation(t *testing.T) {
	// (2, 10) and (5, 5) are both within their own limit, (5, 4) is not.
	// The two tables differ only in the second row, so an implementation that
	// read only the first row would agree with both and be caught here.
	contract := rowSetOperator(
		rowSetExample(semantic.RowExpectSatisfied, rowOf("2", "10"), rowOf("5", "5")),
		rowSetExample(semantic.RowExpectNotSatisfied, rowOf("2", "10"), rowOf("5", "4")),
	)
	implementation := everyAtMost(t)
	if err := validationeval.CheckImplementationMatchesContract(contract, validationeval.OperatorImplementation{
		Operator: "sales.every_line_at_most", Input: semantic.OperatorInputNumber,
		NumberRows: implementation.NumberRows,
	}); err != nil {
		t.Fatalf("the shipped every-at-most was refused: %v", err)
	}
}

// An example whose expected answer the implementation contradicts refuses the
// implementation, and the message names the operator and the answer it
// expected. The rows are unchanged from the first table; only the answer the
// declaration states is different.
func TestRowSetExampleRefusesAnImplementationThatDisagrees(t *testing.T) {
	contract := rowSetOperator(
		rowSetExample(semantic.RowExpectSatisfied, rowOf("2", "10"), rowOf("5", "4")),
	)
	implementation := everyAtMost(t)
	err := validationeval.CheckImplementationMatchesContract(contract, validationeval.OperatorImplementation{
		Operator: "sales.every_line_at_most", Input: semantic.OperatorInputNumber,
		NumberRows: implementation.NumberRows,
	})
	if err == nil {
		t.Fatal("an implementation that contradicts the example was accepted")
	}
	if !strings.Contains(err.Error(), "sales.every_line_at_most") ||
		!strings.Contains(err.Error(), semantic.RowExpectSatisfied) {
		t.Fatalf("the message does not name the operator and the expected answer: %q", err.Error())
	}
}

// The column order and the row order are read as written: the same rows in
// another column order state the same example, and an implementation is asked
// about the rows in the order the table wrote them.
func TestRowSetExampleReadsColumnsByName(t *testing.T) {
	contract := rowSetOperator(
		rowSetExample(semantic.RowExpectNotSatisfied, rowOf("5", "4"), rowOf("2", "10")),
	)
	implementation := everyAtMost(t)
	if err := validationeval.CheckImplementationMatchesContract(contract, validationeval.OperatorImplementation{
		Operator: "sales.every_line_at_most", Input: semantic.OperatorInputNumber,
		NumberRows: implementation.NumberRows,
	}); err != nil {
		t.Fatalf("a table whose first row is the counterexample was refused: %v", err)
	}
}

// A row-set example states no value for an input it does not name, so the
// contract cannot be checked and the shape is refused rather than read as an
// absent value.
func TestRowSetExampleRefusesAnUnstatedColumn(t *testing.T) {
	missing := semantic.OperatorCase{
		Rows:      []map[string]string{{"value": "2"}},
		RowExpect: semantic.RowExpectSatisfied,
	}
	contract := rowSetOperator(missing)
	implementation := everyAtMost(t)
	err := validationeval.CheckImplementationMatchesContract(contract, validationeval.OperatorImplementation{
		Operator: "sales.every_line_at_most", Input: semantic.OperatorInputNumber,
		NumberRows: implementation.NumberRows,
	})
	if err == nil || !strings.Contains(err.Error(), "TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH") {
		t.Fatalf("an example that leaves an input unstated was not refused as a shape mismatch: %v", err)
	}
}

// A row-scoped operator cannot be stated by a one-record example: that example
// would be read as if its single value described a set of rows.
func TestRowScopedOperatorRefusesAOneRecordExample(t *testing.T) {
	contract := rowSetOperator(semantic.OperatorCase{Value: "5", Satisfied: true})
	implementation := everyAtMost(t)
	err := validationeval.CheckImplementationMatchesContract(contract, validationeval.OperatorImplementation{
		Operator: "sales.every_line_at_most", Input: semantic.OperatorInputNumber,
		NumberRows: implementation.NumberRows,
	})
	if err == nil || !strings.Contains(err.Error(), "TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH") {
		t.Fatalf("a one-record example was not refused for a row-scoped operator: %v", err)
	}
}

// A one-record operator cannot be stated by a table of rows: the table answers
// a question the declaration never asks.
func TestRecordOperatorRefusesARowSetExample(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.non_negative", Symbol: "non_negative",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber},
		},
		Cases: []semantic.OperatorCase{rowSetExample(semantic.RowExpectSatisfied, rowOf("2", "10"))},
	}
	var nonNegative validationeval.NamedImplementation
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID == "tei.number.non-negative" {
			nonNegative = available
		}
	}
	if nonNegative.Number == nil {
		t.Fatal("the shipped tei.number.non-negative is missing")
	}
	err := validationeval.CheckImplementationMatchesContract(contract, validationeval.OperatorImplementation{
		Operator: "sales.non_negative", Input: semantic.OperatorInputNumber, Number: nonNegative.Number,
	})
	if err == nil || !strings.Contains(err.Error(), "TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH") {
		t.Fatalf("a row-set example was not refused for a one-record operator: %v", err)
	}
}
