package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule may write `or <mapping>` after an input and the declaration may state
// `absent as <mapping>`. Before this, the evaluator saw the absence first and
// reported value-not-fetched whatever the mapping said, so the mapping was
// accepted at authoring time and never used.
//
// The expectations are written from the design's statement of the mapping: an
// absent value read as a named value is that value, and an implementation that
// cannot read the mapping leaves the decision undecided rather than reading the
// absence as zero or as present.

// absentMapping finds the mapping named open and reads it as 1. It is a plain
// reading of the mapping, written here so a wrong evaluator is caught even if
// the shipped one is wrong too.
func openIsOne() func(string) (*big.Rat, bool) {
	return func(mapping string) (*big.Rat, bool) {
		if mapping != "open" {
			return nil, false
		}
		return big.NewRat(1, 1), true
	}
}

// rangeWithOpen states that the upper bound may be absent, read as open.
func rangeWithOpen(absent string) semantic.Validation {
	rule := discountInRange
	rule.ID = "sales.order_line.range_with_open"
	rule.Inputs = []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "sales.order_line.unit_price", AbsentAs: absent},
	}
	return rule
}

func TestAnAbsentInputIsReadUnderTheNamedMapping(t *testing.T) {
	// unit_price is absent, so `high` has no value; the mapping reads it as 1,
	// and the value 5 is above it, so the rule does not hold.
	source := &attemptingSource{
		values: map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "5")},
		attempts: map[string]string{
			"sales.order_line.discount":   "read-1",
			"sales.order_line.unit_price": "read-1",
		},
	}
	implementation := withinImplementation
	implementation.Operator = "sales.within"
	implementation.Absent = openIsOne()
	got, err := validationeval.Decide(rangeWithOpen("open"), source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("got=%+v, want not-satisfied (the mapped value is 1)", got)
	}
}

// An implementation that cannot read the mapping leaves the decision undecided
// with the absence's own reason. It must not be given zero, and the absence must
// not become a satisfied input.
func TestAnUnreadableMappingStaysUndecided(t *testing.T) {
	source := &attemptingSource{
		values: map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "5")},
		attempts: map[string]string{
			"sales.order_line.discount":   "read-1",
			"sales.order_line.unit_price": "read-1",
		},
	}
	implementation := withinImplementation
	implementation.Operator = "sales.within"
	// No Absent hook: this implementation cannot read any mapping.
	got, err := validationeval.Decide(rangeWithOpen("open"), source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Result != "" || got.Undecided == nil {
		t.Fatalf("got=%+v, want undecided", got)
	}
	if got.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("reason=%q, want %q", got.Undecided.Reason, validationeval.UndecidedValueNotFetched)
	}
}

// A hook that does not know the mapping this rule named is the same case: the
// implementation cannot read it, so the decision stays undecided.
func TestAHookThatDoesNotKnowTheMappingStaysUndecided(t *testing.T) {
	source := &attemptingSource{
		values: map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "5")},
		attempts: map[string]string{
			"sales.order_line.discount":   "read-1",
			"sales.order_line.unit_price": "read-1",
		},
	}
	implementation := withinImplementation
	implementation.Operator = "sales.within"
	implementation.Absent = openIsOne()
	got, err := validationeval.Decide(rangeWithOpen("closed"), source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Result != "" || got.Undecided == nil {
		t.Fatalf("got=%+v, want undecided", got)
	}
}

// A rule that names no mapping keeps the absence's reason even when the
// implementation could read a mapping it was not asked for.
func TestNoNamedMappingKeepsTheAbsenceReason(t *testing.T) {
	source := &attemptingSource{
		values: map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "5")},
		attempts: map[string]string{
			"sales.order_line.discount":   "read-1",
			"sales.order_line.unit_price": "read-1",
		},
	}
	implementation := withinImplementation
	implementation.Operator = "sales.within"
	implementation.Absent = openIsOne()
	got, err := validationeval.Decide(rangeWithOpen(""), source, validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Undecided == nil || got.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("got=%+v, want undecided value-not-fetched", got)
	}
}
