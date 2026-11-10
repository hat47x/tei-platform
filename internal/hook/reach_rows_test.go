package hook_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// rowReachWorld is a record pattern with a row-scoped hook that reads one attribute.
func rowReachWorld(uses bool) ir.World {
	declaration := application.Hook{
		Phase: "before-update",
		Scope: "row",
		Reads: []core.CanonicalID{"sales.order.amount"},
	}
	if uses {
		declaration.Uses = []core.CanonicalID{"sales.check_limit"}
	}
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:          "order_entry.order_record",
				Kind:        application.PatternRecord,
				Data:        "sales.order",
				Interaction: &application.Interaction{Hooks: []application.Hook{declaration}},
			}},
		}},
	}
}

func rowReachRows(values ...string) []map[core.CanonicalID]string {
	out := make([]map[core.CanonicalID]string, 0, len(values))
	for _, value := range values {
		out = append(out, map[core.CanonicalID]string{"sales.order.amount": value})
	}
	return out
}

// ruleImplementation refuses when the amount is not positive, so a row can be made
// to fail while the others pass.
type ruleImplementation struct{ calls int }

func (i *ruleImplementation) Run(_ context.Context, invocation hook.Invocation) error {
	i.calls++
	value := invocation.Reads["sales.order.amount"]
	if value == "bad" {
		return &hook.NotSatisfied{Reason: "the amount is not a number", Outcome: hook.OutcomeValueRejected}
	}
	return nil
}

// TestRunReachableAppliesRowScopeToEveryRow checks that RunReachable applies a
// row-scoped hook once per row when the runner has the row set, instead of skipping
// the phase.
func TestRunReachableAppliesRowScopeToEveryRow(t *testing.T) {
	implementation := &ruleImplementation{}
	runner := hook.Runner{
		World:          rowReachWorld(false),
		Implementation: implementation,
		Rows:           rowReachRows("10", "20", "30"),
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, hook.ReferenceReach{})
	if err != nil {
		t.Fatal(err)
	}
	if implementation.calls != 3 {
		t.Fatalf("calls=%d, want one per row", implementation.calls)
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(report.Skipped) != 0 {
		t.Fatalf("skipped=%v", report.Skipped)
	}
}

// TestRunReachableStopsAtTheRefusedRow checks that a refused row stops the phase and
// the report names the row, so a partly-applied page never reads as passed.
func TestRunReachableStopsAtTheRefusedRow(t *testing.T) {
	implementation := &ruleImplementation{}
	runner := hook.Runner{
		World:          rowReachWorld(false),
		Implementation: implementation,
		Rows:           rowReachRows("10", "bad", "30"),
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, hook.ReferenceReach{})
	if err == nil {
		t.Fatal("the refusal was reported as success")
	}
	if implementation.calls != 2 {
		t.Fatalf("calls=%d, want 2 (the third row must not run)", implementation.calls)
	}
	if report.Refused != "before-update" {
		t.Fatalf("refused=%q", report.Refused)
	}
	if report.Outcome != hook.OutcomeValueRejected {
		t.Fatalf("outcome=%q", report.Outcome)
	}
	if !strings.Contains(report.Reasons["before-update"], "row 1") {
		t.Fatalf("the reason does not name the row: %q", report.Reasons["before-update"])
	}
	for _, phase := range report.Executed {
		if phase == "before-update" {
			t.Fatal("a refused phase was recorded as executed")
		}
	}
}

// TestRunReachableWithoutRowsSkipsARowScope checks that a row-scoped hook with no row
// set is skipped with a reason, not run once for the record.
func TestRunReachableWithoutRowsSkipsARowScope(t *testing.T) {
	implementation := &ruleImplementation{}
	runner := hook.Runner{World: rowReachWorld(false), Implementation: implementation}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, hook.ReferenceReach{})
	if err != nil {
		t.Fatal(err)
	}
	if implementation.calls != 0 {
		t.Fatalf("the hook ran without a row set: %d calls", implementation.calls)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "before-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if !strings.Contains(report.Reasons["before-update"], "row set") {
		t.Fatalf("the reason does not name the missing row set: %q", report.Reasons["before-update"])
	}
}

// TestRunReachableRefusesRowScopeWithAnUnresolvableCall checks the fail-closed side
// that remains: a row-scoped hook that declares a call which cannot be resolved is
// refused, and the rule does not run for that row.
//
// The blanket refusal this test used to pin is gone: a row-scoped `uses` now makes
// its call once per row. What stays fail-closed is a declaration whose call has
// nowhere to go.
func TestRunReachableRefusesRowScopeWithAnUnresolvableCall(t *testing.T) {
	implementation := &ruleImplementation{}
	runner := hook.Runner{
		World:          rowReachWorld(true),
		Implementation: implementation,
		Rows:           rowReachRows("10", "20"),
	}
	_, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, hook.ReferenceReach{})
	if err == nil || !strings.Contains(err.Error(), "HOOK_PATTERN_NOT_DECLARED") {
		t.Fatalf("err=%v", err)
	}
	if implementation.calls != 0 {
		t.Fatalf("the rule ran for a row whose call could not be made: %d calls", implementation.calls)
	}
}

// TestRunRoutesARowScopeToRunRows checks that Run applies the row set it was given
// rather than reporting the scope as needing rows.
func TestRunRoutesARowScopeToRunRows(t *testing.T) {
	implementation := &rowImplementation{}
	runner := hook.Runner{
		World:          rowScopeWorld("row"),
		Implementation: implementation,
		Rows:           rowsOf("10", "20"),
	}
	if err := runner.Run(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}); err != nil {
		t.Fatal(err)
	}
	if len(implementation.seen) != 2 {
		t.Fatalf("calls=%d, want one per row", len(implementation.seen))
	}
}
