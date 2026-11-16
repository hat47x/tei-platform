package hook_test

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// afterRowWorld declares one row-scoped hook on an after-* phase, which is the
// phase a reference execution cannot reach because nothing has been changed yet.
func afterRowWorld() ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{Hooks: []application.Hook{{
					Phase: "after-update",
					Scope: "row",
					Reads: []core.CanonicalID{"sales.order.amount"},
				}}},
			}},
		}},
	}
}

// TestRowScopeRespectsReach checks that a row-scoped hook on an unreachable phase is
// skipped even when the caller supplies a row set.
//
// Reach is decided before scope. Deciding the scope first ran an after-* hook when
// nothing had changed and recorded the phase as executed, which reads as "the
// after-change check ran". A page of rows does not make the change observable.
func TestRowScopeRespectsReach(t *testing.T) {
	implementation := &rowImplementation{}
	runner := hook.Runner{
		World:          afterRowWorld(),
		Implementation: implementation,
		Rows:           rowsOf("10", "20"),
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, hook.ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(implementation.seen) != 0 {
		t.Fatalf("an unreachable phase ran with a row set: %v", implementation.seen)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "after-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if report.Reasons["after-update"] == "" {
		t.Fatal("the skip has no reason")
	}
	for _, phase := range report.Executed {
		if phase == "after-update" {
			t.Fatal("an unreachable phase was recorded as executed")
		}
	}
	// A reach that can see the change runs the phase per row, so the ordering does
	// not revoke the feature it guards.
	applied := &rowImplementation{}
	appliedRunner := hook.Runner{
		World:          afterRowWorld(),
		Implementation: applied,
		Rows:           rowsOf("10", "20"),
	}
	appliedReport, err := appliedRunner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, hook.AppliedChangeReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(applied.seen) != 2 {
		t.Fatalf("the applied reach did not run per row: %v", applied.seen)
	}
	if len(appliedReport.Executed) != 1 || appliedReport.Executed[0] != "after-update" {
		t.Fatalf("executed=%v", appliedReport.Executed)
	}
}
