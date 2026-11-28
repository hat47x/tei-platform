package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A presence rule that asks about more than one attribute reads more than one
// value. Having several answers is not the same as knowing they came from one
// read (decision section 14.2 B-10), so the presence path compares the read
// attempts before the implementation is asked, exactly as the value path does.
//
// Measured before this: a presence rule reading two attributes from two
// attempts was decided, while the same shape on the value path was refused.

// attemptSource states which read attempt each attribute belongs to.
type attemptSource struct {
	attempts map[core.CanonicalID]string
	// anonymous, when set, means the reader cannot state an attempt at all.
	anonymous bool
}

func (s attemptSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (attemptSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s attemptSource) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	if s.anonymous {
		return "", false
	}
	attempt, ok := s.attempts[attribute]
	return attempt, ok
}

func (attemptSource) ReadPresence(core.CanonicalID) (validationeval.PresenceReading, error) {
	return validationeval.PresenceReading{State: validationeval.ValuePresent}, nil
}

// presenceTwoAttributes is the contract and rule this file decides: two
// presence inputs, both read from attributes.
func presenceTwoAttributes() (semantic.Operator, semantic.Validation, validationeval.OperatorImplementation) {
	contract := semantic.Operator{
		ID: "sales.both_stated", Symbol: "both_stated",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
			{Name: "other", Type: semantic.OperatorInputPresence},
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.both_stated", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.both_stated",
		Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: "sales.order_line.unit_price"}},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.both_stated",
		Input:    semantic.OperatorInputPresence,
		Presence: func(present map[string]bool) (bool, error) {
			return present["value"] && present["other"], nil
		},
	}
	return contract, rule, implementation
}

// Two attributes read in two different attempts are refused, not decided.
func TestPresenceRuleMixingTwoReadAttemptsIsRefused(t *testing.T) {
	contract, rule, implementation := presenceTwoAttributes()
	_, err := validationeval.Decide(rule, attemptSource{attempts: map[core.CanonicalID]string{
		"sales.order_line.quantity":   "read-a",
		"sales.order_line.unit_price": "read-b",
	}}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		t.Fatal("a presence rule mixing two read attempts was decided")
	}
	for _, want := range []string{"other", "value", "cannot mix read attempts"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal does not name %q: %v", want, err)
		}
	}
}

// One attribute read, plus one whose attempt the reader cannot state, is
// undecided rather than assumed consistent.
func TestPresenceRuleWithAnUnknownReadAttemptIsUndecided(t *testing.T) {
	contract, rule, implementation := presenceTwoAttributes()
	decision, err := validationeval.Decide(rule, attemptSource{anonymous: true}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("the rule was refused rather than left undecided: %v", err)
	}
	if decision.Result != "" {
		t.Fatalf("result=%q, want no result", decision.Result)
	}
	if decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedReadAttemptUnknown {
		t.Fatalf("undecided=%+v, want read-attempt-unknown", decision.Undecided)
	}
}

// The same two attributes read in one attempt are decided: the check is about
// knowing they are one read, not about the number of inputs.
func TestPresenceRuleInOneReadAttemptIsDecided(t *testing.T) {
	contract, rule, implementation := presenceTwoAttributes()
	decision, err := validationeval.Decide(rule, attemptSource{attempts: map[core.CanonicalID]string{
		"sales.order_line.quantity":   "read-a",
		"sales.order_line.unit_price": "read-a",
	}}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("a one-attempt presence rule was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied", decision.Result)
	}
}
