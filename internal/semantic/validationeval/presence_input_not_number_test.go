package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// An operator may declare `takes value: presence`: whether the attribute holds a
// value, not its value. The authoring surface reads the word and carries it into
// semantic.OperatorInputPresence, but the evaluator's one operator path reads
// every input through ReadNumber. Measured before the fix, a presence contract
// was handed the attribute's number 7 and answered satisfied — a different
// question answered by accident (#4359).
//
// A rule that needs a presence input is now refused, because the evaluator has
// no presence reader on that path yet. It is not answered from a number.
func TestAPresenceInputIsNotAnsweredFromANumber(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.present", Symbol: "present",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
		},
	}
	var asked []*big.Rat
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.present",
		Input:    semantic.OperatorInputPresence,
		Number: func(value *big.Rat) (bool, error) {
			asked = append(asked, value)
			return true, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_present", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.present",
	}

	decision, err := validationeval.Decide(rule, oneNumberSource{value: big.NewRat(7, 1)},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err == nil {
		t.Fatalf("a presence input was decided as %q from a number; it has no reader on this path", decision.Result)
	}
	// The refusal names the input and its type, so the author learns which
	// declaration is not readable yet.
	for _, want := range []string{`"value"`, "presence", string(rule.ID)} {
		if !contains(err.Error(), want) {
			t.Fatalf("the refusal does not name %q: %v", want, err)
		}
	}
	if len(asked) != 0 {
		t.Fatalf("the implementation was asked about %d values for a presence input", len(asked))
	}
}

// A numeric operator is unaffected: this refuses the type it cannot read, not
// operator rules.
func TestANumericOperatorIsStillDecided(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.non_negative", Symbol: "non_negative",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.non_negative",
		Input:    semantic.OperatorInputNumber,
		Number:   func(value *big.Rat) (bool, error) { return value.Sign() >= 0, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_non_negative", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.non_negative",
	}
	decision, err := validationeval.Decide(rule, oneNumberSource{value: big.NewRat(2, 1)},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err != nil {
		t.Fatalf("a numeric operator was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied for 2", decision.Result)
	}
}

// oneNumberSource reports one present number for every attribute and states one
// attempt. It also answers presence, so the refusal is about the declaration,
// not about a reader that cannot answer.
type oneNumberSource struct {
	value *big.Rat
}

func (s oneNumberSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: s.value}, nil
}

func (s oneNumberSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s oneNumberSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s oneNumberSource) ReadPresence(core.CanonicalID) (validationeval.PresenceReading, error) {
	return validationeval.PresenceReading{State: validationeval.ValuePresent}, nil
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
