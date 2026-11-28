package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// An operator may declare `takes value: presence`: whether the attribute holds a
// value, not which value it holds. Measured before the reader existed, a presence
// contract was handed the attribute's number 7 and answered satisfied 窶・a
// different question answered by accident (#4359). It was then refused outright
// for a while; now it has its own reader and its own decision function.
//
// What these tests write from the meaning, not from the implementation:
//
//   - a record that holds a value and one that holds none are both ANSWERS, and
//     they are different answers;
//   - a value nobody read is not an answer, so the rule is undecided with that
//     reason rather than telling the implementation that nothing is there;
//   - a presence implementation and a value implementation are not
//     interchangeable, in either direction.

// presentSource reports a chosen state for every attribute.
type presentSource struct {
	state validationeval.ValueState
}

func (s presentSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: s.state, Value: s.numberOrNil()}, nil
}

func (s presentSource) numberOrNil() *big.Rat {
	if s.state == validationeval.ValuePresent {
		return big.NewRat(7, 1)
	}
	return nil
}

func (s presentSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s presentSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s presentSource) ReadPresence(core.CanonicalID) (validationeval.PresenceReading, error) {
	return validationeval.PresenceReading{State: s.state}, nil
}

func presenceContract(inputs ...semantic.OperatorInputDeclaration) semantic.Operator {
	if len(inputs) == 0 {
		inputs = []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputPresence}}
	}
	return semantic.Operator{
		ID: "sales.present", Symbol: "present",
		Input:  semantic.OperatorInputPresence,
		Inputs: inputs,
	}
}

// asksWhetherPresent records what the implementation was told, so the test can
// assert the answer the implementation received and not only the result.
func asksWhetherPresent(t *testing.T, seen *map[string]bool, answer func(bool) bool) validationeval.OperatorImplementation {
	t.Helper()
	return validationeval.OperatorImplementation{
		Operator: "sales.present",
		Input:    semantic.OperatorInputPresence,
		Presence: func(present map[string]bool) (bool, error) {
			*seen = present
			return answer(present["value"]), nil
		},
	}
}

func decidePresence(t *testing.T, rule semantic.Validation, source validationeval.StatefulSource, implementation validationeval.OperatorImplementation, contract semantic.Operator) (validationeval.Decision, error) {
	t.Helper()
	return validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
}

func presenceRule() semantic.Validation {
	return semantic.Validation{
		ID: "sales.order_line.reason_present", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.reason", Operator: "sales.present",
	}
}

// A record that holds a value is told so, and a rule that requires a value is
// satisfied.
func TestARecordThatHoldsAValueIsToldSo(t *testing.T) {
	var seen map[string]bool
	decision, err := decidePresence(t, presenceRule(), presentSource{state: validationeval.ValuePresent},
		asksWhetherPresent(t, &seen, func(present bool) bool { return present }), presenceContract())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("decision=%+v, want satisfied", decision)
	}
	if present, told := seen["value"]; !told || !present {
		t.Fatalf("the implementation was told present=%v (told=%v), want it told the value is there", present, told)
	}
}

// A record that holds no value is an answer too: the implementation is told
// "nothing is there", and a rule that requires a value is refuted. It is not
// left undecided, because nothing about the read is in doubt.
func TestARecordThatHoldsNoValueIsToldSo(t *testing.T) {
	var seen map[string]bool
	decision, err := decidePresence(t, presenceRule(), presentSource{state: validationeval.ValueNull},
		asksWhetherPresent(t, &seen, func(present bool) bool { return present }), presenceContract())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("decision=%+v, want not-satisfied", decision)
	}
	if present, told := seen["value"]; !told || present {
		t.Fatalf("the implementation was told present=%v (told=%v), want it told nothing is there", present, told)
	}
}

// A value nobody read is not an answer: the rule is undecided with the reader's
// own reason, and the implementation is not asked at all.
func TestAValueNobodyReadIsNotAnswerThatNothingIsThere(t *testing.T) {
	var seen map[string]bool
	decision, err := decidePresence(t, presenceRule(), presentSource{state: validationeval.ValueNotFetched},
		asksWhetherPresent(t, &seen, func(bool) bool { return true }), presenceContract())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("decision=%+v, want undecided value-not-fetched", decision)
	}
	if seen != nil {
		t.Fatalf("the implementation was asked %v although nobody read the attribute", seen)
	}
}

