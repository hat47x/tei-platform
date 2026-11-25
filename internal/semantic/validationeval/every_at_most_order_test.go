package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic/validationeval"
)

// tei.number.every-at-most decides "every row's value is at most its limit". A
// row whose value or limit was not read, or holds no value, cannot be compared,
// but it cannot rescue a comparison that already failed: a witnessed
// counterexample is a counterexample whether the rows that cannot be compared
// come before or after it. Decision section 8 states this for the operator and
// section 15.2 says the same for the evaluator's composition.
//
// The expectations below are written from that meaning and from the rows given
// here, not read back from the implementation.

// everyAtMost returns the shipped implementation, or fails when it is missing.
func everyAtMost(t *testing.T) validationeval.NamedImplementation {
	t.Helper()
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID == "tei.number.every-at-most" {
			return available
		}
	}
	t.Fatal("the shipped tei.number.every-at-most is missing")
	return validationeval.NamedImplementation{}
}

func row(value string) validationeval.NumberReading {
	if value == "" {
		return validationeval.NumberReading{State: validationeval.ValueNull}
	}
	if value == "?" {
		return validationeval.NumberReading{State: validationeval.ValueNotFetched}
	}
	parsed, ok := new(big.Rat).SetString(value)
	if !ok {
		panic("not a decimal: " + value)
	}
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: parsed}
}

func columns(values, limits []string) map[string][]validationeval.NumberReading {
	return map[string][]validationeval.NumberReading{
		"value": mapRows(values),
		"limit": mapRows(limits),
	}
}

func mapRows(texts []string) []validationeval.NumberReading {
	out := make([]validationeval.NumberReading, 0, len(texts))
	for _, text := range texts {
		out = append(out, row(text))
	}
	return out
}

// A counterexample that was read settles the answer whatever unread rows
// surround it. Before this, the implementation returned TriMaybe at the first
// row it could not compare, so the answer depended on the order of the rows.
func TestEveryAtMostKeepsACounterexampleWhereverItWasRead(t *testing.T) {
	cases := []struct {
		name   string
		values []string
		limits []string
		want   validationeval.Tri
	}{
		// The counterexample comes first, then a row that holds no value.
		{name: "counterexample then a null row", values: []string{"5", ""}, limits: []string{"4", "10"}, want: validationeval.TriNo},
		// The same rows in the other order.
		{name: "a null row then the counterexample", values: []string{"", "5"}, limits: []string{"10", "4"}, want: validationeval.TriNo},
		// A row that was not read cannot hide the counterexample either.
		{name: "counterexample then an unread row", values: []string{"5", "?"}, limits: []string{"4", "10"}, want: validationeval.TriNo},
		{name: "an unread row then the counterexample", values: []string{"?", "5"}, limits: []string{"10", "4"}, want: validationeval.TriNo},
		// Without a counterexample the unread or absent row does leave the
		// answer undecided: comparing fewer rows is not holding everywhere.
		{name: "a null row, nothing above", values: []string{"", "5"}, limits: []string{"10", "10"}, want: validationeval.TriMaybe},
		{name: "an unread row, nothing above", values: []string{"?", "5"}, limits: []string{"10", "10"}, want: validationeval.TriMaybe},
		// Every row was read and none is above: the rule holds.
		{name: "every row at most", values: []string{"2", "5"}, limits: []string{"10", "5"}, want: validationeval.TriYes},
	}
	implementation := everyAtMost(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := implementation.NumberRows(columns(tc.values, tc.limits))
			if err != nil {
				t.Fatalf("err=%v", err)
			}
			if got != tc.want {
				t.Fatalf("got %d, want %d", int(got), int(tc.want))
			}
		})
	}
}

// The same rows must give the same answer when they are given in the other
// order, so a table of rows states one meaning rather than one per order.
func TestEveryAtMostAnswersTheSameInEitherOrder(t *testing.T) {
	for name, rows := range map[string]struct {
		values []string
		limits []string
	}{
		"counterexample and an unread row": {values: []string{"5", "?"}, limits: []string{"4", "10"}},
		"counterexample and a null row":    {values: []string{"5", ""}, limits: []string{"4", "10"}},
		"null row and an unread row":       {values: []string{"", "?"}, limits: []string{"10", "4"}},
	} {
		reversedValues := []string{rows.values[1], rows.values[0]}
		reversedLimits := []string{rows.limits[1], rows.limits[0]}
		implementation := everyAtMost(t)
		first, err := implementation.NumberRows(columns(rows.values, rows.limits))
		if err != nil {
			t.Fatalf("%s: err=%v", name, err)
		}
		second, err := implementation.NumberRows(columns(reversedValues, reversedLimits))
		if err != nil {
			t.Fatalf("%s reversed: err=%v", name, err)
		}
		if first != second {
			t.Fatalf("%s: order changed the answer (%d then %d)", name, int(first), int(second))
		}
	}
}
