package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic/validationeval"
)

// A witnessed counterexample settles a row-scoped rule "whatever else is
// unknown" (decision section 6, and §15.2's table). The row's position must not
// change that: a row that holds no value is unknown, not a counterexample, so it
// cannot outrank one that was read.
//
// tei.number.every-at-most returned at the FIRST row it could not compare,
// before looking at later rows. Measured before the fix:
//
//	counterexample then null        → TriNo
//	null then counterexample        → TriMaybe
//
// The same two rows, in the other order, gave different answers — and the
// declaration's `refutes on some rows` then either applied or did not.
func TestEveryAtMostAnswersTheSameWhateverTheRowOrder(t *testing.T) {
	implementation := everyAtMost(t)

	present := func(value int64) validationeval.NumberReading {
		return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(value, 1)}
	}
	var (
		null       = validationeval.NumberReading{State: validationeval.ValueNull}
		notFetched = validationeval.NumberReading{State: validationeval.ValueNotFetched}
	)

	for _, tc := range []struct {
		name string
		// The two rows, each stating its value and its limit.
		value1, limit1 validationeval.NumberReading
		value2, limit2 validationeval.NumberReading
	}{
		{
			// 5 > 4 is a counterexample; the other row certainly holds no value.
			name:   "a counterexample and a row that holds no value",
			value1: present(5), limit1: present(4),
			value2: null, limit2: null,
		},
		{
			// The same pair, with the second row never read.
			name:   "a counterexample and a row that was not read",
			value1: present(5), limit1: present(4),
			value2: notFetched, limit2: notFetched,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The counterexample comes first.
			answer, err := implementation.NumberRows(map[string][]validationeval.NumberReading{
				"value": {tc.value1, tc.value2},
				"limit": {tc.limit1, tc.limit2},
			})
			if err != nil {
				t.Fatalf("rows are well formed: %v", err)
			}
			// The same rows in the other order.
			reversed, err := implementation.NumberRows(map[string][]validationeval.NumberReading{
				"value": {tc.value2, tc.value1},
				"limit": {tc.limit2, tc.limit1},
			})
			if err != nil {
				t.Fatalf("rows are well formed: %v", err)
			}
			if answer != reversed {
				t.Fatalf("the same two rows answered %d and %d depending on order; a counterexample settles the rule whatever else is unknown",
					int(answer), int(reversed))
			}
			// A counterexample was read, so the answer is certain.
			if answer != validationeval.TriNo {
				t.Fatalf("answer=%d, want TriNo: a row read as 5 against its limit 4 breaks the rule", int(answer))
			}
		})
	}

	// Without any counterexample, a row that holds no value still leaves the
	// question open: unknown is not satisfied.
	answer, err := implementation.NumberRows(map[string][]validationeval.NumberReading{
		"value": {present(2), null},
		"limit": {present(10), null},
	})
	if err != nil {
		t.Fatalf("rows are well formed: %v", err)
	}
	if answer != validationeval.TriMaybe {
		t.Fatalf("answer=%d, want TriMaybe for a row that holds no value and no counterexample", int(answer))
	}

	// Every row read and within its limit is the certain yes.
	answer, err = implementation.NumberRows(map[string][]validationeval.NumberReading{
		"value": {present(2), present(3)},
		"limit": {present(10), present(3)},
	})
	if err != nil {
		t.Fatalf("rows are well formed: %v", err)
	}
	if answer != validationeval.TriYes {
		t.Fatalf("answer=%d, want TriYes when every row is within its limit", int(answer))
	}
}

// A row reported as present without a value is a reader contract violation and
// stays an error; the known counterexample in another row must not hide it.
func TestEveryAtMostStillRefusesAPresentRowWithoutAValue(t *testing.T) {
	implementation := everyAtMost(t)
	broken := validationeval.NumberReading{State: validationeval.ValuePresent}
	_, err := implementation.NumberRows(map[string][]validationeval.NumberReading{
		"value": {validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(9, 1)}, broken},
		"limit": {validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}, validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}},
	})
	if err == nil {
		t.Fatal("a present row without a value was accepted because another row held a counterexample")
	}
}

func everyAtMost(t *testing.T) validationeval.NamedImplementation {
	t.Helper()
	for _, candidate := range validationeval.BuiltinImplementations() {
		if candidate.ID == "tei.number.every-at-most" {
			return candidate
		}
	}
	t.Fatal("the shipped implementation tei.number.every-at-most is missing")
	return validationeval.NamedImplementation{}
}
