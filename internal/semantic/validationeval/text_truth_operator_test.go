package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A text is read as a text and a truth as a truth. The reader reports what the
// attribute holds, and the implementation receives the value carrying the type
// it was read as (decision section 8.4), so it never has to ask which kind it
// holds and never sees a Source.
//
// The expectations are written from what a text and a truth are, not read back
// from the reader: a text is what the attribute holds, and the number 7 is not
// the text "7".

// textTruthSource reads one record: one text attribute and one truth
// attribute, both holding values, plus a number that must never be read as a
// text or a truth.
type textTruthSource struct {
	text   string
	truth  bool
	number string
}

func (s textTruthSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (s textTruthSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s textTruthSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s textTruthSource) ReadText(attribute core.CanonicalID) (validationeval.TextReading, error) {
	switch attribute {
	case "sales.order_line.code":
		return validationeval.TextReading{State: validationeval.ValuePresent, Value: s.text}, nil
	case "sales.order_line.quantity":
		return validationeval.TextReading{State: validationeval.ValueNotFetched}, nil
	}
	return validationeval.TextReading{State: validationeval.ValueNotFetched}, nil
}

func (s textTruthSource) ReadTruth(attribute core.CanonicalID) (validationeval.TruthReading, error) {
	switch attribute {
	case "sales.order_line.approved":
		return validationeval.TruthReading{State: validationeval.ValuePresent, Value: s.truth}, nil
	case "sales.order_line.quantity":
		return validationeval.TruthReading{State: validationeval.ValueNotFetched}, nil
	}
	return validationeval.TruthReading{State: validationeval.ValueNotFetched}, nil
}

// A text input is decided from the text the attribute holds, and the
// implementation receives it as a text.
func TestATextInputIsDecidedFromTheTextItHolds(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.text.same-as", Symbol: "same-as",
		Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText},
			{Name: "other", Type: semantic.OperatorInputText},
		},
	}
	for _, tc := range []struct {
		name  string
		src   textTruthSource
		other string
		want  validationeval.Result
	}{
		{"the same text", textTruthSource{text: "A-100"}, `"A-100"`, validationeval.ResultSatisfied},
		{"a different text", textTruthSource{text: "B-200"}, `"A-100"`, validationeval.ResultNotSatisfied},
		// The empty text is a value of length zero, not the absence of a
		// value, so it is asked about and answered (section 8.3).
		{"an empty text", textTruthSource{text: ""}, `""`, validationeval.ResultSatisfied},
		// A text that holds no value is a different state from the empty
		// text, and the rule is undecided rather than answered.
		{"a text against an empty one", textTruthSource{text: "A-100"}, `""`, validationeval.ResultNotSatisfied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := semantic.Validation{
				ID: "sales.order_line.code_matches", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.code", Operator: "sales.text.same-as",
				Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: tc.other}},
			}
			implementation, err := validationeval.ResolveImplementations(
				[]validationeval.ImplementationSelection{{Operator: "sales.text.same-as", Implementation: "tei.text.same-as"}},
				validationeval.BuiltinImplementations(), nil)
			if err != nil {
				t.Fatalf("the implementation could not be resolved: %v", err)
			}
			decision, err := validationeval.Decide(rule, tc.src, validationeval.Operators{
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

// A truth input is decided from the truth the attribute holds, and the
// implementation receives it as a truth rather than as the number 1.
func TestATruthInputIsDecidedFromTheTruthItHolds(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.truth.is-true", Symbol: "is-true",
		Input: semantic.OperatorInputTruth,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputTruth},
		},
	}
	for _, tc := range []struct {
		name string
		src  textTruthSource
		want validationeval.Result
	}{
		{"true", textTruthSource{truth: true}, validationeval.ResultSatisfied},
		{"false", textTruthSource{truth: false}, validationeval.ResultNotSatisfied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := semantic.Validation{
				ID: "sales.order_line.is_approved", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.approved", Operator: "sales.truth.is-true",
			}
			implementation, err := validationeval.ResolveImplementations(
				[]validationeval.ImplementationSelection{{Operator: "sales.truth.is-true", Implementation: "tei.truth.is-true"}},
				validationeval.BuiltinImplementations(), nil)
			if err != nil {
				t.Fatalf("the implementation could not be resolved: %v", err)
			}
			decision, err := validationeval.Decide(rule, tc.src, validationeval.Operators{
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

// A record that holds no text has no value to ask the operator about, and the
// answer is undecided with the record's own reason. The empty text is a value,
// so it is not this state.
func TestARecordWithNoTextIsUndecided(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.text.same-as", Symbol: "same-as",
		Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText},
			{Name: "other", Type: semantic.OperatorInputText},
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.code_matches", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.code", Operator: "sales.text.same-as",
		Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: `"A-100"`}},
	}
	implementation, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.text.same-as", Implementation: "tei.text.same-as"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		t.Fatalf("the implementation could not be resolved: %v", err)
	}
	decision, err := validationeval.Decide(rule, absentTextSource{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: implementation,
	})
	if err != nil {
		t.Fatalf("the rule was refused: %v", err)
	}
	if decision.Result != "" {
		t.Fatalf("result=%q, want no result", decision.Result)
	}
	if decision.Undecided == nil {
		t.Fatal("the decision states no reason it is undecided")
	}
	if decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("reason=%q, want value-null", decision.Undecided.Reason)
	}
}

