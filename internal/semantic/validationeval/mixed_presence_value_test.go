package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A contract that declares both a presence input and a value input is not a
// signature this evaluator can answer: the presence path asks the reader
// whether EVERY input holds a value, so a `number` input would be answered
// "holds a value" with its value never read.
//
// Measured before this refusal: the mixed contract passed the shape check and a
// number input beside a presence input was decided as presence. The expected
// behaviour is a refusal that names the input that cannot share the signature.
func TestAMixedPresenceAndValueContractIsRefused(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.target_at_least", Symbol: "target_at_least",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "target", Type: semantic.OperatorInputPresence},
			{Name: "limit", Type: semantic.OperatorInputNumber},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.target_at_least",
		Input:    semantic.OperatorInputPresence,
		Presence: func(map[string]bool) (bool, error) { return true, nil },
	}
	err := validationeval.CheckImplementationMatchesContract(contract, implementation)
	if err == nil {
		t.Fatal("a contract mixing presence with a value input was accepted")
	}
	// The refusal names the input the author has to change, and the type it is
	// declared with, so the message points at the declaration rather than at
	// the evaluator.
	for _, want := range []string{`"limit"`, "number", "presence"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal does not name %q: %v", want, err)
		}
	}
}

// The same refusal comes from the rule path, not only from the declarative
// check: a selected implementation of the wrong shape never decides the rule.
func TestAMixedPresenceAndValueRuleIsNotDecided(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.target_at_least", Symbol: "target_at_least",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "target", Type: semantic.OperatorInputPresence},
			{Name: "limit", Type: semantic.OperatorInputNumber},
		},
	}
	asked := false
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.target_at_least",
		Input:    semantic.OperatorInputPresence,
		Presence: func(map[string]bool) (bool, error) {
			asked = true
			return true, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.target_stated", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.unit_price", Operator: "sales.target_at_least",
		Inputs: []semantic.ValidationInputAssignment{{Name: "limit", Value: "5"}},
	}
	decision, err := validationeval.Decide(rule, refusingTypeSource{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		if decision.Result != "" {
			t.Fatalf("the mixed rule was decided as %q", decision.Result)
		}
		t.Fatal("the mixed rule was not refused")
	}
	if asked {
		t.Fatal("the implementation was asked about a signature it cannot answer")
	}
}
