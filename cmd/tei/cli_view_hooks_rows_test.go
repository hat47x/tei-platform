package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// rowHookWorld declares a pattern with one row-scoped hook. The CLI's hook entry
// is the one under test: the runner side (hook.RunRows) was already tested, and
// what was missing was a caller that supplies the row set at all.
func rowHookWorld() ir.World {
	return hookViewWorld(application.Hook{
		Phase: "before-update",
		Scope: "row",
		Reads: []core.CanonicalID{"sales.order.amount"},
	})
}

// rowHookRows is the row set a caller hands over, one map per row. Each row also
// carries an attribute the hook does not declare, so a test can show the runner
// narrows each row to the declared reads instead of forwarding it whole.
func rowHookRows(values ...string) []map[core.CanonicalID]string {
	rows := make([]map[core.CanonicalID]string, 0, len(values))
	for _, value := range values {
		rows = append(rows, map[core.CanonicalID]string{
			"sales.order.amount": value,
			"sales.order.note":   "not-declared",
		})
	}
	return rows
}

// countingRowRule records the value each invocation was decided with, and leaves
// the decision to an inner rule when one is set.
type countingRowRule struct {
	inner     hook.Implementation
	values    []string
	sawUnread bool
}

func (r *countingRowRule) Run(ctx context.Context, invocation hook.Invocation) error {
	r.values = append(r.values, invocation.Reads["sales.order.amount"])
	if _, ok := invocation.Reads["sales.order.note"]; ok {
		r.sawUnread = true
	}
	if r.inner != nil {
		return r.inner.Run(ctx, invocation)
	}
	return nil
}

// TestRunDeclaredHooksWithRowsAppliesTheRowScopeToEveryRow is the wiring check:
// when the caller hands the CLI hook entry a row set, the row-scoped hook runs
// once per row, with that row's declared values, and the phase is recorded as
// executed rather than skipped.
func TestRunDeclaredHooksWithRowsAppliesTheRowScopeToEveryRow(t *testing.T) {
	rule := &countingRowRule{}
	report, err := runDeclaredHooksWithRows(
		context.Background(),
		rowHookWorld(),
		"order_entry",
		"order_entry.order_record",
		// The record inputs carry a different amount on purpose: a row-scoped
		// hook must decide on the row's value, not on the record's.
		map[string]string{"sales.order.amount": "999"},
		rowHookRows("10", "20", "30"),
		rule,
	)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no hook report for a pattern that declares a hook")
	}
	if len(rule.values) != 3 || rule.values[0] != "10" || rule.values[1] != "20" || rule.values[2] != "30" {
		t.Fatalf("the hook was not applied once per row: %v", rule.values)
	}
	if rule.sawUnread {
		t.Fatal("a row reached the rule with an attribute the declaration does not read")
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(report.Skipped) != 0 || len(report.NotEvaluated) != 0 {
		t.Fatalf("a row-scoped phase with rows was not applied: skipped=%v notEvaluated=%v", report.Skipped, report.NotEvaluated)
	}
}

// TestRunDeclaredHooksWithRowsNamesTheRefusedRow checks that a refusal at a row
// stops the page there, and that the row number survives to the CLI's caller:
// the returned error names the row, and the report's reason does too.
func TestRunDeclaredHooksWithRowsNamesTheRefusedRow(t *testing.T) {
	rule := &countingRowRule{inner: hook.ReferenceRule{Positive: []core.CanonicalID{"sales.order.amount"}}}
	report, err := runDeclaredHooksWithRows(
		context.Background(),
		rowHookWorld(),
		"order_entry",
		"order_entry.order_record",
		nil,
		rowHookRows("10", "0", "30"),
		rule,
	)
	if err == nil {
		t.Fatal("a non-positive row did not stop the run")
	}
	var refusal *hook.RowRefusal
	if !errors.As(err, &refusal) {
		t.Fatalf("the refusal does not carry the row: %v", err)
	}
	if refusal.Row != 1 {
		t.Fatalf("row=%d, want the second row (index 1)", refusal.Row)
	}
	if len(rule.values) != 2 {
		t.Fatalf("the rule ran for %v, want the two rows up to the refusal", rule.values)
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("report=%+v", report)
	}
	if report.Outcome != hook.OutcomeValueRejected {
		t.Fatalf("outcome=%q", report.Outcome)
	}
	if reason := report.Reasons["before-update"]; !strings.Contains(reason, "row 1") {
		t.Fatalf("the reason does not name the refused row: %q", reason)
	}
	for _, phase := range report.Executed {
		if phase == "before-update" {
			t.Fatal("a refused phase was recorded as executed")
		}
	}
}

// TestRunDeclaredHooksWithoutRowsSkipsTheRowScope pins the other side: the CLI's
// own command path supplies no row set, so the phase is skipped with a reason and
// the hook never runs once for the record. The wiring must not turn "no rows"
// into "one row".
func TestRunDeclaredHooksWithoutRowsSkipsTheRowScope(t *testing.T) {
	rule := &countingRowRule{}
	report, err := runDeclaredHooksWithRule(
		context.Background(),
		rowHookWorld(),
		"order_entry",
		"order_entry.order_record",
		map[string]string{"sales.order.amount": "10"},
		rule,
	)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rule.values) != 0 {
		t.Fatalf("a row-scoped hook ran without a row set: %v", rule.values)
	}
	if report == nil {
		t.Fatal("no hook report")
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "before-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if reason := report.Reasons["before-update"]; !strings.Contains(reason, "row set") {
		t.Fatalf("the reason does not name the missing row set: %q", reason)
	}
}