// absentTextSource reports the subject text attribute as holding no value.
type absentTextSource struct{ textTruthSource }

func (absentTextSource) ReadText(attribute core.CanonicalID) (validationeval.TextReading, error) {
	if attribute == "sales.order_line.code" {
		return validationeval.TextReading{State: validationeval.ValueNull}, nil
	}
	return validationeval.TextReading{State: validationeval.ValueNotFetched}, nil
}

// A constant assigned to a text input is read as a text, so it is written with
// quotes (section 8.3). One written without them is refused at the position of
// the assignment rather than read as the nearest value.
func TestATextConstantIsReadAsAText(t *testing.T) {
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
		t.Fatalf("the implementation could not be resolved: %v", err)
	}
	for _, tc := range []struct {
		name    string
		other   string
		refused bool
	}{
		{"a quoted text", `"A-100"`, false},
		{"an empty text", `""`, false},
		{"an unquoted text", "A-100", true},
		{"a number", "100", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := semantic.Validation{
				ID: "sales.order_line.code_matches", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.code", Operator: "sales.text.same-as",
				Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: tc.other}},
			}
			decision, err := validationeval.Decide(rule, textTruthSource{text: "A-100"}, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: implementation,
			})
			if tc.refused {
				if err == nil {
					t.Fatalf("%q was read as a text constant", tc.other)
				}
				return
			}
			if err != nil {
				t.Fatalf("the quoted text was refused: %v", err)
			}
			if tc.other == `""` {
				// The subject holds "A-100", so the answer is not satisfied.
				if decision.Result != validationeval.ResultNotSatisfied {
					t.Fatalf("result=%q, want not-satisfied", decision.Result)
				}
				return
			}
			if decision.Result != validationeval.ResultSatisfied {
				t.Fatalf("result=%q, want satisfied", decision.Result)
			}
		})
	}
}

// A truth constant is read as a truth, written true or false. The number 1 is
// not one.
func TestATruthConstantIsReadAsATruth(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.truth.is-true", Symbol: "is-true",
		Input: semantic.OperatorInputTruth,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputTruth},
			{Name: "other", Type: semantic.OperatorInputTruth},
		},
	}
	implementation, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.truth.is-true", Implementation: "tei.truth.is-true"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		t.Fatalf("the implementation could not be resolved: %v", err)
	}
	for _, tc := range []struct {
		other   string
		refused bool
	}{
		{"true", false},
		{"false", false},
		{"1", true},
		{"yes", true},
	} {
		t.Run(tc.other, func(t *testing.T) {
			rule := semantic.Validation{
				ID: "sales.order_line.is_approved", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.approved", Operator: "sales.truth.is-true",
				Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: tc.other}},
			}
			_, err := validationeval.Decide(rule, textTruthSource{truth: true}, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: implementation,
			})
			if tc.refused {
				if err == nil {
					t.Fatalf("%q was read as a truth constant", tc.other)
				}
				return
			}
			if err != nil {
				t.Fatalf("the truth constant was refused: %v", err)
			}
		})
	}
}

// A reader that cannot read a text refuses a rule that needs one, rather than
// answering from a number. A reader that cannot read a truth refuses a rule
// that needs one, rather than answering from the number 1.
func TestAReaderThatCannotReadTheTextOrTruthRefusesTheRule(t *testing.T) {
	for _, tc := range []struct {
		input     semantic.OperatorInput
		attribute core.CanonicalID
	}{
		{semantic.OperatorInputText, "sales.order_line.code"},
		{semantic.OperatorInputTruth, "sales.order_line.approved"},
	} {
		t.Run(string(tc.input), func(t *testing.T) {
			contract := semantic.Operator{
				ID: "sales.op", Symbol: "op", Input: tc.input,
				Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: tc.input}},
			}
			implementation := validationeval.OperatorImplementation{
				Operator: "sales.op",
				Input:    tc.input,
				Values:   func(map[string]semantic.OperatorValue) (bool, error) { return true, nil },
			}
			rule := semantic.Validation{
				ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
				Attribute: tc.attribute, Operator: "sales.op",
			}
			_, err := validationeval.Decide(rule, numberOnlySource{}, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
			if err == nil {
				t.Fatalf("a %s input was decided by a reader that cannot read one", tc.input)
			}
			if !strings.Contains(err.Error(), "cannot read a "+string(tc.input)) {
				t.Fatalf("the refusal does not say what the reader cannot read: %v", err)
			}
		})
	}
}

// numberOnlySource reads a number and nothing else.
type numberOnlySource struct{}

func (numberOnlySource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(7, 1)}, nil
}

func (numberOnlySource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (numberOnlySource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }
