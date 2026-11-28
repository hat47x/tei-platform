package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A presence input read over every row of a relation states its meaning in a
// row-example table (decision sections 8.6 and 8.1). The cells write whether a
// row holds a value rather than which value it holds: `present`, `absent` and
// `not-fetched`.
//
// The expected answers below are written from that table and from the shipped
// implementation's definition ("every row of the relation holds a value"), not
// from running the evaluator first.
func TestAPresenceRowExampleIsCheckedAgainstTheImplementation(t *testing.T) {
	contract := presenceRowContract()
	for _, tc := range []struct {
		name    string
		rows    []map[string]string
		answer  semantic.OperatorRowExpect
		wantErr string
	}{
		{
			name:   "every row holds a value",
			rows:   []map[string]string{{"value": "present"}, {"value": "present"}},
			answer: semantic.OperatorRowSatisfied,
		},
		{
			name:   "one row holds no value",
			rows:   []map[string]string{{"value": "present"}, {"value": "absent"}},
			answer: semantic.OperatorRowNotSatisfied,
		},
		{
			name:   "a row nobody read leaves the answer undecided",
			rows:   []map[string]string{{"value": "present"}, {"value": "not-fetched"}},
			answer: semantic.OperatorRowUndecided,
		},
		{
			// The implementation answers not-satisfied, so a table that states
			// satisfied is not this operator's meaning.
			name:    "an answer the implementation contradicts",
			rows:    []map[string]string{{"value": "present"}, {"value": "absent"}},
			answer:  semantic.OperatorRowSatisfied,
			wantErr: "TEI_OPERATOR_IMPLEMENTATION_MISMATCH",
		},
		{
			// The empty cell already means "holds no value" for every other
			// input, so a presence cell writes the word instead.
			name:    "an empty cell",
			rows:    []map[string]string{{"value": "present"}, {"value": ""}},
			answer:  semantic.OperatorRowSatisfied,
			wantErr: "a presence input writes present or absent",
		},
		{
			// A number is a value, not whether a value is there.
			name:    "a number cell",
			rows:    []map[string]string{{"value": "100"}},
			answer:  semantic.OperatorRowSatisfied,
			wantErr: "a presence input writes present or absent",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			example := semantic.OperatorRowCase{Answer: tc.answer, Rows: tc.rows}
			withExample := contract
			withExample.RowCases = []semantic.OperatorRowCase{example}
			implementation := presenceRowImplementation()
			err := validationeval.CheckImplementationMatchesContract(withExample, implementation)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("the example was refused: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("the example was accepted although it states %s", tc.answer)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("the refusal does not name %q: %v", tc.wantErr, err)
			}
		})
	}
}

// An example that states the empty relation is read: no row is a
// counterexample, so the shipped implementation answers satisfied whatever
// `refutes on some rows` says (section 8.2).
func TestAPresenceRowExampleOverNoRowsIsSatisfied(t *testing.T) {
	contract := presenceRowContract()
	contract.RowCases = []semantic.OperatorRowCase{
		{Answer: semantic.OperatorRowSatisfied, Rows: []map[string]string{}},
	}
	if err := validationeval.CheckImplementationMatchesContract(contract, presenceRowImplementation()); err != nil {
		t.Fatalf("the empty-relation example was refused: %v", err)
	}
}

func presenceRowContract() semantic.Operator {
	return semantic.Operator{
		ID: "sales.every_has", Symbol: "every_has",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
	}
}

func presenceRowImplementation() validationeval.OperatorImplementation {
	for _, named := range validationeval.BuiltinImplementations() {
		if named.ID != "tei.presence.every-present" {
			continue
		}
		return validationeval.OperatorImplementation{
			Operator:     "sales.every_has",
			Input:        semantic.OperatorInputPresence,
			PresenceRows: named.PresenceRows,
		}
	}
	return validationeval.OperatorImplementation{}
}
