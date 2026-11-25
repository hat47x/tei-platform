package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic/validationeval"
)

// A reader that reports a row as present without a value breaks the reader
// contract. That is a fact about the read, not about the rows' order, so the
// answer must not depend on where in the column it appears (decision section
// 8.1). Both orders below hold the same two rows.
//
// The fix that made this hold is on main. These two cases are the ones that
// were measured failing before it: the existing every-at-most tests cover null
// and unread rows but not a present row with no value, so removing the
// contract check again would leave every test green.
func shippedEveryAtMost(t *testing.T) validationeval.NamedImplementation {
	t.Helper()
	for _, implementation := range validationeval.BuiltinImplementations() {
		if implementation.ID == "tei.number.every-at-most" {
			return implementation
		}
	}
	t.Fatal("every-at-most is not shipped")
	return validationeval.NamedImplementation{}
}

func numberRow(value int64) validationeval.NumberReading {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(value, 1)}
}

// A present row without a value: the reader contradicts itself.
func contradictoryRow() validationeval.NumberReading {
	return validationeval.NumberReading{State: validationeval.ValuePresent}
}

func TestEveryAtMostDoesNotHideAContractViolation(t *testing.T) {
	implementation := shippedEveryAtMost(t)
	for _, tc := range []struct {
		name   string
		values []validationeval.NumberReading
		limits []validationeval.NumberReading
	}{
		// A counterexample (5 > 4) at row 0, the violation at row 1.
		{name: "counterexample then the violation",
			values: []validationeval.NumberReading{numberRow(5), contradictoryRow()},
			limits: []validationeval.NumberReading{numberRow(4), numberRow(4)}},
		// The same two rows the other way round.
		{name: "the violation then a counterexample",
			values: []validationeval.NumberReading{contradictoryRow(), numberRow(5)},
			limits: []validationeval.NumberReading{numberRow(4), numberRow(4)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := implementation.NumberRows(map[string][]validationeval.NumberReading{
				"value": tc.values, "limit": tc.limits,
			})
			if err == nil {
				t.Fatalf("a contract violation produced no error (answer %d); the reader's contradiction must be reported whatever order it appears in", int(got))
			}
		})
	}
}

// A violation on the limit side is the same contradiction and is reported too.
func TestEveryAtMostReportsAContractViolationOnTheLimitSide(t *testing.T) {
	implementation := shippedEveryAtMost(t)
	if _, err := implementation.NumberRows(map[string][]validationeval.NumberReading{
		"value": {numberRow(5), numberRow(2)},
		"limit": {numberRow(4), contradictoryRow()},
	}); err == nil {
		t.Fatal("a contract violation on the limit side produced no error")
	}
}
