package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A presence input asks whether the attribute holds a value, not what its value
// is. The answer travels in OperatorValue.Presence, so a number is never read as
// the answer to that question (#4359 measured a presence contract answered
// satisfied from the number 7).
//
// The expectations are drawn from what the declaration says the input reads,
// not from the code.
func TestAPresenceInputIsReadAsPresence(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.holds", Symbol: "holds",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
		},
	}
	seen := make(chan semantic.OperatorValue, 1)
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.holds",
		Input:    semantic.OperatorInputPresence,
		Values: func(values map[string]semantic.OperatorValue) (bool, error) {
			seen <- values["value"]
			return true, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.code_present", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.item_code", Operator: "sales.holds",
	}
	// A record that holds no value answers the presence question with "no". It
	// is a settled answer, so the rule is decided rather than left undecided.
	decision, err := validationeval.Decide(rule, presenceOnlySource{state: validationeval.ValueNull},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err != nil {
		t.Fatalf("a presence input was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied", decision.Result)
	}
	select {
	case got := <-seen:
		if got.Kind() != semantic.OperatorInputPresence {
			t.Fatalf("the implementation received %s, want presence", got.Kind())
		}
		if got.Presence == nil {
			t.Fatal("the presence answer was not carried")
		}
		if *got.Presence {
			t.Fatalf("a record that holds no value was read as holding one")
		}
	default:
		t.Fatal("the implementation was not asked about the presence input")
	}
}

// A presence input nobody read leaves the rule undecided: nobody looked, which
// is not the same as looking and finding no value.
func TestAPresenceInputNobodyReadIsUndecided(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.holds", Symbol: "holds",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.holds",
		Input:    semantic.OperatorInputPresence,
		Values:   func(map[string]semantic.OperatorValue) (bool, error) { return true, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.code_present", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.item_code", Operator: "sales.holds",
	}
	decision, err := validationeval.Decide(rule, presenceOnlySource{state: validationeval.ValueNotFetched},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err != nil {
		t.Fatalf("a presence input was refused: %v", err)
	}
	if decision.Undecided == nil {
		t.Fatalf("a value nobody read was decided as %q", decision.Result)
	}
	if decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("reason=%q, want value-not-fetched", decision.Undecided.Reason)
	}
}

// A reader that cannot read presence is refused by name rather than answered
// from a number, which is the bug this field exists to prevent.
func TestAReaderWithoutPresenceIsRefused(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.holds", Symbol: "holds",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.holds",
		Input:    semantic.OperatorInputPresence,
		Values:   func(map[string]semantic.OperatorValue) (bool, error) { return true, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.code_present", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.item_code", Operator: "sales.holds",
	}
	_, err := validationeval.Decide(rule, numbersOnlySource{},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err == nil {
		t.Fatal("a presence input was answered by a reader that reads numbers")
	}
	if !strings.Contains(err.Error(), "presence") {
		t.Fatalf("the refusal does not name presence: %v", err)
	}
}

// presenceOnlySource reports the one presence state it was given and states one
// read attempt.
type presenceOnlySource struct {
	state validationeval.ValueState
}

func (s presenceOnlySource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(7, 1)}, nil
}

func (s presenceOnlySource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s presenceOnlySource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s presenceOnlySource) ReadPresence(core.CanonicalID) (validationeval.PresenceReading, error) {
	return validationeval.PresenceReading{State: s.state}, nil
}

// numbersOnlySource reads numbers and refuses to answer presence.
type numbersOnlySource struct{}

func (numbersOnlySource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(7, 1)}, nil
}

func (numbersOnlySource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (numbersOnlySource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }
