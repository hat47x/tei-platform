package derivationbatch

import (
	"errors"
	"testing"

	"tei/internal/semantic/derivationref"
)

func missingInputError() error {
	_, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Missing},
		derivationref.Input{State: derivationref.Present, Decimal: "300"},
	)
	return err
}

func operatorError() error {
	_, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Present, Decimal: "abc"},
		derivationref.Input{State: derivationref.Present, Decimal: "300"},
	)
	return err
}

func value(decimal string) derivationref.Input {
	return derivationref.Input{State: derivationref.Present, Decimal: decimal}
}

// A value, B unavailable, C zero, D a declared NULL.
func fixture() []Row {
	return []Row{
		{ID: "A", Result: value("600")},
		{ID: "B", Err: missingInputError()},
		{ID: "C", Result: value("0")},
		{ID: "D", Result: derivationref.Input{State: derivationref.Null}},
	}
}

func evaluatedFixture() []Row {
	rows := fixture()
	return []Row{rows[0], rows[2], rows[3]}
}

func TestD13AllEvaluableIsComplete(t *testing.T) {
	decision, err := Plan(DisplayOnly, false, evaluatedFixture())
	if err != nil || !decision.Complete || decision.Partial {
		t.Fatalf("all-evaluable batch: %+v, %v", decision, err)
	}
	if len(decision.Rows) != 3 || decision.Rows[0].Value != "600" || decision.Rows[1].Value != "0" ||
		decision.Rows[2].Outcome != RowNull {
		t.Fatalf("projected rows wrong: %+v", decision.Rows)
	}
}

func TestD13PartialDisplayKeepsEveryRowAndMarksUnevaluated(t *testing.T) {
	decision, err := Plan(DisplayOnly, true, fixture())
	if err != nil {
		t.Fatalf("allowed partial display failed: %v", err)
	}
	if decision.Complete || !decision.Partial || len(decision.Rows) != 4 {
		t.Fatalf("partial projection wrong: %+v", decision)
	}
	if len(decision.Unevaluated) != 1 || decision.Unevaluated[0] != "B" {
		t.Fatalf("unevaluated rows wrong: %+v", decision.Unevaluated)
	}
	if decision.Rows[1].Outcome != RowInputUnavailable || decision.Rows[1].Value != "" {
		t.Fatalf("unevaluated row was not marked: %+v", decision.Rows[1])
	}
}

func TestD13DisplayWithoutAllowanceFailsWhole(t *testing.T) {
	decision, err := Plan(DisplayOnly, false, fixture())
	if !errors.Is(err, ErrIncomplete) {
		t.Fatalf("unallowed partial display did not fail: %v", err)
	}
	if decision.Complete || decision.Partial || len(decision.Rows) != 0 {
		t.Fatalf("failed batch leaked a partial result: %+v", decision)
	}
}

func TestD13SetOperationsFailOnUnevaluated(t *testing.T) {
	for _, usage := range []Usage{Filter, Sort, Aggregate, Group, CountDerived, PageCursor} {
		for _, allowPartial := range []bool{false, true} {
			t.Run(string(usage), func(t *testing.T) {
				decision, err := Plan(usage, allowPartial, fixture())
				if !errors.Is(err, ErrIncomplete) {
					t.Fatalf("%s did not fail with an unevaluated row: %v", usage, err)
				}
				if decision.Complete || decision.Partial || len(decision.Rows) != 0 {
					t.Fatalf("%s returned a partial set result: %+v", usage, decision)
				}
			})
		}
	}
}

func TestD13SourceRowCountIsUnaffected(t *testing.T) {
	decision, err := Plan(CountSourceRows, false, fixture())
	if err != nil || !decision.Complete {
		t.Fatalf("source row count depended on the derived column: %+v, %v", decision, err)
	}
	if len(decision.Rows) != 4 {
		t.Fatalf("source row count wrong: %d", len(decision.Rows))
	}
}

func TestD13UnevaluatedIsNotZeroOrNull(t *testing.T) {
	decision, err := Plan(DisplayOnly, true, fixture())
	if err != nil {
		t.Fatal(err)
	}
	row := decision.Rows[1]
	if row.Outcome == RowNull || row.Outcome == RowValue || row.Value == "0" {
		t.Fatalf("unevaluated row was flattened to NULL or 0: %+v", row)
	}
}

func TestD13RowOutcomeClassification(t *testing.T) {
	cases := []struct {
		name  string
		value derivationref.Input
		err   error
		want  RowOutcome
	}{
		{"value", value("600"), nil, RowValue},
		{"null", derivationref.Input{State: derivationref.Null}, nil, RowNull},
		{"missing", derivationref.Input{}, missingInputError(), RowInputUnavailable},
		{"operator", derivationref.Input{}, operatorError(), RowEvaluationFailed},
	}
	seen := map[RowOutcome]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RowOutcomeOf(tc.value, tc.err)
			if got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
			seen[got] = true
		})
	}
	if len(seen) != 4 {
		t.Fatalf("outcomes were collapsed: %v", seen)
	}
}

func TestD13UnknownUsageAndEmptyRowIDRejected(t *testing.T) {
	if _, err := Plan(Usage("nonsense"), false, evaluatedFixture()); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("unknown usage accepted: %v", err)
	}
	if _, err := Plan(DisplayOnly, false, []Row{{Result: value("1")}}); !errors.Is(err, ErrInvalidRow) {
		t.Fatalf("empty row id accepted: %v", err)
	}
}
