package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// Decision section 8.1 sends the empty relation to this change and refuses to
// read it as one row that holds no value. Every-at-most states that each row's
// value is at most its own limit; with no row there is nothing to refute, so
// the contract holds, and the answer does not depend on the declaration's
// `refutes on some rows` (that statement is about a partly read relation, and
// an empty relation was read completely).
//
// The expectations are written from that meaning, not read back from the
// implementation.

func emptyRelationContract(refutes bool) semantic.Operator {
	contract := threeValuedRowContract()
	contract.RefutesOnSomeRows = refutes
	contract.RowCases = []semantic.OperatorRowCase{{
		Answer: semantic.OperatorRowSatisfied,
		Empty:  true,
	}}
	return contract
}

func TestAnEmptyRelationHoldsEveryAtMost(t *testing.T) {
	for _, refutes := range []bool{true, false} {
		contract := emptyRelationContract(refutes)
		if err := validationeval.CheckImplementationMatchesContract(contract, resolvedEveryAtMost(t, contract)); err != nil {
			t.Fatalf("refutes=%v: the shipped implementation disagrees about an empty relation: %v", refutes, err)
		}
	}
}

// Claiming that the empty relation does not hold is refused: no row can
// refute a rule about every row.
func TestAnEmptyRelationDoesNotRefuteItself(t *testing.T) {
	contract := emptyRelationContract(true)
	contract.RowCases = []semantic.OperatorRowCase{{
		Answer: semantic.OperatorRowNotSatisfied,
		Empty:  true,
	}}
	err := validationeval.CheckImplementationMatchesContract(contract, resolvedEveryAtMost(t, contract))
	if err == nil {
		t.Fatal("an empty relation was accepted as not-satisfied")
	}
	if !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_MISMATCH") {
		t.Fatalf("the refusal does not name the mismatch: %v", err)
	}
}

// An example that states the empty relation and rows at once says two things;
// it is refused rather than read as either.
func TestAnExampleCannotStateNoRowsAndRowsAtOnce(t *testing.T) {
	contract := emptyRelationContract(true)
	contract.RowCases = []semantic.OperatorRowCase{{
		Answer: semantic.OperatorRowSatisfied,
		Empty:  true,
		Rows:   []map[string]string{{"value": "2", "limit": "10"}},
	}}
	err := validationeval.CheckImplementationMatchesContract(contract, resolvedEveryAtMost(t, contract))
	if err == nil {
		t.Fatal("an example stating the empty relation and rows was accepted")
	}
}

// A tool that builds a contract by hand leaves the answer unset; that is not
// the empty relation, and it is refused rather than read as "holds".
func TestARowExampleWithoutRowsOrTheEmptyMarkerIsRefused(t *testing.T) {
	contract := threeValuedRowContract()
	contract.RowCases = []semantic.OperatorRowCase{{Answer: semantic.OperatorRowSatisfied}}
	err := validationeval.CheckImplementationMatchesContract(contract, resolvedEveryAtMost(t, contract))
	if err == nil {
		t.Fatal("a row example with no rows and no empty marker was accepted")
	}
}
