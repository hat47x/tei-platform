package validationeval_test

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The operator used here, sales.non_negative, is not known to any production
// code. Its contract is a value built in this file, as a loaded document would
// provide it, and its implementation is a function written in this file, as a
// selected plugin would provide it. Nothing else is added to evaluate it.

const nonNegative core.CanonicalID = "sales.non_negative"

var nonNegativeContract = semantic.Operator{ID: nonNegative, Symbol: "non_negative", Input: semantic.OperatorInputNumber}

var discountRule = semantic.Validation{
	ID: "sales.order_line.discount_non_negative", Symbol: "discount_non_negative",
	Kind: semantic.ValidationOperator, Attribute: "sales.order_line.discount", Operator: nonNegative,
}

type countingImplementation struct {
	calls int
	seen  []string
}

func (c *countingImplementation) nonNegative() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: nonNegative,
		Input:    semantic.OperatorInputNumber,
		Number: func(value *big.Rat) (bool, error) {
			c.calls++
			c.seen = append(c.seen, value.RatString())
			return value.Sign() >= 0, nil
		},
	}
}

type operatorSource struct {
	value *big.Rat
	err   error
	calls []string
}

func (s *operatorSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	s.calls = append(s.calls, "Number("+string(attribute)+")")
	return s.value, s.err
}

func (s *operatorSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	s.calls = append(s.calls, "RelationKeys("+string(relation)+","+string(attribute)+")")
	return nil, fmt.Errorf("an operator that takes a number must not read a relation")
}

func TestDeclaredOperatorDecidesTheRuleAndReadsOnlyItsAttribute(t *testing.T) {
	// Independent expected values: the definition of "not negative" on three
	// hand-picked numbers. Zero is the case that separates it from positive.
	cases := []struct {
		value *big.Rat
		want  validationeval.Result
	}{
		{big.NewRat(2, 1), validationeval.ResultSatisfied},
		{new(big.Rat), validationeval.ResultSatisfied},
		{big.NewRat(-1, 100), validationeval.ResultNotSatisfied},
	}
	for _, tc := range cases {
		implementation := &countingImplementation{}
		source := &operatorSource{value: tc.value}
		got, err := validationeval.EvaluateWithOperators(discountRule, source, validationeval.Operators{
			Declared:        []semantic.Operator{nonNegativeContract},
			Implementations: []validationeval.OperatorImplementation{implementation.nonNegative()},
		})
		if err != nil || got != tc.want {
			t.Fatalf("value %s: result=%q err=%v, want %q", tc.value.RatString(), got, err, tc.want)
		}
		if calls := strings.Join(source.calls, ","); calls != "Number(sales.order_line.discount)" {
			t.Fatalf("value %s: the rule read %q, want exactly its declared attribute once", tc.value.RatString(), calls)
		}
		if implementation.calls != 1 || implementation.seen[0] != tc.value.RatString() {
			t.Fatalf("implementation calls=%d seen=%v", implementation.calls, implementation.seen)
		}
	}

	// The built-in positive disagrees on zero, so the two are different rules
	// and a declared operator was not quietly mapped onto a built-in kind.
	positive := semantic.Validation{ID: "sales.p", Kind: semantic.ValidationPositive, Attribute: "sales.order_line.discount"}
	if got, err := validationeval.Evaluate(positive, &operatorSource{value: new(big.Rat)}); err != nil || got != validationeval.ResultNotSatisfied {
		t.Fatalf("positive on zero: %q %v", got, err)
	}
}