// A presence implementation is not a value implementation: an operator that
// declares presence cannot be decided by one that reads values. This is the
// mistake #4359 measured, refused at the selection rather than answered from
// the attribute's number.
func TestAnOperatorThatAsksPresenceIsNotDecidedByAValueImplementation(t *testing.T) {
	var asked []*big.Rat
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.present",
		Input:    semantic.OperatorInputPresence,
		Number: func(value *big.Rat) (bool, error) {
			asked = append(asked, value)
			return true, nil
		},
	}
	_, err := decidePresence(t, presenceRule(), presentSource{state: validationeval.ValuePresent}, implementation, presenceContract())
	if err == nil {
		t.Fatal("a presence input was decided by an implementation that reads values")
	}
	// The refusal names the input and the rule it could not decide, so the
	// author learns which declaration to change.
	for _, want := range []string{`"value"`, "presence"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal does not name %q: %v", want, err)
		}
	}
	if len(asked) != 0 {
		t.Fatalf("the implementation was asked about %d values for a presence input", len(asked))
	}
}

// The other direction is refused too: an operator that reads no presence input
// cannot be decided by an implementation that answers presence.
func TestAValueOperatorIsNotDecidedByAPresenceImplementation(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.non_negative", Symbol: "non_negative",
		Input:  semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputNumber}},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.non_negative",
		Input:    semantic.OperatorInputNumber,
		Presence: func(map[string]bool) (bool, error) { return true, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_non_negative", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.non_negative",
	}
	_, err := decidePresence(t, rule, presentSource{state: validationeval.ValuePresent}, implementation, contract)
	if err == nil {
		t.Fatal("a value operator was decided by an implementation that answers presence")
	}
	if !strings.Contains(err.Error(), "presence") {
		t.Fatalf("the refusal does not name presence: %v", err)
	}
}

// A row-scoped presence operator is refused rather than decided from the rows
// of one record: a reader that answers presence for every row of a relation
// does not exist yet, and answering from one row would answer a question about
// the relation from one record of it.
func TestAPresenceInputOverEveryRowIsRefused(t *testing.T) {
	contract := presenceContract(semantic.OperatorInputDeclaration{
		Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true,
	})
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.present",
		Input:    semantic.OperatorInputPresence,
		Presence: func(map[string]bool) (bool, error) { return true, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order.lines.reason_present", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.reason", Operator: "sales.present",
		Relation: "sales.order.lines",
		Inputs:   []semantic.ValidationInputAssignment{{Name: "value", Value: "sales.order_line.reason"}},
	}
	_, err := decidePresence(t, rule, presentSource{state: validationeval.ValuePresent}, implementation, contract)
	if err == nil {
		t.Fatal("a presence input over every row was decided from one record")
	}
}

// A presence input's examples are the two words section 8.3 fixes, and an
// implementation that contradicts one of them is not an implementation of that
// operator.
func TestAPresenceExampleIsCheckedAgainstTheImplementation(t *testing.T) {
	requiresPresence := func(present map[string]bool) (bool, error) { return present["value"], nil }
	contradicts := func(map[string]bool) (bool, error) { return false, nil }
	contract := presenceContract()
	contract.Cases = []semantic.OperatorCase{
		{Value: semantic.OperatorCasePresencePresent, Satisfied: true},
		{Value: semantic.OperatorCasePresenceAbsent, Satisfied: false},
	}
	sample := []semantic.OperatorCase{
		{Value: semantic.OperatorCasePresencePresent, Satisfied: false},
	}
	for _, tc := range []struct {
		name           string
		cases          []semantic.OperatorCase
		implementation validationeval.OperatorImplementation
		wantErr        bool
	}{
		{
			name:  "the words agree with the meaning",
			cases: contract.Cases,
			implementation: validationeval.OperatorImplementation{
				Operator: "sales.present", Input: semantic.OperatorInputPresence, Presence: requiresPresence,
			},
		},
		{
			name:  "an example the implementation contradicts",
			cases: sample,
			implementation: validationeval.OperatorImplementation{
				Operator: "sales.present", Input: semantic.OperatorInputPresence, Presence: requiresPresence,
			},
			wantErr: true,
		},
		{
			name:  "an example states a decimal for a presence input",
			cases: []semantic.OperatorCase{{Value: "7", Satisfied: true}},
			implementation: validationeval.OperatorImplementation{
				Operator: "sales.present", Input: semantic.OperatorInputPresence, Presence: contradicts,
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			subject := contract
			subject.Cases = tc.cases
			err := validationeval.CheckImplementationMatchesContract(subject, tc.implementation)
			if tc.wantErr && err == nil {
				t.Fatal("the example was accepted")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("the example was refused: %v", err)
			}
		})
	}
}
