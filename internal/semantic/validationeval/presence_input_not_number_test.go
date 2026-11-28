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
// semantic.OperatorInputPresence, and the evaluator reads it through the reader
// that says whether a value is there.
//
// The two questions stay apart. Measured before the reader was connected, a
// presence contract was handed the attribute's number 7 and answered satisfied —
// a different question answered by accident (#4359). This pins that a presence
// rule is decided from presence only: the numeric implementation is never
// asked, and a record with no value is not-satisfied rather than read as a
// number.
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
	// The numeric implementation is not the implementation of a presence
	// contract, so the mismatch is refused and the number is never asked about.
	if err == nil {
		t.Fatalf("a presence input was decided as %q from a number", decision.Result)
	}
	if !contains(err.Error(), "whether a value is there") {
		t.Fatalf("the refusal does not name the question: %v", err)
	}
	if len(asked) != 0 {
		t.Fatalf("the implementation was asked about %d values for a presence input", len(asked))
	}
}

// The shipped presence implementation decides from section 8.5's two answers:
// an attribute that holds a value is satisfied and one that holds none is not.
func TestTheShippedPresenceImplementationDecidesTheStates(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.present", Symbol: "present",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
		},
	}
	var implementations []validationeval.OperatorImplementation
	for _, named := range validationeval.BuiltinImplementations() {
		if named.ID != "tei.presence.present" {
			continue
		}
		var called int
		decided, err := validationeval.ResolveImplementations(
			[]validationeval.ImplementationSelection{{Operator: "sales.present", Implementation: named.ID}},
			[]validationeval.NamedImplementation{named},
			func(validationeval.ImplementationCall) { called++ },
		)
		if err != nil {
			t.Fatalf("the shipped presence implementation did not resolve: %v", err)
		}
		if called != 0 {
			t.Fatalf("resolution called the implementation %d times; being selected and being asked are different", called)
		}
		implementations = decided
	}
	if len(implementations) != 1 {
		t.Fatalf("implementations=%d, want the one shipped presence implementation", len(implementations))
	}
	rule := semantic.Validation{
		ID: "sales.order_line.item_code_present", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.item_code", Operator: "sales.present",
	}
	for _, tc := range []struct {
		name  string
		state validationeval.ValueState
		want  validationeval.Result
	}{
		{"a present attribute", validationeval.ValuePresent, validationeval.ResultSatisfied},
		{"a record that holds no value", validationeval.ValueNull, validationeval.ResultNotSatisfied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := validationeval.Decide(rule, oneStateSource{state: tc.state},
				validationeval.Operators{
					Declared:        []semantic.Operator{contract},
					Implementations: implementations,
				})
			if err != nil {
				t.Fatalf("the presence rule was refused: %v", err)
			}
			if decision.Result != tc.want {
				t.Fatalf("result=%q, want %q", decision.Result, tc.want)
			}
		})
	}
	// An attribute nobody read is not handed to the implementation: section 5
	// puts that handling in the evaluator, so the rule stays undecided with
	// its own reason rather than being answered as "no value".
	var presenceAsked int
	for _, impl := range implementations {
		asked := impl.Presence
		impl.Presence = func(values map[string]bool) (bool, error) {
			presenceAsked++
			return asked(values)
		}
		_ = impl
	}
	decision, err := validationeval.Decide(rule, oneStateSource{state: validationeval.ValueNotFetched},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: implementations,
		})
	if err != nil {
		t.Fatalf("an attribute nobody read was refused: %v", err)
	}
	if decision.Undecided == nil {
		t.Fatalf("an attribute nobody read was decided %q", decision.Result)
	}
	if decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("reason=%q, want value-not-fetched", decision.Undecided.Reason)
	}
}

// oneStateSource reports one presence state for every attribute.
type oneStateSource struct {
	state validationeval.ValueState
}

func (s oneStateSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (s oneStateSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s oneStateSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s oneStateSource) ReadPresence(core.CanonicalID) (validationeval.PresenceReading, error) {
	return validationeval.PresenceReading{State: s.state}, nil
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
