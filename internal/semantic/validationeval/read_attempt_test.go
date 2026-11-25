package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// B-10 of the decision document: a rule that reads the subject and a named
// input from attributes must read them from the same Read Attempt. Having three
// present values is not the same as knowing they are one snapshot, so a stated
// mismatch is refused and an unstated attempt is undecided rather than assumed.

// attemptingSource is a stateful reader that reports one attempt per attribute.
// A test can make two attributes disagree, which the legacy Source interface
// cannot express.
type attemptingSource struct {
	values   map[string]*big.Rat
	attempts map[string]string
	calls    []string
	// nulls names the attributes the record holds as empty. An attribute in
	// neither values nor nulls was not fetched. The two are different states,
	// so a test must be able to say which one it means.
	nulls map[string]bool
}

func (s *attemptingSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	s.calls = append(s.calls, string(attribute))
	value, ok := s.values[string(attribute)]
	if !ok {
		if s.nulls[string(attribute)] {
			return validationeval.NumberReading{State: validationeval.ValueNull}, nil
		}
		return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
	}
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: value}, nil
}

func (s *attemptingSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s *attemptingSource) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	attempt, stated := s.attempts[string(attribute)]
	return attempt, stated
}

// quantityWithinUnitPrice reads the subject (quantity) and one named input
// (high) from attributes, with low a constant.
func quantityWithinUnitPrice() semantic.Validation {
	rule := discountInRange
	rule.ID = "sales.order_line.quantity_within_unit_price"
	rule.Attribute = "sales.order_line.quantity"
	rule.Inputs = []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "sales.order_line.unit_price"},
	}
	return rule
}

func withinOperators() validationeval.Operators {
	return validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{withinImplementation},
	}
}

// Two attribute inputs read in one attempt are decided as usual: the check must
// not refuse a read that does report a single attempt.
func TestOneReadAttemptDecidesTheRule(t *testing.T) {
	source := &attemptingSource{
		values: map[string]*big.Rat{
			"sales.order_line.quantity":   mustParseRat(t, "5"),
			"sales.order_line.unit_price": mustParseRat(t, "100"),
		},
		attempts: map[string]string{
			"sales.order_line.quantity":   "read-1",
			"sales.order_line.unit_price": "read-1",
		},
	}
	got, err := validationeval.Decide(quantityWithinUnitPrice(), source, withinOperators())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Result != validationeval.ResultSatisfied {
		t.Fatalf("got=%+v, want satisfied", got)
	}
}

// Two attribute inputs read in different attempts are refused, not decided.
func TestTwoReadAttemptsAreRefused(t *testing.T) {
	source := &attemptingSource{
		values: map[string]*big.Rat{
			"sales.order_line.quantity":   mustParseRat(t, "5"),
			"sales.order_line.unit_price": mustParseRat(t, "100"),
		},
		attempts: map[string]string{
			"sales.order_line.quantity":   "read-1",
			"sales.order_line.unit_price": "read-2",
		},
	}
	got, err := validationeval.Decide(quantityWithinUnitPrice(), source, withinOperators())
	if err == nil {
		t.Fatalf("two attempts were accepted: %+v", got)
	}
	if !strings.Contains(err.Error(), "cannot mix read attempts") {
		t.Fatalf("the refusal does not name the mismatch: %v", err)
	}
	// The values were present and in range; the refusal must not be reported
	// as a decided answer.
	if got.Result != "" {
		t.Fatalf("a refused read produced the answer %q", got.Result)
	}
}

// A reader that cannot state an attempt leaves the rule undecided. It must not
// be assumed that the values came from one snapshot.
func TestAnUnstatedAttemptIsUndecided(t *testing.T) {
	source := &attemptingSource{
		values: map[string]*big.Rat{
			"sales.order_line.quantity":   mustParseRat(t, "5"),
			"sales.order_line.unit_price": mustParseRat(t, "100"),
		},
		attempts: map[string]string{},
	}
	got, err := validationeval.Decide(quantityWithinUnitPrice(), source, withinOperators())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Result != "" || got.Undecided == nil {
		t.Fatalf("got=%+v, want undecided", got)
	}
	if got.Undecided.Reason != validationeval.UndecidedReadAttemptUnknown {
		t.Fatalf("reason=%q, want %q", got.Undecided.Reason, validationeval.UndecidedReadAttemptUnknown)
	}
}

// A single attribute read needs no cross-read consistency, so a reader that
// states no attempt still decides an operator that reads only the subject.
// Requiring an attempt there would refuse every operator rule on a plain row.
func TestOneAttributeReadNeedsNoAttempt(t *testing.T) {
	rule := discountRule
	source := &attemptingSource{
		values: map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "-1")},
	}
	got, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{nonNegativeContract},
		Implementations: []validationeval.OperatorImplementation{(&countingImplementation{}).nonNegative()},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("got=%+v, want not-satisfied", got)
	}
}
