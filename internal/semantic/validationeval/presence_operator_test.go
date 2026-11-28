package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A presence input is answered by the reader that knows whether the record
// holds a value, and the value an implementation receives says which of the two
// answers it is. The expectations are written from what presence asks, not read
// back from the reader.
//
// The distinction the reader keeps is between "the record certainly holds no
// value" and "nobody read whether it holds one": the first is the answer
// "absent", the second is undecided. Reading the second as the first would
// answer a rule about a record nobody looked at.

// presenceAnswerSource reads one attribute's presence with a chosen state.
type presenceAnswerSource struct {
	state validationeval.ValueState
}

func (s presenceAnswerSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (presenceAnswerSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (presenceAnswerSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s presenceAnswerSource) ReadPresence(core.CanonicalID) (validationeval.PresenceReading, error) {
	return validationeval.PresenceReading{State: s.state}, nil
}

// A presence rule is decided from what the reader reports, and the
// implementation receives the answer rather than the attribute's value.
func TestAPresenceRuleIsDecidedFromWhatTheReaderReports(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.holds_a_value", Symbol: "holds_a_value",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
		},
	}
	var seen *bool
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.holds_a_value",
		Input:    semantic.OperatorInputPresence,
		Values: func(values map[string]semantic.OperatorValue) (bool, error) {
			presence := values["value"].Presence
			seen = presence
			return presence != nil && *presence, nil
		},
	}
	for _, tc := range []struct {
		name  string
		state validationeval.ValueState
		want  validationeval.Result
		held  bool
	}{
		{"the record holds a value", validationeval.ValuePresent, validationeval.ResultSatisfied, true},
		{"the record holds none", validationeval.ValueNull, validationeval.ResultNotSatisfied, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := semantic.Validation{
				ID: "sales.order_line.quantity_stated", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.quantity", Operator: "sales.holds_a_value",
			}
			decision, err := validationeval.Decide(rule, presenceAnswerSource{state: tc.state},
				validationeval.Operators{
					Declared:        []semantic.Operator{contract},
					Implementations: []validationeval.OperatorImplementation{implementation},
				})
			if err != nil {
				t.Fatalf("the presence rule was refused: %v", err)
			}
			if decision.Result != tc.want {
				t.Fatalf("result=%q, want %q", decision.Result, tc.want)
			}
			if seen == nil || *seen != tc.held {
				t.Fatalf("the implementation received %v, want %v", seen, tc.held)
			}
		})
	}
}

// A record nobody fetched is undecided, not "holds no value": nobody read
// whether it holds one, and the two are different answers.
func TestAPresenceRuleOverAnUnreadRecordIsUndecided(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.holds_a_value", Symbol: "holds_a_value",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.holds_a_value",
		Input:    semantic.OperatorInputPresence,
		Values: func(map[string]semantic.OperatorValue) (bool, error) {
			t.Fatal("the implementation was asked about a record nobody read")
			return false, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_stated", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.holds_a_value",
	}
	decision, err := validationeval.Decide(rule, presenceAnswerSource{state: validationeval.ValueNotFetched},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err != nil {
		t.Fatalf("the rule was refused: %v", err)
	}
	if decision.Result != "" {
		t.Fatalf("result=%q, want no result", decision.Result)
	}
	if decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("undecided=%+v, want the not-fetched reason", decision.Undecided)
	}
}

// A numeric implementation answers a value, which is a different question than
// presence asks, so the rule is refused and the refusal names the input.
func TestAPresenceRuleIsRefusedByANumericImplementation(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.holds_a_value", Symbol: "holds_a_value",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "stated", Type: semantic.OperatorInputPresence},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.holds_a_value",
		Input:    semantic.OperatorInputPresence,
		Number:   func(*big.Rat) (bool, error) { return true, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_stated", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.holds_a_value",
	}
	_, err := validationeval.Decide(rule, presenceAnswerSource{state: validationeval.ValuePresent},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err == nil {
		t.Fatal("a numeric implementation decided a presence rule")
	}
	if !strings.Contains(err.Error(), `"stated"`) || !strings.Contains(err.Error(), "presence") {
		t.Fatalf("the refusal does not name the input and the type: %v", err)
	}
}

// A reader that cannot read presence refuses the rule rather than answering it
// from a value.
func TestAPresenceRuleIsRefusedByAReaderThatCannotReadIt(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.holds_a_value", Symbol: "holds_a_value",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.holds_a_value",
		Input:    semantic.OperatorInputPresence,
		Values:   func(map[string]semantic.OperatorValue) (bool, error) { return true, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_stated", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.holds_a_value",
	}
	_, err := validationeval.Decide(rule, refusingTypeSource{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		t.Fatal("a reader without a presence writer decided a presence rule")
	}
	if !strings.Contains(err.Error(), "cannot read whether a value is there") {
		t.Fatalf("the refusal does not say what the reader lacks: %v", err)
	}
}

// A presence input that reads every row of a relation is refused: this reader
// reads whether ONE record holds a value, and a column of presence has no
// reader on this path.
func TestAPresenceColumnIsRefused(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.holds_a_value", Symbol: "holds_a_value",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.holds_a_value",
		Input:    semantic.OperatorInputPresence,
		Values:   func(map[string]semantic.OperatorValue) (bool, error) { return true, nil },
	}
	rule := semantic.Validation{
		ID: "sales.order.lines.every_quantity_stated", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.holds_a_value",
		Relation: "sales.order.lines",
	}
	_, err := validationeval.Decide(rule, presenceAnswerSource{state: validationeval.ValuePresent},
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err == nil {
		t.Fatal("a column of presence was decided from one record")
	}
	// The refusal comes from the contract check, which runs before any rule is
	// read: a row-scoped contract needs a row-scoped implementation, and a
	// presence column needs a reader this path does not have. Either way the
	// rule is refused rather than answered from one record of the relation.
	if !strings.Contains(err.Error(), "reads every row") {
		t.Fatalf("the refusal does not say what is missing: %v", err)
	}
}
