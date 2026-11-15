package validationeval_test

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The contract states three examples of "not negative". They were written
// from the meaning of the words, not taken from an implementation.
var nonNegativeWithCases = semantic.Operator{
	ID: nonNegative, Symbol: "non_negative", Input: semantic.OperatorInputNumber,
	Cases: []semantic.OperatorCase{
		{Value: "300", Satisfied: true},
		{Value: "0", Satisfied: true},
		{Value: "-0.01", Satisfied: false},
	},
}

func numberImplementation(decide func(*big.Rat) (bool, error)) (validationeval.OperatorImplementation, *[]string) {
	var asked []string
	return validationeval.OperatorImplementation{
		Operator: nonNegative, Input: semantic.OperatorInputNumber,
		Number: func(value *big.Rat) (bool, error) {
			asked = append(asked, value.RatString())
			return decide(value)
		},
	}, &asked
}

func TestImplementationAgreeingWithTheExamplesDecidesTheRule(t *testing.T) {
	implementation, asked := numberImplementation(func(value *big.Rat) (bool, error) { return value.Sign() >= 0, nil })
	source := &operatorSource{value: big.NewRat(5, 1)}
	got, err := validationeval.EvaluateWithOperators(discountRule, source, validationeval.Operators{
		Declared:        []semantic.Operator{nonNegativeWithCases},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil || got != validationeval.ResultSatisfied {
		t.Fatalf("result=%q err=%v", got, err)
	}
	// The examples are asked first, in the order written, then the real value.
	if want := "300,0,-1/100,5"; strings.Join(*asked, ",") != want {
		t.Fatalf("implementation was asked %v, want %s", *asked, want)
	}
	if strings.Join(source.calls, ",") != "Number(sales.order_line.discount)" {
		t.Fatalf("source calls=%v", source.calls)
	}
}

// The real value 5 is satisfied under "not negative", "not zero", and
// "positive" alike. Only the example for 0 tells them apart, so without the
// examples a wrong implementation would pass on this data.
func TestImplementationDisagreeingWithAnExampleIsNotUsed(t *testing.T) {
	wrong := map[string]func(*big.Rat) (bool, error){
		"not zero":     func(value *big.Rat) (bool, error) { return value.Sign() != 0, nil },
		"positive":     func(value *big.Rat) (bool, error) { return value.Sign() > 0, nil },
		"always true":  func(*big.Rat) (bool, error) { return true, nil },
		"always false": func(*big.Rat) (bool, error) { return false, nil },
		"negated":      func(value *big.Rat) (bool, error) { return value.Sign() < 0, nil },
	}
	for name, decide := range wrong {
		t.Run(name, func(t *testing.T) {
			implementation, _ := numberImplementation(decide)
			source := &operatorSource{value: big.NewRat(5, 1)}
			got, err := validationeval.EvaluateWithOperators(discountRule, source, validationeval.Operators{
				Declared:        []semantic.Operator{nonNegativeWithCases},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
			if err == nil || got != "" || !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_MISMATCH") {
				t.Fatalf("a wrong implementation decided the rule: %q %v", got, err)
			}
			if len(source.calls) != 0 {
				t.Fatalf("real data was read for an implementation that failed the examples: %v", source.calls)
			}
		})
	}

	// The same wrong implementation is accepted when the contract states no
	// example. This is what the examples buy.
	implementation, _ := numberImplementation(wrong["not zero"])
	got, err := validationeval.EvaluateWithOperators(discountRule, &operatorSource{value: big.NewRat(5, 1)}, validationeval.Operators{
		Declared:        []semantic.Operator{nonNegativeContract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil || got != validationeval.ResultSatisfied {
		t.Fatalf("without examples: %q %v", got, err)
	}
}

func TestImplementationFailingOnAnExampleIsNotUsed(t *testing.T) {
	failed := errors.New("cannot decide")
	implementation, _ := numberImplementation(func(value *big.Rat) (bool, error) {
		if value.Sign() == 0 {
			return false, failed
		}
		return true, nil
	})
	source := &operatorSource{value: big.NewRat(5, 1)}
	got, err := validationeval.EvaluateWithOperators(discountRule, source, validationeval.Operators{
		Declared:        []semantic.Operator{nonNegativeWithCases},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if !errors.Is(err, failed) || got != "" || len(source.calls) != 0 {
		t.Fatalf("result=%q err=%v source calls=%v", got, err, source.calls)
	}
}

func TestOperatorCaseValuesAreReadExactly(t *testing.T) {
	// Expected fractions are written by hand.
	for text, want := range map[string]*big.Rat{
		"0": new(big.Rat), "300": big.NewRat(300, 1), "-0.01": big.NewRat(-1, 100),
		"0.10": big.NewRat(1, 10), "12345678901234567890.5": func() *big.Rat { r, _ := new(big.Rat).SetString("24691357802469135781/2"); return r }(),
	} {
		got, err := semantic.ParseOperatorCaseValue(text)
		if err != nil || got.Cmp(want) != 0 {
			t.Fatalf("%q read as %v (%v), want %s", text, got, err, want.RatString())
		}
	}
	for _, text := range []string{"", "-", ".", "1e5", "1,000", "+1", "007", ".5", "5.", " 1", "1 ", "0x10", "1.2.3", "--1", "１"} {
		if got, err := semantic.ParseOperatorCaseValue(text); err == nil {
			t.Fatalf("%q was read as %s", text, got.RatString())
		}
	}
}
