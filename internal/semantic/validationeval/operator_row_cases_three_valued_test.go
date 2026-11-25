package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// Decision section 8.1's table of three-valued row examples, written by hand
// from the operator's meaning: every row's value is at most its own limit, and
// a row that cannot be compared cannot rescue a comparison that failed.
//
//	(value, limit) rows                     every-at-most
//	(2, 10), (5, 5)                         TriYes
//	(2, 10), (5, 4)                         TriNo
//	(no value, 10), (5, 4)                  TriNo
//	(5, 4), (no value, 10)                  TriNo
//	(not-fetched, 10), (5, 4)               TriNo
//	(5, 4), (not-fetched, 10)               TriNo
//	(not-fetched, 10), (5, 5)               TriMaybe
//	(no value, 10), (5, 5)                  TriMaybe
//
// The answer is the same whichever order the rows are written in.

func threeValuedRowContract() semantic.Operator {
	return semantic.Operator{
		ID:     "sales.every_line_at_most",
		Symbol: "every_line_at_most",
		Input:  semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber},
		},
		RefutesOnSomeRows: true,
	}
}

func rowCase(rows [][2]string, answer semantic.OperatorRowExpect) semantic.OperatorRowCase {
	out := semantic.OperatorRowCase{Answer: answer}
	for _, row := range rows {
		out.Rows = append(out.Rows, map[string]string{"value": row[0], "limit": row[1]})
	}
	return out
}

func TestThreeValuedRowExamplesDecideAsTheTableStates(t *testing.T) {
	noValue := semantic.RowCellNoValue
	notFetched := semantic.RowCellNotFetched
	for _, tc := range []struct {
		name   string
		rows   [][2]string
		answer semantic.OperatorRowExpect
	}{
		{name: "every row at most", rows: [][2]string{{"2", "10"}, {"5", "5"}}, answer: semantic.OperatorRowSatisfied},
		{name: "one row above", rows: [][2]string{{"2", "10"}, {"5", "4"}}, answer: semantic.OperatorRowNotSatisfied},
		{name: "no value then a counterexample", rows: [][2]string{{noValue, "10"}, {"5", "4"}}, answer: semantic.OperatorRowNotSatisfied},
		{name: "counterexample then no value", rows: [][2]string{{"5", "4"}, {noValue, "10"}}, answer: semantic.OperatorRowNotSatisfied},
		{name: "unread then a counterexample", rows: [][2]string{{notFetched, "10"}, {"5", "4"}}, answer: semantic.OperatorRowNotSatisfied},
		{name: "counterexample then unread", rows: [][2]string{{"5", "4"}, {notFetched, "10"}}, answer: semantic.OperatorRowNotSatisfied},
		{name: "unread row, nothing above", rows: [][2]string{{notFetched, "10"}, {"5", "5"}}, answer: semantic.OperatorRowUndecided},
		{name: "a row with no value, nothing above", rows: [][2]string{{noValue, "10"}, {"5", "5"}}, answer: semantic.OperatorRowUndecided},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract := threeValuedRowContract()
			contract.RowCases = []semantic.OperatorRowCase{rowCase(tc.rows, tc.answer)}
			if err := validationeval.CheckImplementationMatchesContract(contract, resolvedEveryAtMost(t, contract)); err != nil {
				t.Fatalf("the shipped implementation disagrees with the table: %v", err)
			}
		})
	}
}

func resolvedEveryAtMost(t *testing.T, contract semantic.Operator) validationeval.OperatorImplementation {
	t.Helper()
	available := shippedEveryAtMost(t)
	return validationeval.OperatorImplementation{
		Operator:   contract.ID,
		Input:      available.Input,
		NumberRows: available.NumberRows,
	}
}

// An example whose rows hold values but whose answer claims silence is
// refused: the table would state that the rows cannot be compared when they
// plainly can.
func TestAnUndecidedExampleWithComparableRowsIsRefused(t *testing.T) {
	contract := threeValuedRowContract()
	contract.RowCases = []semantic.OperatorRowCase{
		rowCase([][2]string{{"2", "10"}, {"5", "5"}}, semantic.OperatorRowUndecided),
	}
	err := validationeval.CheckImplementationMatchesContract(contract, resolvedEveryAtMost(t, contract))
	if err == nil {
		t.Fatal("an undecided example over comparable rows was accepted")
	}
	if !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_MISMATCH") {
		t.Fatalf("the refusal does not name the mismatch: %v", err)
	}
	if !strings.Contains(err.Error(), "undecided") {
		t.Fatalf("the refusal does not say what the example states: %v", err)
	}
}

// A row cell that is neither a decimal, nor empty, nor `not-fetched` is
// refused rather than read as a number.
func TestARowExampleRefusesAnUnreadableCell(t *testing.T) {
	contract := threeValuedRowContract()
	contract.RowCases = []semantic.OperatorRowCase{
		rowCase([][2]string{{"abc", "10"}}, semantic.OperatorRowSatisfied),
	}
	err := validationeval.CheckImplementationMatchesContract(contract, resolvedEveryAtMost(t, contract))
	if err == nil {
		t.Fatal("a row example with an unreadable cell was accepted")
	}
}
