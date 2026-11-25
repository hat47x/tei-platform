package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule may write `absent as <mapping>` on one of its named inputs and the
// declaration may state the same mapping. The mapping is for a value the record
// certainly holds no value for. It is NOT for a value the reader never fetched:
// nothing is known about an unread value, including whether it is empty, so
// reading it as the mapped value would decide a rule from a value nobody read
// (decision section 5).
//
// The row-scoped path already guarded this (`withAbsentMapping` reads only rows
// in the ValueNull state). The per-record path did not: it applied the mapping
// to whatever reason the reader gave, so the same rule answered differently
// depending on which path read it.
//
// The expectations below are written from that meaning: a null value is read as
// the mapped 1, an unread value stays undecided with its own reason, and the
// operator is asked about neither in the unread case.

// stateSource returns a stated state for each attribute, so one test can hold a
// value, a null value, and an unread value at the same time.
type stateSource struct {
	readings map[string]validationeval.NumberReading
	attempts map[string]string
}

func (s *stateSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	reading, ok := s.readings[string(attribute)]
	if !ok {
		return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
	}
	return reading, nil
}

func (s *stateSource) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	attempt, stated := s.attempts[string(attribute)]
	return attempt, stated
}

func (s *stateSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func stateSourceFor(high validationeval.NumberReading) *stateSource {
	return &stateSource{
		readings: map[string]validationeval.NumberReading{
			"sales.order_line.discount":   {State: validationeval.ValuePresent, Value: big.NewRat(5, 1)},
			"sales.order_line.unit_price": high,
		},
		attempts: map[string]string{
			"sales.order_line.discount":   "read-1",
			"sales.order_line.unit_price": "read-1",
		},
	}
}

func decideRangeWithOpen(t *testing.T, high validationeval.NumberReading) validationeval.Decision {
	t.Helper()
	implementation := withinImplementation
	implementation.Operator = "sales.within"
	implementation.Absent = openIsOne()
	decision, err := validationeval.Decide(rangeWithOpen("open"), stateSourceFor(high), validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	return decision
}

// A value the record holds no value for is read under the named mapping: the
// bound becomes 1, and the value 5 is above it.
func TestANullInputIsReadUnderTheNamedMapping(t *testing.T) {
	decision := decideRangeWithOpen(t, validationeval.NumberReading{State: validationeval.ValueNull})
	if decision.Result != validationeval.ResultNotSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want not-satisfied (the mapped value is 1)", decision)
	}
}

// A value that was not fetched is unknown, not absent. The mapping does not
// apply, and the rule stays undecided with the reader's own reason.
func TestAnUnreadInputIsNotReadUnderTheNamedMapping(t *testing.T) {
	decision := decideRangeWithOpen(t, validationeval.NumberReading{State: validationeval.ValueNotFetched})
	if decision.Result != "" {
		t.Fatalf("decision=%+v, want no result: an unread value cannot be decided from a mapping", decision)
	}
	if decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNotFetched {
		t.Fatalf("decision=%+v, want undecided value-not-fetched (the reader's reason, not the mapped value)", decision)
	}
}

// The same rule with no mapping named keeps the absence reason: the guard must
// not be the mapping alone.
func TestANullInputWithoutANamedMappingStaysUndecided(t *testing.T) {
	rule := rangeWithOpen("")
	implementation := withinImplementation
	implementation.Operator = "sales.within"
	implementation.Absent = openIsOne()
	decision, err := validationeval.Decide(rule, stateSourceFor(validationeval.NumberReading{State: validationeval.ValueNull}), validationeval.Operators{
		Declared:        []semantic.Operator{withinContract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Undecided == nil || decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("decision=%+v, want undecided value-null", decision)
	}
}
