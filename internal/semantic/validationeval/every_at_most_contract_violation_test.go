package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic/validationeval"
)

// A row the reader reports as PRESENT without a value is a contract violation of
// the reader, not an unknown row: the reading contradicts itself. Decision
// section 8.1 states that a known counterexample in another row must not hide
// it, because then an implementation check would pass over a reading that is
// malformed.
//
// tei.number.every-at-most compared as it walked, so it returned TriNo at the
// first row that refuted the contract and never looked at a malformed row
// further along. Measured before the fix: a present-without-value row beside a
// counterexample answered satisfied-shaped TriNo with no error at all (#4359).
func TestEveryAtMostReportsAPresentRowWithoutAValue(t *testing.T) {
	implementation := everyAtMostForContract(t)
	broken := validationeval.NumberReading{State: validationeval.ValuePresent}
	present := func(value int64) validationeval.NumberReading {
		return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(value, 1)}
	}

	for _, tc := range []struct {
		name           string
		values, limits []validationeval.NumberReading
	}{
		{
			// The counterexample is read first and the malformed row follows.
			name:   "a counterexample then a row present without a value",
			values: []validationeval.NumberReading{present(9), broken},
			limits: []validationeval.NumberReading{present(1), present(1)},
		},
		{
			// The malformed row is read first; the answer must not depend on
			// the order either.
			name:   "a row present without a value then a counterexample",
			values: []validationeval.NumberReading{broken, present(9)},
			limits: []validationeval.NumberReading{present(1), present(1)},
		},
		{
			// A limit reported as present without a value is the same
			// violation on the other input.
			name:   "a limit present without a value",
			values: []validationeval.NumberReading{present(9), present(1)},
			limits: []validationeval.NumberReading{present(1), {State: validationeval.ValuePresent}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := implementation.NumberRows(map[string][]validationeval.NumberReading{
				"value": tc.values, "limit": tc.limits,
			}); err == nil {
				t.Fatal("a row reported as present without a value was accepted")
			}
		})
	}

	// A row that certainly holds no value is not a violation: it stays an
	// unknown row, and a counterexample beside it still decides.
	null := validationeval.NumberReading{State: validationeval.ValueNull}
	answer, err := implementation.NumberRows(map[string][]validationeval.NumberReading{
		"value": {present(9), null},
		"limit": {present(1), null},
	})
	if err != nil {
		t.Fatalf("a row that holds no value is not a contract violation: %v", err)
	}
	if answer != validationeval.TriNo {
		t.Fatalf("answer=%d, want TriNo: a counterexample was read", int(answer))
	}
}

func everyAtMostForContract(t *testing.T) validationeval.NamedImplementation {
	t.Helper()
	for _, candidate := range validationeval.BuiltinImplementations() {
		if candidate.ID == "tei.number.every-at-most" {
			return candidate
		}
	}
	t.Fatal("the shipped implementation tei.number.every-at-most is missing")
	return validationeval.NamedImplementation{}
}
