package validationeval_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A value no type reads is not a value. Handing it to an implementation as an
// empty one would let a rule be answered from a value it never held, which is
// what section 8.4 refuses, so the reader that produced it is named instead.
//
// Measured before this was refused: the named input received an empty value,
// no error was reported, and the rule was decided satisfied.
func TestAnAnyValueNoTypeReadsIsRefused(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: semantic.OperatorInputAny,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputAny},
			{Name: "other", Type: semantic.OperatorInputAny},
		},
	}
	// The reader hands back a value whose Type says text and whose Text is
	// nil: no type reads it.
	source := unreadableAnySource{}
	asked := false
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op", Input: semantic.OperatorInputAny,
		Values: func(values map[string]semantic.OperatorValue) (bool, error) {
			asked = true
			for name, value := range values {
				if !value.Valid() {
					t.Errorf("input %q was handed a value no type reads: %+v", name, value)
				}
			}
			return true, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.op",
		Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: "sales.order_line.item_name"}},
	}
	_, err := validationeval.Decide(rule, source,
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
	if err == nil {
		t.Fatal("a value no type reads was accepted")
	}
	if !strings.Contains(err.Error(), "no type reads") {
		t.Fatalf("the refusal does not name what is wrong: %v", err)
	}
	if !strings.Contains(err.Error(), "sales.order_line.item_name") {
		t.Fatalf("the refusal does not name the attribute the reader misread: %v", err)
	}
	if asked {
		t.Error("the implementation was asked about a value no type reads")
	}
}

// unreadableAnySource reports one value no type reads for every attribute.
type unreadableAnySource struct{}

func (unreadableAnySource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (unreadableAnySource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent}, nil
}

func (unreadableAnySource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (unreadableAnySource) ReadAny(attribute core.CanonicalID) (validationeval.AnyReading, error) {
	// Type says text and Text is nil, so AnyValue.Valid() is false.
	return validationeval.AnyReading{
		State: validationeval.ValuePresent,
		Value: semantic.AnyValue{Type: semantic.OperatorInputText},
	}, nil
}

// A value the reader did read, of a type it did read it as, is carried
// unchanged: this fix refuses only the value no type reads.
func TestAnAnyValueTheReaderDidReadIsCarried(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: semantic.OperatorInputAny,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputAny},
		},
	}
	text := "A-100"
	source := readableAnySource{value: semantic.AnyValue{
		Type: semantic.OperatorInputText, Text: &text,
	}}
	var got semantic.OperatorValue
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op", Input: semantic.OperatorInputAny,
		Values: func(values map[string]semantic.OperatorValue) (bool, error) {
			got = values["value"]
			return true, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.op",
	}
	if _, err := validationeval.Decide(rule, source,
		validationeval.Operators{
			Declared:        []semantic.Operator{contract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		}); err != nil {
		t.Fatalf("a value the reader did read was refused: %v", err)
	}
	if got.Kind() != semantic.OperatorInputAny {
		t.Fatalf("kind=%q, want any", got.Kind())
	}
	if got.Any == nil || got.Any.Type != semantic.OperatorInputText || got.Any.Text == nil || *got.Any.Text != "A-100" {
		t.Fatalf("value=%+v, want the text A-100 keeping its type", got)
	}
}

// readableAnySource reports one value of one type for every attribute.
type readableAnySource struct {
	value semantic.AnyValue
}

func (s readableAnySource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (readableAnySource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent}, nil
}

func (readableAnySource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s readableAnySource) ReadAny(attribute core.CanonicalID) (validationeval.AnyReading, error) {
	value := s.value
	return validationeval.AnyReading{State: validationeval.ValuePresent, Value: value}, nil
}
