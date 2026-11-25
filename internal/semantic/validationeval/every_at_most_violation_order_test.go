package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/semantic/validationeval"
)

// The shipped tei.number.every-at-most is asked about rows a reader produced.
// A reader that reports a row as ValuePresent with no value has broken its own
// contract: that is not "the row holds no value", it is a reading that
// contradicts itself. Decision section 8.1 says a known counterexample must not
// hide it, so the violation is reported whatever order the rows were read in.
//
// Before this, the violation was looked at inside the comparison loop, after
// the counterexample had already answered TriNo: a counterexample in the first
// row hid the violation in the second, and the same rows with the violation
// first answered TriMaybe with an error. The answer depended on row order.
func TestEveryAtMostReportsAContractViolationWhateverTheRowOrder(t *testing.T) {
	shipped := shippedEveryAtMost(t)

	present := validationeval.ValuePresent
	counterexample := validationeval.NumberReading{State: present, Value: big.NewRat(2, 1)}
	violation := validationeval.NumberReading{State: present, Value: nil}
	limit := validationeval.NumberReading{State: present, Value: big.NewRat(1, 1)}

	columns := func(values []validationeval.NumberReading) map[string][]validationeval.NumberReading {
		return map[string][]validationeval.NumberReading{
			"value": values,
			"limit": {limit, limit},
		}
	}

	cases := []struct {
		name   string
		values []validationeval.NumberReading
		// wantIndex is the row the error must name, so the failure points at
		// the row the reader broke rather than at the one that was compared.
		wantIndex string
	}{
		{"counterexample first", []validationeval.NumberReading{counterexample, violation}, "row 1"},
		{"violation first", []validationeval.NumberReading{violation, counterexample}, "row 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answer, err := shipped.NumberRows(columns(tc.values))
			if err == nil {
				t.Fatalf("answer=%v, want a contract violation reported", answer)
			}
			if answer != validationeval.TriMaybe {
				t.Fatalf("answer=%v, want TriMaybe: a contradictory reading is not a decision", answer)
			}
			if !strings.Contains(err.Error(), "present without a value") ||
				!strings.Contains(err.Error(), tc.wantIndex) {
				t.Fatalf("error=%q, want it to name %s", err, tc.wantIndex)
			}
		})
	}
}

// shippedEveryAtMost is the implementation this binary carries, by the name a
// composition selects it by. The test asks the shipped one on purpose: a local
// stand-in would prove nothing about what a Workspace gets.
func shippedEveryAtMost(t *testing.T) validationeval.NamedImplementation {
	t.Helper()
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID == "tei.number.every-at-most" {
			return available
		}
	}
	t.Fatal("the shipped tei.number.every-at-most is missing")
	return validationeval.NamedImplementation{}
}