func TestDeclaredOperatorIsNeverDecidedWithoutExactlyOneMatchingImplementation(t *testing.T) {
	good := &countingImplementation{}
	other := validationeval.OperatorImplementation{
		Operator: "purchase.non_negative", Input: semantic.OperatorInputNumber,
		Number: func(*big.Rat) (bool, error) { return true, errors.New("the unselected implementation was called") },
	}
	wrongInput := good.nonNegative()
	wrongInput.Input = "keys"
	noFunction := good.nonNegative()
	noFunction.Number = nil

	cases := []struct {
		name      string
		operators validationeval.Operators
		want      string
	}{
		{"not declared", validationeval.Operators{
			Implementations: []validationeval.OperatorImplementation{good.nonNegative()}}, "is not declared"},
		{"declared twice", validationeval.Operators{
			Declared:        []semantic.Operator{nonNegativeContract, nonNegativeContract},
			Implementations: []validationeval.OperatorImplementation{good.nonNegative()}}, "declared 2 times"},
		{"no implementation", validationeval.Operators{
			Declared: []semantic.Operator{nonNegativeContract}}, "no selected implementation"},
		{"only another operator's implementation", validationeval.Operators{
			Declared:        []semantic.Operator{nonNegativeContract},
			Implementations: []validationeval.OperatorImplementation{other}}, "no selected implementation"},
		{"two implementations", validationeval.Operators{
			Declared:        []semantic.Operator{nonNegativeContract},
			Implementations: []validationeval.OperatorImplementation{good.nonNegative(), good.nonNegative()}}, "2 implementations"},
		{"implementation written for another input", validationeval.Operators{
			Declared:        []semantic.Operator{nonNegativeContract},
			Implementations: []validationeval.OperatorImplementation{wrongInput}}, "its implementation takes"},
		{"implementation without a number function", validationeval.Operators{
			Declared:        []semantic.Operator{nonNegativeContract},
			Implementations: []validationeval.OperatorImplementation{noFunction}}, "has no decision function"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			good.calls = 0
			source := &operatorSource{value: big.NewRat(2, 1)}
			got, err := validationeval.EvaluateWithOperators(discountRule, source, tc.operators)
			if err == nil || got != "" || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("result=%q err=%v, want an error containing %q and no result", got, err, tc.want)
			}
			if good.calls != 0 || len(source.calls) != 0 {
				t.Fatalf("a refused evaluation still ran: implementation calls=%d source calls=%v", good.calls, source.calls)
			}
		})
	}

	t.Run("an unselected implementation of another operator is not called", func(t *testing.T) {
		good.calls = 0
		for _, implementations := range [][]validationeval.OperatorImplementation{
			{good.nonNegative(), other}, {other, good.nonNegative()},
		} {
			got, err := validationeval.EvaluateWithOperators(discountRule, &operatorSource{value: big.NewRat(2, 1)}, validationeval.Operators{
				Declared: []semantic.Operator{nonNegativeContract}, Implementations: implementations,
			})
			if err != nil || got != validationeval.ResultSatisfied {
				t.Fatalf("result=%q err=%v", got, err)
			}
		}
		if good.calls != 2 {
			t.Fatalf("selected implementation calls=%d, want 2", good.calls)
		}
	})

	t.Run("the plain evaluator refuses an operator rule", func(t *testing.T) {
		got, err := validationeval.Evaluate(discountRule, &operatorSource{value: big.NewRat(2, 1)})
		if err == nil || got != "" {
			t.Fatalf("an operator rule was decided without operators: %q %v", got, err)
		}
	})
}

func TestDeclaredOperatorDoesNotTurnAMissingValueIntoANumber(t *testing.T) {
	implementation := &countingImplementation{}
	operators := validationeval.Operators{
		Declared:        []semantic.Operator{nonNegativeContract},
		Implementations: []validationeval.OperatorImplementation{implementation.nonNegative()},
	}
	// No value: zero would be "satisfied" for this operator, so treating the
	// missing value as zero would report a rule as satisfied without data.
	if got, err := validationeval.EvaluateWithOperators(discountRule, &operatorSource{}, operators); err == nil || got != "" {
		t.Fatalf("a missing value was decided: %q %v", got, err)
	}
	failed := errors.New("row was not fetched")
	if got, err := validationeval.EvaluateWithOperators(discountRule, &operatorSource{err: failed}, operators); !errors.Is(err, failed) || got != "" {
		t.Fatalf("a failed read was decided: %q %v", got, err)
	}
	if implementation.calls != 0 {
		t.Fatalf("the implementation was asked %d times about a value that does not exist", implementation.calls)
	}
}

func TestDeclaredOperatorImplementationCannotChangeTheSourceValue(t *testing.T) {
	held := big.NewRat(5, 1)
	mutating := validationeval.OperatorImplementation{
		Operator: nonNegative, Input: semantic.OperatorInputNumber,
		Number: func(value *big.Rat) (bool, error) {
			value.SetInt64(-9)
			return true, nil
		},
	}
	_, err := validationeval.EvaluateWithOperators(discountRule, &operatorSource{value: held}, validationeval.Operators{
		Declared: []semantic.Operator{nonNegativeContract}, Implementations: []validationeval.OperatorImplementation{mutating},
	})
	if err != nil {
		t.Fatal(err)
	}
	if held.Cmp(big.NewRat(5, 1)) != 0 {
		t.Fatalf("the implementation changed the value the Source holds: %s", held.RatString())
	}
}
