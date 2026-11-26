package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A named input is read in the type its declaration states, the same way the
// rule's subject is. Before this the operator path read an assigned attribute as
// a number and refused every other type by name, so a text bound could not be
// read at all: a rule `... is same-as name: sales.order_line.item_code` was
// refused with "the text input has no reader on this path yet" even though the
// reader reads text (decision section 8.4).
//
// The expectations are written from that meaning: the implementation receives
// the value in the field its declaration names, and a value nobody read is not
// the zero of that type.

// assignedTypedSource reads a different value per attribute, so one test can
// hold a text on the subject and another on the named bound, and state that
// either of them holds no value.
type assignedTypedSource struct {
	texts  map[string]validationeval.TextReading
	truths map[string]validationeval.TruthReading
}

func (s assignedTypedSource) ReadText(attribute core.CanonicalID) (validationeval.TextReading, error) {
	reading, ok := s.texts[string(attribute)]
	if !ok {
		return validationeval.TextReading{State: validationeval.ValueNotFetched}, nil
	}
	return reading, nil
}

func (s assignedTypedSource) ReadTruth(attribute core.CanonicalID) (validationeval.TruthReading, error) {
	reading, ok := s.truths[string(attribute)]
	if !ok {
		return validationeval.TruthReading{State: validationeval.ValueNotFetched}, nil
	}
	return reading, nil
}

func (s assignedTypedSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (s assignedTypedSource) ReadRelationKeys(_, _ core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

// ReadAttempt states that every attribute this source reads belongs to one read,
// so the two-attempt check has nothing to refuse.
func (s assignedTypedSource) ReadAttempt(core.CanonicalID) (string, bool) {
	return "read-1", true
}

// sameAsText decides "the value holds exactly the other text". It is a plain
// reading of that meaning, written here so a wrong evaluator is caught even if
// the shipped one were wrong too.
func sameAsText() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "sales.same_as",
		Input:    semantic.OperatorInputText,
		Values: func(values map[string]semantic.OperatorValue) (bool, error) {
			value, ok := values["value"]
			if !ok || value.Text == nil {
				return false, nil
			}
			other, ok := values["other"]
			if !ok || other.Text == nil {
				return false, nil
			}
			return *value.Text == *other.Text, nil
		},
	}
}

func sameAsTextContract() semantic.Operator {
	return semantic.Operator{
		ID: "sales.same_as", Symbol: "same_as",
		Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText},
			{Name: "other", Type: semantic.OperatorInputText},
		},
	}
}

func sameAsTextRule() semantic.Validation {
	return semantic.Validation{
		ID: "sales.order_line.code_matches_description", Symbol: "code_matches_description",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.item_code",
		Operator:  "sales.same_as",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "other", Value: "sales.order_line.description"},
		},
	}
}

// The named bound is read as a text, and the two texts are compared as texts.
func TestANamedTextInputIsReadAsAText(t *testing.T) {
	source := assignedTypedSource{
		texts: map[string]validationeval.TextReading{
			"sales.order_line.item_code":   {State: validationeval.ValuePresent, Value: "A-100"},
			"sales.order_line.description": {State: validationeval.ValuePresent, Value: "A-100"},
		},
	}
	decision, err := validationeval.Decide(sameAsTextRule(), source, validationeval.Operators{
		Declared:        []semantic.Operator{sameAsTextContract()},
		Implementations: []validationeval.OperatorImplementation{sameAsText()},
	})
	if err != nil {
		t.Fatalf("the named text input was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want satisfied", decision)
	}
}

// A different text refutes it. The comparison is of the texts, and not of a
// number either of them was read as.
func TestANamedTextInputIsComparedAsAText(t *testing.T) {
	source := assignedTypedSource{
		texts: map[string]validationeval.TextReading{
			"sales.order_line.item_code":   {State: validationeval.ValuePresent, Value: "A-100"},
			"sales.order_line.description": {State: validationeval.ValuePresent, Value: "A-101"},
		},
	}
	decision, err := validationeval.Decide(sameAsTextRule(), source, validationeval.Operators{
		Declared:        []semantic.Operator{sameAsTextContract()},
		Implementations: []validationeval.OperatorImplementation{sameAsText()},
	})
	if err != nil {
		t.Fatalf("the named text input was refused: %v", err)
	}
	if decision.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("decision=%+v, want not-satisfied", decision)
	}
}

