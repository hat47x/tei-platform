package batchcompleteness

import (
	"reflect"
	"testing"
)

// The same D13 fixture is used for the sort/limit and SUM negatives. B has no
// price, so its derived amount is unknown.
func TestSortDescLimitOneDoesNotClaimMax(t *testing.T) {
	report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpSort, PartialProjectionAllowed: true}})
	if report.Complete || report.Partial || report.Code != CodeIncompleteSetOperation {
		t.Fatalf("report=%+v want INCOMPLETE_SET_OPERATION", report)
	}
	if report.Count != nil {
		t.Fatalf("sort must not return a count: %v", *report.Count)
	}
	if len(report.Unresolved) != 1 || report.Unresolved[0] != "B" {
		t.Fatalf("unresolved=%v want [B]", report.Unresolved)
	}
	if row := rowsByID(t, report)["B"]; row.Amount != "" || row.State != RowInputUnavailable {
		t.Fatalf("B=%+v: an unknown maximum must not carry a value", row)
	}
}

func TestSumDoesNotSubstituteZeroOrNullForMissingRow(t *testing.T) {
	report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpAggregate}})
	if report.Complete || report.Code != CodeIncompleteSetOperation {
		t.Fatalf("report=%+v want INCOMPLETE_SET_OPERATION", report)
	}
	row := rowsByID(t, report)["B"]
	if row.Amount != "" {
		t.Fatalf("B got a substituted amount %q", row.Amount)
	}
	if row.State != RowInputUnavailable {
		t.Fatalf("B state=%q, want input-unavailable (not null or zero)", row.State)
	}
	for _, candidate := range report.Rows {
		if candidate.ID == "B" && candidate.State == RowNull {
			t.Fatal("missing input was turned into a normal NULL")
		}
	}
}

func TestFilterDoesNotSilentlyDropRow(t *testing.T) {
	report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpFilter}})
	if report.Complete || report.Code != CodeIncompleteSetOperation {
		t.Fatalf("report=%+v want INCOMPLETE_SET_OPERATION", report)
	}
	if len(report.Unresolved) != 1 || report.Unresolved[0] != "B" {
		t.Fatalf("unresolved=%v want [B]", report.Unresolved)
	}
	if report.Count != nil {
		t.Fatalf("filter must not return a partial count: %v", *report.Count)
	}
}

func TestCountDerivedRejectsPartialCount(t *testing.T) {
	report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpCountDerived}})
	if report.Complete || report.Code != CodeIncompleteSetOperation {
		t.Fatalf("report=%+v want INCOMPLETE_SET_OPERATION", report)
	}
	if report.Count != nil {
		t.Fatalf("derived count must not be returned as complete: %v", *report.Count)
	}
}

func TestTypeErrorIsDistinctFromInputUnavailable(t *testing.T) {
	rows := append(d13Rows(), Row{
		ID:       "E",
		Quantity: Operand{State: "present", Decimal: "abc"},
		Price:    Operand{State: "present", Decimal: "300"},
	})
	report := Evaluate(Input{Rows: rows, Query: Query{Op: OpAggregate}})
	got := rowsByID(t, report)["E"]
	if got.State != RowTypeError {
		t.Fatalf("E state=%q, want %q", got.State, RowTypeError)
	}
	if got.State == RowInputUnavailable {
		t.Fatal("a type failure was reported as an unreadable input")
	}
	if got.Amount != "" {
		t.Fatalf("E got a value %q", got.Amount)
	}
	if report.Complete || report.Code != CodeIncompleteSetOperation {
		t.Fatalf("report=%+v want INCOMPLETE_SET_OPERATION", report)
	}
}

func TestExecutionUnavailableIsDistinct(t *testing.T) {
	rows := append(d13Rows(), Row{
		ID:       "F",
		Quantity: Operand{State: "unavailable"},
		Price:    Operand{State: "present", Decimal: "300"},
	})
	report := Evaluate(Input{Rows: rows, Query: Query{Op: OpDisplay, PartialProjectionAllowed: true}})
	got := rowsByID(t, report)["F"]
	if got.State != RowExecutionFailed {
		t.Fatalf("F state=%q, want %q", got.State, RowExecutionFailed)
	}
	if got.Amount != "" {
		t.Fatalf("F got a value %q", got.Amount)
	}
}

func TestSortAndSumNegativesAreDeterministic(t *testing.T) {
	sort1 := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpSort}})
	sort2 := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpSort}})
	if !reflect.DeepEqual(sort1, sort2) {
		t.Fatalf("sort is not deterministic: %+v != %+v", sort1, sort2)
	}
	sum1 := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpAggregate}})
	sum2 := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpAggregate}})
	if !reflect.DeepEqual(sum1, sum2) {
		t.Fatalf("SUM is not deterministic: %+v != %+v", sum1, sum2)
	}
}
