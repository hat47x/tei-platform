package main

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/hook"
)

// appliedHooks are the two hooks a run that applies its change consults on either
// side of the change.
func appliedHooks() []application.Hook {
	return []application.Hook{
		{Phase: "before-update"},
		{Phase: "after-update"},
	}
}

// TestRunDeclaredHooksHalfSplitsAroundTheChange checks that the before half stops
// at the change and the after half starts there.
//
// A reference run consults every phase in one pass because it has no change. A run
// that applies one cannot: the change sits between the halves, and one pass has no
// point in between.
func TestRunDeclaredHooksHalfSplitsAroundTheChange(t *testing.T) {
	world := hookViewWorld(appliedHooks()...)

	before, err := runDeclaredHooksHalf(context.Background(), world, "order_entry",
		"order_entry.order_record", nil, hook.ReferenceRule{}, hook.ReferenceReach{}, hookHalfBefore)
	if err != nil {
		t.Fatalf("before: %v", err)
	}
	if len(before.Executed) != 1 || before.Executed[0] != "before-update" {
		t.Fatalf("before executed=%v", before.Executed)
	}
	// Asking about the first half does not report the second half as skipped: the
	// caller did not ask about it, and a skip would read as a check it did not run.
	for _, phase := range before.Skipped {
		if phase == "after-update" {
			t.Fatalf("the second half was reported as skipped: %v", before.Skipped)
		}
	}

	after, err := runDeclaredHooksHalf(context.Background(), world, "order_entry",
		"order_entry.order_record", nil, hook.ReferenceRule{}, hook.AppliedChangeReach{}, hookHalfAfter)
	if err != nil {
		t.Fatalf("after: %v", err)
	}
	if len(after.Executed) != 1 || after.Executed[0] != "after-update" {
		t.Fatalf("after executed=%v", after.Executed)
	}
}

// TestRunDeclaredHooksAfterHalfStaysSkippedWithoutAChange checks that the reach
// still governs the second half: a reference reach has no after-change point, so
// the after phase is skipped with the reason rather than run.
func TestRunDeclaredHooksAfterHalfStaysSkippedWithoutAChange(t *testing.T) {
	world := hookViewWorld(appliedHooks()...)
	after, err := runDeclaredHooksHalf(context.Background(), world, "order_entry",
		"order_entry.order_record", nil, hook.ReferenceRule{}, hook.ReferenceReach{}, hookHalfAfter)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(after.Executed) != 0 {
		t.Fatalf("an after phase ran without a change: %v", after.Executed)
	}
	if len(after.Skipped) != 1 || after.Skipped[0] != "after-update" {
		t.Fatalf("skipped=%v", after.Skipped)
	}
}

// TestMergeHookReportsKeepsBothHalves checks that the printed run shows what both
// halves saw, in the order they ran.
func TestMergeHookReportsKeepsBothHalves(t *testing.T) {
	world := hookViewWorld(appliedHooks()...)
	before, err := runDeclaredHooksHalf(context.Background(), world, "order_entry",
		"order_entry.order_record", nil, hook.ReferenceRule{}, hook.ReferenceReach{}, hookHalfBefore)
	if err != nil {
		t.Fatalf("before: %v", err)
	}
	after, err := runDeclaredHooksHalf(context.Background(), world, "order_entry",
		"order_entry.order_record", nil, hook.ReferenceRule{}, hook.AppliedChangeReach{}, hookHalfAfter)
	if err != nil {
		t.Fatalf("after: %v", err)
	}
	merged := mergeHookReports(before, after)
	if merged == nil {
		t.Fatal("merge returned nothing")
	}
	if len(merged.Executed) != 2 || merged.Executed[0] != "before-update" || merged.Executed[1] != "after-update" {
		t.Fatalf("executed=%v", merged.Executed)
	}
	if merged.Pattern != "order_entry.order_record" {
		t.Fatalf("pattern=%s", merged.Pattern)
	}
}

// TestMergeHookReportsHandlesOneMissingHalf checks the degenerate case, so a
// caller that only has one report does not get a panic or an empty report.
func TestMergeHookReportsHandlesOneMissingHalf(t *testing.T) {
	world := hookViewWorld(appliedHooks()...)
	before, err := runDeclaredHooksHalf(context.Background(), world, "order_entry",
		"order_entry.order_record", nil, hook.ReferenceRule{}, hook.ReferenceReach{}, hookHalfBefore)
	if err != nil {
		t.Fatalf("before: %v", err)
	}
	if got := mergeHookReports(before, nil); got != before {
		t.Fatalf("merge with a missing half returned %+v", got)
	}
	if got := mergeHookReports(nil, before); got != before {
		t.Fatalf("merge with a missing half returned %+v", got)
	}
}
