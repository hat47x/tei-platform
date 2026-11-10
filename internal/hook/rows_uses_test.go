package hook_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"

	"tei/internal/hook"
)

// TestRunRowsRefusesUses checks that a row-scoped hook declaring `uses` is
// refused rather than run with the promise quietly dropped.
//
// The record path keeps a `uses` promise through RunDeclaredCalls, once per
// phase. No per-row equivalent exists, so calling once for the whole page would
// keep the promise for no particular row. Before this rule was shared,
// RunRows ran the hook (and made the call through declaredCaller while
// applyReachableRows refused the same declaration), so the same shape both
// succeeded and failed depending on the entry point.
func TestRunRowsRefusesUses(t *testing.T) {
	implementation := &rowImplementation{}
	runner := hook.Runner{
		World:          rowUsesWorld(),
		Implementation: implementation,
	}
	err := runner.RunRows(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}, []map[core.CanonicalID]string{{"sales.order.amount": "10"}})
	if err == nil {
		t.Fatal("a row-scoped hook with uses was run")
	}
	if !strings.Contains(err.Error(), hook.CodeRowScopeUsesUnsupported) {
		t.Fatalf("err=%v", err)
	}
	if len(implementation.seen) != 0 {
		t.Fatalf("the hook ran anyway: %d calls", len(implementation.seen))
	}
}

// TestRunRowsWithoutUsesStillRuns checks that the refusal is about `uses` alone:
// a row-scoped hook that declares no capability still applies per row.
func TestRunRowsWithoutUsesStillRuns(t *testing.T) {
	implementation := &rowImplementation{}
	runner := hook.Runner{
		World:          rowScopeWorld("row"),
		Implementation: implementation,
	}
	if err := runner.RunRows(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}, []map[core.CanonicalID]string{
		{"sales.order.amount": "10"},
		{"sales.order.amount": "20"},
	}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(implementation.seen) != 2 {
		t.Fatalf("calls=%d, want one per row", len(implementation.seen))
	}
}
