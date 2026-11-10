package hook

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// afterChangeWorld declares a before-update and an after-update hook on one
// record pattern, so a test can run one half, apply a change, and run the other.
func afterChangeWorld() ir.World {
	return reachWorld(
		application.Hook{Phase: "before-update"},
		application.Hook{Phase: "after-update"},
	)
}

// TestAfterChangePhaseRunsAfterTheChangeIsApplied checks the execution path the
// reference run does not have.
//
// A reference run writes nothing, so it has no after-change point and reports
// after-* as unreachable. A run that applies its change does have one. The two
// halves are run separately — RunBeforeChange, apply, RunAfterChange — because the
// change happens between them and a single pass has no point in between.
func TestAfterChangePhaseRunsAfterTheChangeIsApplied(t *testing.T) {
	runner := Runner{World: afterChangeWorld(), Implementation: ReferenceRule{}}

	before, err := runner.RunBeforeChange(context.Background(), "order_entry",
		"order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("before: %v", err)
	}
	if len(before.Executed) != 1 || before.Executed[0] != "before-update" {
		t.Fatalf("before executed=%v", before.Executed)
	}
	// The after phase is not part of the first half at all.
	for _, phase := range before.Executed {
		if phase == "after-update" {
			t.Fatalf("the after phase ran before the change: %v", before.Executed)
		}
	}

	// The change is applied here. The store is the reference one, so the test does
	// not need a provider to have a real change to point at.
	store := command.NewMemoryStore()
	if err := store.ApplyRecord(
		&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"},
		"order=ORD-1",
		map[string]string{"order": "ORD-1"},
		nil,
	); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, ok := store.Get("sales.order", "order=ORD-1"); !ok {
		t.Fatal("the change was not applied")
	}

	after, err := runner.RunAfterChange(context.Background(), "order_entry",
		"order_entry.order_record", nil, AppliedChangeReach{})
	if err != nil {
		t.Fatalf("after: %v", err)
	}
	if len(after.Executed) != 1 || after.Executed[0] != "after-update" {
		t.Fatalf("after executed=%v", after.Executed)
	}
}

// TestAfterChangePhaseStaysUnreachableWithoutAnAppliedChange checks that the
// reference reach still governs the second half: asking for the after phases with
// a reach that has no after-change point reports them as skipped with the reason,
// rather than running them.
func TestAfterChangePhaseStaysUnreachableWithoutAnAppliedChange(t *testing.T) {
	runner := Runner{World: afterChangeWorld(), Implementation: ReferenceRule{}}
	report, err := runner.RunAfterChange(context.Background(), "order_entry",
		"order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 0 {
		t.Fatalf("an after phase ran without an applied change: %v", report.Executed)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "after-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if report.Reasons["after-update"] != ReasonAfterChangeUnreachable {
		t.Fatalf("reason=%q", report.Reasons["after-update"])
	}
}

// TestBeforeChangeHalfDoesNotReportTheAfterHalf checks that asking about one half
// does not report the other half as skipped: the caller asked about one half, and
// a skip it did not ask about reads as a check it did not request.
func TestBeforeChangeHalfDoesNotReportTheAfterHalf(t *testing.T) {
	runner := Runner{World: afterChangeWorld(), Implementation: ReferenceRule{}}
	report, err := runner.RunBeforeChange(context.Background(), "order_entry",
		"order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	for _, phase := range report.Skipped {
		if phase == "after-update" {
			t.Fatalf("the other half was reported as skipped: %v", report.Skipped)
		}
	}
	if _, reported := report.Reasons["after-update"]; reported {
		t.Fatalf("the other half has a reason: %q", report.Reasons["after-update"])
	}
}

// TestRunReachableStillRunsEveryDeclaredPhase checks that splitting the loop did
// not change what the single entry point does.
func TestRunReachableStillRunsEveryDeclaredPhase(t *testing.T) {
	runner := Runner{World: afterChangeWorld(), Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry",
		"order_entry.order_record", nil, AppliedChangeReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 2 {
		t.Fatalf("executed=%v", report.Executed)
	}
	if report.Executed[0] != "before-update" || report.Executed[1] != "after-update" {
		t.Fatalf("executed=%v", report.Executed)
	}
}
