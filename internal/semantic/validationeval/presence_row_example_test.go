package validationeval_test

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A presence input declared `of each row` is stated by a table of rows whose
// cells say whether a row holds a value: `present`, `absent`, or the shared word
// for "not read" (section 8.6). An implementation that answers one of those
// examples differently is not an implementation of the operator, so it is not
// asked about real data (section 8.1).
//
// The expectations are read from the operator's own meaning, not from the code.
func TestAPresenceRowExampleIsCheckedAgainstTheImplementation(t *testing.T) {
	contract := semantic.Operator{
		ID:    "sales.all_hold",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
		RowCases: []semantic.OperatorRowCase{{
			Rows: []map[string]string{
				{"value": semantic.OperatorCasePresencePresent},
				{"value": semantic.OperatorCasePresencePresent},
			},
			Answer: semantic.OperatorRowSatisfied,
		}},
	}
	// The implementation answers as the operator means: every row holding a value
	// satisfies it.
	agreeing := validationeval.OperatorImplementation{
		Operator: "sales.all_hold",
		Input:    semantic.OperatorInputPresence,
		PresenceRows: func(rows map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
			for _, row := range rows["value"] {
				if row.State != validationeval.ValuePresent {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
	if err := validationeval.CheckImplementationMatchesContract(contract, agreeing); err != nil {
		t.Fatalf("an implementation that agrees with the operator's row example was refused: %v", err)
	}

	// An implementation that reads the example differently is not one of this
	// operator, and the refusal names the operator.
	disagreeing := validationeval.OperatorImplementation{
		Operator: "sales.all_hold",
		Input:    semantic.OperatorInputPresence,
		PresenceRows: func(map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
			return validationeval.TriNo, nil
		},
	}
	err := validationeval.CheckImplementationMatchesContract(contract, disagreeing)
	if err == nil {
		t.Fatal("an implementation that contradicts the operator's row example was accepted")
	}
}

// A row example whose cell states a value of another type is refused rather than
// read as one of the three words.
func TestAPresenceRowExampleRefusesACellOfAnotherType(t *testing.T) {
	contract := semantic.Operator{
		ID:    "sales.all_hold",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
		},
		RowCases: []semantic.OperatorRowCase{{
			Rows:   []map[string]string{{"value": "123"}},
			Answer: semantic.OperatorRowSatisfied,
		}},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.all_hold",
		Input:    semantic.OperatorInputPresence,
		PresenceRows: func(map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
			return validationeval.TriYes, nil
		},
	}
	if err := validationeval.CheckImplementationMatchesContract(contract, implementation); err == nil {
		t.Fatal("a number was accepted as a presence row cell")
	}
}
