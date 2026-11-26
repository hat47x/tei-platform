package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// typedValueSource is a reader that can read every kind this program reads, so
// the operator path can be exercised for each declared type.
type typedValueSource struct {
	number validationeval.NumberReading
	date   validationeval.DateReading
	text   validationeval.TextReading
	truth  validationeval.TruthReading
	any    validationeval.AnyReading
}

func (s typedValueSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return s.number, nil
}

func (s typedValueSource) ReadDate(core.CanonicalID) (validationeval.DateReading, error) {
	return s.date, nil
}

func (s typedValueSource) ReadText(core.CanonicalID) (validationeval.TextReading, error) {
	return s.text, nil
}

func (s typedValueSource) ReadTruth(core.CanonicalID) (validationeval.TruthReading, error) {
	return s.truth, nil
}

func (s typedValueSource) ReadAny(core.CanonicalID) (validationeval.AnyReading, error) {
	return s.any, nil
}

func (s typedValueSource) ReadRelationKeys(_, _ core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

// TestTheOperatorPathReadsEachDeclaredTypeAsThatType pins decision section 8.4's
// rule for the operator path: the evaluator reads a value the way the
// declaration says the input is read and fills that type's field only. An
// implementation therefore receives the value in the field its declaration
// names, and a value the reader did not fetch is never read as another type.
func TestTheOperatorPathReadsEachDeclaredTypeAsThatType(t *testing.T) {
	number := new(big.Rat)
	number.SetInt64(7)

	cases := []struct {
		name  string
		input semantic.OperatorInput
		value semantic.OperatorValue
	}{
		{"number", semantic.OperatorInputNumber, semantic.OperatorValue{Number: number}},
		{"text", semantic.OperatorInputText, semantic.OperatorValue{Text: strPtr("A-100")}},
		{"truth", semantic.OperatorInputTruth, truthVal()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := typedValueSource{
				number: validationeval.NumberReading{State: validationeval.ValuePresent, Value: number},
				text:   validationeval.TextReading{State: validationeval.ValuePresent, Value: "A-100"},
				truth:  validationeval.TruthReading{State: validationeval.ValuePresent, Value: true},
			}
			contract := semantic.Operator{
				ID: "sales.op", Symbol: "op", Input: tc.input,
				Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: tc.input}},
			}
			rule := semantic.Validation{
				ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.quantity", Operator: "sales.op",
			}
			seen := make(chan semantic.OperatorValue, 1)
			implementation := validationeval.OperatorImplementation{
				Operator: "sales.op",
				Input:    tc.input,
				Values: func(values map[string]semantic.OperatorValue) (bool, error) {
					seen <- values["value"]
					return true, nil
				},
			}
			_, decideErr := validationeval.Decide(rule, source, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
			if decideErr != nil {
				t.Fatalf("the %s rule was refused: %v", tc.name, decideErr)
			}
			select {
			case got := <-seen:
				if got.Kind() != tc.input {
					t.Fatalf("the implementation received %s, want %s", got.Kind(), tc.input)
				}
				// The value has to be the one the reader read, not the zero
				// value of the field: reading a text as an empty string would
				// answer a question about a value nobody read.
				if !sameOperatorValue(got, tc.value) {
					t.Fatalf("the implementation received %+v, want %+v", got, tc.value)
				}
			default:
				t.Fatalf("the implementation was not asked about the %s input", tc.name)
			}
		})
	}
}

// TestTheOperatorPathRefusesAReaderWithoutTheDeclaredType pins the other side of
// section 8.4: a reader that cannot read the declared type is an error, and the
// rule is not decided from a value it was never read as.
func TestTheOperatorPathRefusesAReaderWithoutTheDeclaredType(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputText}},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity", Operator: "sales.op",
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op",
		Input:    semantic.OperatorInputText,
		Values: func(map[string]semantic.OperatorValue) (bool, error) {
			t.Fatal("the implementation was asked by a reader that cannot read text")
			return false, nil
		},
	}
	// numbersOnly reads numbers and nothing else.
	_, err := validationeval.Decide(rule, numbersOnly{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		t.Fatal("a text rule was decided by a reader that reads numbers")
	}
}

type numbersOnly struct{}

func (numbersOnly) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: new(big.Rat)}, nil
}

func (numbersOnly) ReadRelationKeys(_, _ core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func truthVal() semantic.OperatorValue {
	t := true
	return semantic.OperatorValue{Truth: &t}
}

// sameOperatorValue reports whether two values are the same value, so a test
// can check the field and the value in it rather than only the kind.
func sameOperatorValue(left, right semantic.OperatorValue) bool {
	if left.Kind() != right.Kind() {
		return false
	}
	switch left.Kind() {
	case semantic.OperatorInputNumber:
		return left.Number != nil && right.Number != nil && left.Number.Cmp(right.Number) == 0
	case semantic.OperatorInputDate:
		return left.Date != nil && right.Date != nil && left.Date.Compare(*right.Date) == 0
	case semantic.OperatorInputText:
		return left.Text != nil && right.Text != nil && *left.Text == *right.Text
	case semantic.OperatorInputTruth:
		return left.Truth != nil && right.Truth != nil && *left.Truth == *right.Truth
	case semantic.OperatorInputAny:
		return left.Any != nil && right.Any != nil && left.Any.Valid() == right.Any.Valid()
	default:
		return false
	}
}
