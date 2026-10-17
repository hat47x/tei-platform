package batchcompleteness

import "testing"

// The fixture follows Data Derivation D13 (#1724).
//
//	row | quantity | price     | derived amount
//	A   | 2        | 300       | 600
//	B   | 3        | (missing) | input unavailable
//	C   | 0        | 300       | 0
//	D   | null     | 200       | declared null
func d13Rows() []Row {
	return []Row{
		{ID: "A", Quantity: Operand{State: "present", Decimal: "2"}, Price: Operand{State: "present", Decimal: "300"}},
		{ID: "B", Quantity: Operand{State: "present", Decimal: "3"}, Price: Operand{State: "missing"}},
		{ID: "C", Quantity: Operand{State: "present", Decimal: "0"}, Price: Operand{State: "present", Decimal: "300"}},
		{ID: "D", Quantity: Operand{State: "null"}, Price: Operand{State: "present", Decimal: "200"}},
	}
}

func rowsByID(t *testing.T, report Report) map[string]RowResult {
	t.Helper()
	out := map[string]RowResult{}
	for _, row := range report.Rows {
		out[row.ID] = row
	}
	return out
}

func TestDisplayPartialProjectionIsAllowedExplicitly(t *testing.T) {
	report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpDisplay, PartialProjectionAllowed: true}})
	if report.Complete || !report.Partial || report.Code != CodePartialProjection {
		t.Fatalf("report=%+v want partial projection", report)
	}
	if len(report.Unresolved) != 1 || report.Unresolved[0] != "B" {
		t.Fatalf("unresolved=%v want [B]", report.Unresolved)
	}
	rows := rowsByID(t, report)
	if rows["A"].State != RowPresent || rows["A"].Amount != "600" {
		t.Fatalf("A=%+v want present 600", rows["A"])
	}
	if rows["C"].State != RowPresent || rows["C"].Amount != "0" {
		t.Fatalf("C=%+v want present 0 (zero is not null)", rows["C"])
	}
	if rows["D"].State != RowNull {
		t.Fatalf("D=%+v want null", rows["D"])
	}
	if rows["B"].State != RowInputUnavailable || rows["B"].Amount != "" {
		t.Fatalf("B=%+v want input-unavailable without a value", rows["B"])
	}
}

func TestDisplayWithoutPermissionFailsWhole(t *testing.T) {
	report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpDisplay}})
	if report.Complete || report.Partial || report.Code != CodeIncompleteDisplay {
		t.Fatalf("report=%+v want INCOMPLETE_DISPLAY", report)
	}
}

func TestSetOperationsRejectUnevaluableRows(t *testing.T) {
	for _, op := range []string{OpFilter, OpSort, OpAggregate, OpCountDerived} {
		report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: op, PartialProjectionAllowed: true}})
		if report.Complete || report.Partial || report.Code != CodeIncompleteSetOperation {
			t.Fatalf("op=%s report=%+v want INCOMPLETE_SET_OPERATION", op, report)
		}
		if len(report.Unresolved) != 1 || report.Unresolved[0] != "B" {
			t.Fatalf("op=%s unresolved=%v want [B]", op, report.Unresolved)
		}
	}
}

func TestPlainRowCountIsUnaffectedByDerivationFailure(t *testing.T) {
	report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: OpCountRows}})
	if !report.Complete || report.Code != CodeOK {
		t.Fatalf("report=%+v want complete", report)
	}
	if report.Count == nil || *report.Count != 4 {
		t.Fatalf("count=%v want 4", report.Count)
	}
}

func TestCompleteSetOperationWhenAllRowsEvaluate(t *testing.T) {
	rows := []Row{
		{ID: "A", Quantity: Operand{State: "present", Decimal: "2"}, Price: Operand{State: "present", Decimal: "300"}},
		{ID: "D", Quantity: Operand{State: "null"}, Price: Operand{State: "present", Decimal: "200"}},
	}
	report := Evaluate(Input{Rows: rows, Query: Query{Op: OpAggregate}})
	if !report.Complete || report.Partial || report.Code != CodeOK {
		t.Fatalf("report=%+v want complete (null is not a failure)", report)
	}
	if len(report.Unresolved) != 0 {
		t.Fatalf("unresolved=%v want empty", report.Unresolved)
	}
}

func TestUnsupportedOperationIsRejected(t *testing.T) {
	report := Evaluate(Input{Rows: d13Rows(), Query: Query{Op: "join"}})
	if report.Complete || report.Code != CodeUnsupported {
		t.Fatalf("report=%+v want UNSUPPORTED_OPERATION", report)
	}
}
