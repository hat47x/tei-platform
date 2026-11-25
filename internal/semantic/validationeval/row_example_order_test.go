package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// namedImplementation returns the shipped implementation with that ID. The
// expectations are written from what the declaration says, not read back from
// the implementation, but the implementation under test is the shipped one.
func namedImplementation(t *testing.T, id string) (validationeval.OperatorImplementation, bool) {
	t.Helper()
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID != id {
			continue
		}
		return validationeval.OperatorImplementation{
			Operator:   "sales.every_line",
			Input:      available.Input,
			Number:     available.Number,
			Numbers:    available.Numbers,
			NumberRows: available.NumberRows,
			Absent:     available.Absent,
		}, true
	}
	return validationeval.OperatorImplementation{}, false
}

// contractOf states the declared signature the examples in this file are read
// against: a row-scoped operator that refutes on some rows.
func contractOf(rows ...map[string]string) semantic.Operator {
	return semantic.Operator{
		ID: "sales.every_line", Symbol: "every_line",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
		RowCases: []semantic.OperatorRowCase{
			{Answer: semantic.OperatorRowSatisfied, Rows: rows},
		},
	}
}

// Section 8.1 fixes the row-set example as a truth table that includes
// counterexamples, and it states the expectation as the order of the rows does
// not matter: a witnessed counterexample is a counterexample whatever the other
// rows hold, so the same rows in either order state the same example.
//
// The two expectations below are drawn from that section, not from the shipped
// implementation's behaviour: an example whose rows include a counterexample
// states not-satisfied, and an example whose every row was left unread states
// undecided because there is nothing to compare.
func TestARowSetExampleDoesNotDependOnTheRowOrder(t *testing.T) {
	implementation, ok := namedImplementation(t, "tei.number.every-at-most")
	if !ok {
		t.Fatal("the shipped tei.number.every-at-most is missing")
	}
	// The unread row comes first, the counterexample after it.
	unreadFirst := contractOf(
		map[string]string{"value": "not-fetched", "limit": "10"},
		map[string]string{"value": "5", "limit": "4"},
	)
	// The same two rows in the other order.
	counterexampleFirst := contractOf(
		map[string]string{"value": "5", "limit": "4"},
		map[string]string{"value": "not-fetched", "limit": "10"},
	)

	first := validationeval.CheckImplementationMatchesContract(unreadFirst, implementation)
	second := validationeval.CheckImplementationMatchesContract(counterexampleFirst, implementation)

	// Both orders state the same example, so both are refused, and the refusal
	// says the same thing: the answer does not depend on the row order.
	if first == nil || second == nil {
		t.Fatalf("an example that includes a counterexample was accepted as satisfied (%v, %v)", first, second)
	}
	if !strings.Contains(first.Error(), "not-satisfied") {
		t.Fatalf("the unread row first example is not not-satisfied: %v", first)
	}
	if first.Error() != second.Error() {
		t.Fatalf("the answer changed with the row order: %q vs %q", first.Error(), second.Error())
	}
}

// An example whose every row was left unread states that the rows cannot be
// compared, so its answer is undecided rather than satisfied or not-satisfied.
func TestARowSetExampleWithOnlyUnreadRowsIsUndecided(t *testing.T) {
	implementation, ok := namedImplementation(t, "tei.number.every-at-most")
	if !ok {
		t.Fatal("the shipped tei.number.every-at-most is missing")
	}
	contract := contractOf(map[string]string{"value": "not-fetched", "limit": "10"})

	err := validationeval.CheckImplementationMatchesContract(contract, implementation)
	if err == nil {
		t.Fatal("an example whose rows were all left unread was accepted as satisfied")
	}
	if !strings.Contains(err.Error(), "undecided") {
		t.Fatalf("the answer is not undecided: %v", err)
	}
}

// The control for the order test: two counterexamples and no unread row. If the
// order test above fails, this control says whether the row order alone moved
// the answer, or whether an unread row hid the counterexample.
func TestARowSetExampleOfOnlyCounterexamplesDoesNotDependOnTheRowOrder(t *testing.T) {
	implementation, ok := namedImplementation(t, "tei.number.every-at-most")
	if !ok {
		t.Fatal("the shipped tei.number.every-at-most is missing")
	}
	first := validationeval.CheckImplementationMatchesContract(contractOf(
		map[string]string{"value": "5", "limit": "4"},
		map[string]string{"value": "7", "limit": "6"},
	), implementation)
	second := validationeval.CheckImplementationMatchesContract(contractOf(
		map[string]string{"value": "7", "limit": "6"},
		map[string]string{"value": "5", "limit": "4"},
	), implementation)

	if first == nil || second == nil {
		t.Fatalf("an example that only states counterexamples was accepted as satisfied (%v, %v)", first, second)
	}
	if !strings.Contains(first.Error(), "not-satisfied") {
		t.Fatalf("the answer is not not-satisfied: %v", first)
	}
	if first.Error() != second.Error() {
		t.Fatalf("the answer changed with the row order with no unread row: %q vs %q", first.Error(), second.Error())
	}
}