// A bound the record holds no text for has no text to compare, and the decision
// is undecided with the record's own reason. An empty string is a text, so the
// absence is not read as one.
func TestANamedTextInputThatHoldsNoTextStaysUndecided(t *testing.T) {
	source := assignedTypedSource{
		texts: map[string]validationeval.TextReading{
			"sales.order_line.item_code":   {State: validationeval.ValuePresent, Value: "A-100"},
			"sales.order_line.description": {State: validationeval.ValueNull},
		},
	}
	decision, err := validationeval.Decide(sameAsTextRule(), source, validationeval.Operators{
		Declared:        []semantic.Operator{sameAsTextContract()},
		Implementations: []validationeval.OperatorImplementation{sameAsText()},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("decision=%+v, want undecided value-null", decision)
	}
}

// A bound nobody read is unknown, not empty, and stays undecided with the
// reader's reason.
func TestANamedTextInputNobodyReadStaysUndecided(t *testing.T) {
	source := assignedTypedSource{
		texts: map[string]validationeval.TextReading{
			"sales.order_line.item_code": {State: validationeval.ValuePresent, Value: "A-100"},
		},
	}
	decision, err := validationeval.Decide(sameAsTextRule(), source, validationeval.Operators{
		Declared:        []semantic.Operator{sameAsTextContract()},
		Implementations: []validationeval.OperatorImplementation{sameAsText()},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Result != "" || decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("decision=%+v, want undecided value-not-fetched", decision)
	}
}

// A named truth input is read as a truth, so a rule about whether two flags hold
// is decided from the flags and not from a number either was read as.
func TestANamedTruthInputIsReadAsATruth(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.same_flag", Symbol: "same_flag",
		Input: semantic.OperatorInputTruth,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputTruth},
			{Name: "other", Type: semantic.OperatorInputTruth},
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.flag_matches", Symbol: "flag_matches",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.urgent",
		Operator:  "sales.same_flag",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "other", Value: "sales.order_line.confirmed"},
		},
	}
	source := assignedTypedSource{
		truths: map[string]validationeval.TruthReading{
			"sales.order_line.urgent":    {State: validationeval.ValuePresent, Value: true},
			"sales.order_line.confirmed": {State: validationeval.ValuePresent, Value: false},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.same_flag",
		Input:    semantic.OperatorInputTruth,
		Values: func(values map[string]semantic.OperatorValue) (bool, error) {
			value, ok := values["value"]
			if !ok || value.Truth == nil {
				return false, nil
			}
			other, ok := values["other"]
			if !ok || other.Truth == nil {
				return false, nil
			}
			return *value.Truth == *other.Truth, nil
		},
	}
	decision, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("the named truth input was refused: %v", err)
	}
	if decision.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("decision=%+v, want not-satisfied (true is not false)", decision)
	}
}

// A reader that cannot read the declared type is an error, so the rule is
// refused rather than answered from a number the bound never held.
func TestANamedInputWithoutTheDeclaredTypeIsRefused(t *testing.T) {
	// numbersOnly reads numbers and nothing else, so the text bound cannot be
	// read as a text.
	_, err := validationeval.Decide(sameAsTextRule(), numbersOnly{}, validationeval.Operators{
		Declared:        []semantic.Operator{sameAsTextContract()},
		Implementations: []validationeval.OperatorImplementation{sameAsText()},
	})
	if err == nil {
		t.Fatal("a rule whose bound nobody could read as text was decided")
	}
}

// A named input assigned a constant keeps the type its declaration gives it, so
// a text constant is a text and not the number it would parse as.
func TestANamedTextConstantIsReadAsAText(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.same_as", Symbol: "same_as",
		Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText},
			{Name: "other", Type: semantic.OperatorInputText},
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.code_is_promo", Symbol: "code_is_promo",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.item_code",
		Operator:  "sales.same_as",
		Inputs:    []semantic.ValidationInputAssignment{{Name: "other", Value: `"A-100"`}},
	}
	source := assignedTypedSource{
		texts: map[string]validationeval.TextReading{
			"sales.order_line.item_code": {State: validationeval.ValuePresent, Value: "A-100"},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.same_as",
		Input:    semantic.OperatorInputText,
		Values: func(values map[string]semantic.OperatorValue) (bool, error) {
			value, ok := values["value"]
			if !ok || value.Text == nil {
				return false, nil
			}
			other, ok := values["other"]
			if !ok || other.Text == nil {
				return false, nil
			}
			return *value.Text == *other.Text, nil
		},
	}
	decision, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("a text constant was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("decision=%+v, want satisfied", decision)
	}
}
