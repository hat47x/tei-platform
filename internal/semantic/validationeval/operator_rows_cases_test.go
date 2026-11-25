package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A row-scoped example states the rows of the relation and one answer. An
// implementation that answers an example differently is not an implementation
// of that operator, so it is never used for real data (decision section 8,
// "one table is one example"; section 8.1 fixes the form).
//
// The examples below are written by hand from tei.number.every-at-most's
// meaning: every row's value is at most its own limit.

func everyAtMostRowContract() semantic.Operator {
	satisfied := true
	notSatisfied := false
	return semantic.Operator{
		ID:     "sales.every_line_at_most",
		Symbol: "every_line_at_most",
		Input:  semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber},
		},
		RefutesOnSomeRows: true,
		Cases: []semantic.OperatorCase{
			{
				Rows: []map[string]string{
					{"value": "2", "limit": "10"},
					{"value": "5", "limit": "5"},
				},
				RowAnswer: &satisfied,
			},
			{
				Rows: []map[string]string{
					{"value": "2", "limit": "10"},
					{"value": "5", "limit": "4"},
				},
				RowAnswer: &notSatisfied,
			},
		},
	}
}

func resolvedForContract(t *testing.T, contract semantic.Operator) validationeval.OperatorImplementation {
	t.Helper()
	available := shippedEveryAtMost(t)
	return validationeval.OperatorImplementation{
		Operator:   contract.ID,
		Input:      available.Input,
		NumberRows: available.NumberRows,
	}
}

// The shipped implementation answers the rows of both examples as the tables
// state them, so the selection is accepted.
func TestRowScopedExamplesAgreeWithTheShipImplementation(t *testing.T) {
	contract := everyAtMostRowContract()
	if err := validationeval.CheckImplementationMatchesContract(contract, resolvedForContract(t, contract)); err != nil {
		t.Fatalf("the shipped implementation disagrees with its own examples: %v", err)
	}
}

// An implementation that answers the satisfied example as not satisfied is
// refused, and the refusal names the example's rows.
func TestARowScopedExampleRefusesADisagreeingImplementation(t *testing.T) {
	contract := everyAtMostRowContract()
	implementation := validationeval.OperatorImplementation{
		Operator: contract.ID,
		Input:    semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			// Always "does not hold": the first example expects that it holds.
			return validationeval.TriNo, nil
		},
	}
	err := validationeval.CheckImplementationMatchesContract(contract, implementation)
	if err == nil {
		t.Fatal("an implementation that contradicts an example was accepted")
	}
	if !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_MISMATCH") {
		t.Fatalf("the refusal does not name the mismatch: %v", err)
	}
	if !strings.Contains(err.Error(), "limit: 10, value: 2") {
		t.Fatalf("the refusal does not name the rows of the example: %v", err)
	}
}

// A row-scoped example must state its rows and an explicit answer; a structure
// left over from the one-record form is refused rather than read as an answer.
func TestARowScopedExampleMustStateRowsAndAnAnswer(t *testing.T) {
	contract := everyAtMostRowContract()
	withoutAnswer := contract
	withoutAnswer.Cases = []semantic.OperatorCase{{
		Rows: []map[string]string{{"value": "2", "limit": "10"}},
	}}
	if err := validationeval.CheckImplementationMatchesContract(withoutAnswer, resolvedForContract(t, contract)); err == nil {
		t.Fatal("an example without an answer was accepted")
	}

	withOneRecordValue := contract
	withOneRecordValue.Cases = []semantic.OperatorCase{{
		Value:     "2",
		Rows:      []map[string]string{{"value": "2", "limit": "10"}},
		RowAnswer: contract.Cases[0].RowAnswer,
	}}
	if err := validationeval.CheckImplementationMatchesContract(withOneRecordValue, resolvedForContract(t, contract)); err == nil {
		t.Fatal("an example that also states a one-record value was accepted")
	}

	missingInput := contract
	missingInput.Cases = []semantic.OperatorCase{{
		Rows:      []map[string]string{{"value": "2"}},
		RowAnswer: contract.Cases[0].RowAnswer,
	}}
	if err := validationeval.CheckImplementationMatchesContract(missingInput, resolvedForContract(t, contract)); err == nil {
		t.Fatal("an example that states no value for a declared input was accepted")
	}

	unknownInput := contract
	unknownInput.Cases = []semantic.OperatorCase{{
		Rows:      []map[string]string{{"value": "2", "limit": "10", "extra": "3"}},
		RowAnswer: contract.Cases[0].RowAnswer,
	}}
	if err := validationeval.CheckImplementationMatchesContract(unknownInput, resolvedForContract(t, contract)); err == nil {
		t.Fatal("an example column that is not an input was accepted")
	}
}

// A row-scoped example is not the one-record form: the solver must not read it
// as a value for a single input.
func TestRowScopedExamplesAreNotReadAsOneValue(t *testing.T) {
	contract := everyAtMostRowContract()
	// A one-input operator cannot have a row-scoped example: the rows name
	// columns the declaration does not have.
	oneInput := semantic.Operator{
		ID:     contract.ID,
		Symbol: contract.Symbol,
		Input:  semantic.OperatorInputNumber,
		Cases:  contract.Cases,
	}
	oneInput.Cases = []semantic.OperatorCase{{
		Rows:      []map[string]string{{"value": "2", "limit": "10"}},
		RowAnswer: contract.Cases[0].RowAnswer,
	}}
	if err := validationeval.CheckImplementationMatchesContract(oneInput, validationeval.OperatorImplementation{
		Operator:   oneInput.ID,
		Input:      semantic.OperatorInputNumber,
		NumberRows: shippedEveryAtMost(t).NumberRows,
	}); err == nil {
		t.Fatal("a row-scoped example was accepted for an operator with one input")
	}
}
