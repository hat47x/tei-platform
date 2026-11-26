package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A composition selects an implementation by name, and a rule whose operator
// declares a text or a truth is decided by the one that decides that kind of
// value. Before this, a text or truth rule could be read but not decided: the
// reader carried the value, and nothing in this binary decided one.
//
// The expectations are written from what the operators mean, not read back
// from the code.

// typedSource reads one record: a text attribute and a truth attribute.
type typedSource struct {
	text  string
	truth bool
}

func (s typedSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (typedSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (typedSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s typedSource) ReadText(attribute core.CanonicalID) (validationeval.TextReading, error) {
	if attribute == "sales.order_line.code" {
		return validationeval.TextReading{State: validationeval.ValuePresent, Value: s.text}, nil
	}
	return validationeval.TextReading{State: validationeval.ValueNotFetched}, nil
}

func (s typedSource) ReadTruth(attribute core.CanonicalID) (validationeval.TruthReading, error) {
	if attribute == "sales.order_line.approved" {
		return validationeval.TruthReading{State: validationeval.ValuePresent, Value: s.truth}, nil
	}
	return validationeval.TruthReading{State: validationeval.ValueNotFetched}, nil
}

// A text rule is decided by the implementation the composition selected for
// it, from the text the reader read.
func TestASelectedTextImplementationDecidesTheRule(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.text.same-as", Symbol: "same-as",
		Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText},
			{Name: "other", Type: semantic.OperatorInputText},
		},
	}
	implementation, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.text.same-as", Implementation: "tei.text.same-as"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		t.Fatalf("the text implementation could not be resolved: %v", err)
	}
	for _, tc := range []struct {
		name string
		text string
		want validationeval.Result
	}{
		{"the same text", "A-100", validationeval.ResultSatisfied},
		{"a different text", "B-200", validationeval.ResultNotSatisfied},
		// The empty text is a value of length zero, not the absence of one.
		{"the empty text", "", validationeval.ResultNotSatisfied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := semantic.Validation{
				ID: "sales.order_line.code_matches", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.code", Operator: "sales.text.same-as",
				Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: `"A-100"`}},
			}
			decision, err := validationeval.Decide(rule, typedSource{text: tc.text}, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: implementation,
			})
			if err != nil {
				t.Fatalf("the text rule was refused: %v", err)
			}
			if decision.Result != tc.want {
				t.Fatalf("result=%q, want %q", decision.Result, tc.want)
			}
		})
	}
}

// A truth rule is decided by the implementation the composition selected for
// it, from the truth the reader read.
func TestASelectedTruthImplementationDecidesTheRule(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.truth.is-true", Symbol: "is-true",
		Input: semantic.OperatorInputTruth,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputTruth},
		},
	}
	implementation, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.truth.is-true", Implementation: "tei.truth.is-true"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		t.Fatalf("the truth implementation could not be resolved: %v", err)
	}
	for _, tc := range []struct {
		name  string
		truth bool
		want  validationeval.Result
	}{
		{"true", true, validationeval.ResultSatisfied},
		{"false", false, validationeval.ResultNotSatisfied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := semantic.Validation{
				ID: "sales.order_line.is_approved", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.approved", Operator: "sales.truth.is-true",
			}
			decision, err := validationeval.Decide(rule, typedSource{truth: tc.truth}, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: implementation,
			})
			if err != nil {
				t.Fatalf("the truth rule was refused: %v", err)
			}
			if decision.Result != tc.want {
				t.Fatalf("result=%q, want %q", decision.Result, tc.want)
			}
		})
	}
}

// A selected implementation is recorded as actually having been asked. Being
// selected and being called are different facts.
func TestASelectedTypedImplementationIsRecordedWhenItDecides(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.truth.is-true", Symbol: "is-true",
		Input: semantic.OperatorInputTruth,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputTruth},
		},
	}
	var observed []validationeval.ImplementationCall
	implementation, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.truth.is-true", Implementation: "tei.truth.is-true"}},
		validationeval.BuiltinImplementations(), func(call validationeval.ImplementationCall) {
			observed = append(observed, call)
		})
	if err != nil {
		t.Fatalf("the truth implementation could not be resolved: %v", err)
	}
	rule := semantic.Validation{
		ID: "sales.order_line.is_approved", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.approved", Operator: "sales.truth.is-true",
	}
	if _, err := validationeval.Decide(rule, typedSource{truth: true}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: implementation,
	}); err != nil {
		t.Fatalf("the truth rule was refused: %v", err)
	}
	if len(observed) != 1 {
		t.Fatalf("calls=%+v, want exactly one", observed)
	}
	if observed[0].Operator != "sales.truth.is-true" || observed[0].Implementation != "tei.truth.is-true" {
		t.Fatalf("call=%+v", observed[0])
	}
}

// A rule whose operator declares a text is not decided by a number
// implementation, even one selected for another operator: the value carries
// the type, and an implementation that decides numbers refuses a value that is
// not one.
func TestATextRuleIsNotDecidedByANumberImplementation(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.text.same-as", Symbol: "same-as",
		Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText},
			{Name: "other", Type: semantic.OperatorInputText},
		},
	}
	implementation, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.text.same-as", Implementation: "tei.number.non-negative"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		// The selection resolves: the name exists. What refuses it is the
		// contract check, which happens before any rule is read.
		t.Fatalf("the number implementation could not be resolved: %v", err)
	}
	if err := validationeval.CheckImplementationMatchesContract(contract, implementation[0]); err == nil {
		t.Fatal("a number implementation was accepted for a text contract")
	}
}
