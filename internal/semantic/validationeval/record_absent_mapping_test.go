package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A value the record certainly does not hold is absent, and a rule may state how
// that absence is read. A value the reader never fetched is a different state:
// nothing is known about it, including whether it is empty, so a mapping must not
// turn it into a value.
//
// The row-scoped path was corrected for this; the per-record path was not, so the
// same rule answered differently depending on which path read it. Measured before
// the fix: a not-fetched `high` with `or zero` named answered not-satisfied,
// because the mapping replaced the unread value with zero (#4359).
func TestAPerRecordMappingIsAppliedOnlyToAValueTheRecordDoesNotHold(t *testing.T) {
	rule := discountInRange
	rule.ID = "sales.order_line.range_with_open"
	rule.Inputs = []semantic.ValidationInputAssignment{
		{Name: "low", Value: "0"},
		{Name: "high", Value: "sales.order_line.unit_price", AbsentAs: "open"},
	}

	decide := func(t *testing.T, source validationeval.StatefulSource) validationeval.Decision {
		t.Helper()
		implementation := withinImplementation
		implementation.Operator = "sales.within"
		implementation.Absent = openIsOne()
		got, err := validationeval.Decide(rule, source, validationeval.Operators{
			Declared:        []semantic.Operator{withinContract},
			Implementations: []validationeval.OperatorImplementation{implementation},
		})
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		return got
	}

	t.Run("a value the record holds as empty is read under the mapping", func(t *testing.T) {
		source := &attemptingSource{
			values: map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "5")},
			nulls:  map[string]bool{"sales.order_line.unit_price": true},
			attempts: map[string]string{
				"sales.order_line.discount":   "read-1",
				"sales.order_line.unit_price": "read-1",
			},
		}
		// The mapping reads `open` as 1, and 5 is above it.
		if got := decide(t, source); got.Result != validationeval.ResultNotSatisfied {
			t.Fatalf("got=%+v, want not-satisfied: the record holds no price, and the mapping says how to read that",
				got)
		}
	})

	t.Run("a value the reader did not fetch is not read under the mapping", func(t *testing.T) {
		source := &attemptingSource{
			values: map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "5")},
			attempts: map[string]string{
				"sales.order_line.discount":   "read-1",
				"sales.order_line.unit_price": "read-1",
			},
		}
		got := decide(t, source)
		if got.Undecided == nil || got.Undecided.Reason != validationeval.UndecidedValueNotFetched {
			t.Fatalf("got=%+v, want undecided value-not-fetched: the price was never read, so nothing states it is empty",
				got)
		}
	})

	t.Run("the two reads of one rule must not disagree about which state it is", func(t *testing.T) {
		// Deciding the same rule twice with the same values, once with the
		// attribute empty and once unread, must not produce one answer: the
		// distinction is the point.
		empty := decide(t, &attemptingSource{
			values:   map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "5")},
			nulls:    map[string]bool{"sales.order_line.unit_price": true},
			attempts: map[string]string{"sales.order_line.discount": "read-1", "sales.order_line.unit_price": "read-1"},
		})
		unread := decide(t, &attemptingSource{
			values:   map[string]*big.Rat{"sales.order_line.discount": mustParseRat(t, "5")},
			attempts: map[string]string{"sales.order_line.discount": "read-1", "sales.order_line.unit_price": "read-1"},
		})
		if empty.Result == unread.Result && empty.Undecided == nil && unread.Undecided == nil {
			t.Fatal("an empty value and an unread value produced the same decided answer")
		}
	})
}
