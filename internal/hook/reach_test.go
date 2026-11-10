package hook

import (
	"context"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// reachWorld declares a record pattern with the given hooks.
func reachWorld(hooks ...application.Hook) ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Hooks: hooks,
			},
		}},
	}}}
}

// TestRunReachableRunsTheReachablePhasesInOrder checks that RunReachable runs the
// phases the reach allows, in PhaseOrder, not in declaration order.
func TestRunReachableRunsTheReachablePhasesInOrder(t *testing.T) {
	world := reachWorld(
		application.Hook{Phase: "before-update"},
		application.Hook{Phase: "before-read"},
	)
	runner := Runner{World: world, Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no report for a pattern that declares hooks")
	}
	want := []string{"before-read", "before-update"}
	if strings.Join(report.Executed, ",") != strings.Join(want, ",") {
		t.Fatalf("executed=%v want=%v", report.Executed, want)
	}
}

// TestRunReachableRecordsTheUnreachablePhasesWithAReason checks that a phase the
// reach cannot reach is skipped with a stated reason instead of being dropped.
func TestRunReachableRecordsTheUnreachablePhasesWithAReason(t *testing.T) {
	world := reachWorld(application.Hook{Phase: "after-update"})
	runner := Runner{World: world, Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no report")
	}
	if len(report.Executed) != 0 {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "after-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if report.Reasons["after-update"] != ReasonAfterChangeUnreachable {
		t.Fatalf("reason=%q", report.Reasons["after-update"])
	}
}

// TestRunReachableRecordsAMissingReadAsNotEvaluated checks that a declared read
// with no supplied value makes the phase "not evaluated", with the attribute
// named in the reason, and that it is not counted as reached.
func TestRunReachableRecordsAMissingReadAsNotEvaluated(t *testing.T) {
	world := reachWorld(application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.status"}})
	runner := Runner{World: world, Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 0 || len(report.Warned) != 0 || report.Refused != "" {
		t.Fatalf("a phase with no read was reported as run: %+v", report)
	}
	if len(report.NotEvaluated) != 1 || report.NotEvaluated[0] != "before-update" {
		t.Fatalf("notEvaluated=%v", report.NotEvaluated)
	}
	if !strings.Contains(report.Reasons["before-update"], "sales.order.status") {
		t.Fatalf("the reason does not name the missing attribute: %q", report.Reasons["before-update"])
	}
	// NotEvaluated and Skipped are the same phase here, so a reader counting
	// skipped phases does not see it as "running later".
	if len(report.Skipped) != 1 {
		t.Fatalf("skipped=%v", report.Skipped)
	}
}

// TestRunReachableStopsAtARefusal checks that an unsatisfied phase stops the
// sequence and records the phase and the business outcome.
func TestRunReachableStopsAtARefusal(t *testing.T) {
	world := reachWorld(
		application.Hook{Phase: "before-read"},
		application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.quantity"}},
	)
	// The rule refuses only where the guarded read is present, so the earlier
	// phase (which declares no read) still runs. ReferenceRule is not
	// phase-aware, so the test uses a rule that is.
	runner := Runner{
		World:          world,
		Implementation: refuseOnQuantity{},
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record",
		map[core.CanonicalID]string{"sales.order.quantity": "0"}, ReferenceReach{})
	if err == nil {
		t.Fatal("a non-positive value did not stop the sequence")
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("report=%+v", report)
	}
	if report.Outcome != "value-rejected" {
		t.Fatalf("outcome=%q", report.Outcome)
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-read" {
		t.Fatalf("executed=%v", report.Executed)
	}
}

// TestRunReachableWithNoHooksReportsNothing checks that a pattern without a hook
// returns no report, so a caller does not present "no hooks" as "hooks passed".
func TestRunReachableWithNoHooksReportsNothing(t *testing.T) {
	runner := Runner{World: reachWorld(), Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report != nil {
		t.Fatalf("report=%+v", report)
	}
}

// TestRunReachableRecordsAnUnsupportedPhase checks that a phase the order does
// not name is recorded, not silently dropped by the loop.
func TestRunReachableRecordsAnUnsupportedPhase(t *testing.T) {
	world := reachWorld(application.Hook{Phase: "during-update"})
	runner := Runner{World: world, Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 0 {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "during-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if report.Reasons["during-update"] != ReasonPhaseUnsupported {
		t.Fatalf("reason=%q", report.Reasons["during-update"])
	}
}

// TestRunReachableKeepsOnFailureWarnGoing checks that a warn phase is recorded
// and the sequence continues.
func TestRunReachableKeepsOnFailureWarnGoing(t *testing.T) {
	world := reachWorld(
		application.Hook{Phase: "before-read", Failure: "warn"},
		application.Hook{Phase: "before-update", Failure: "warn"},
	)
	runner := Runner{World: world, Implementation: failingRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("a warn phase stopped the sequence: %v", err)
	}
	if len(report.Warned) != 2 {
		t.Fatalf("warned=%v", report.Warned)
	}
	if len(report.Executed) != 0 {
		t.Fatalf("executed=%v", report.Executed)
	}
}

// refuseOnQuantity is unsatisfied when the guarded read is present and not
// positive. It leaves a phase that declares no read alone.
type refuseOnQuantity struct{}

func (refuseOnQuantity) Run(_ context.Context, invocation Invocation) error {
	value, observed := invocation.Reads["sales.order.quantity"]
	if !observed {
		return nil
	}
	if value == "0" {
		return &NotSatisfied{Reason: "quantity is not positive", Outcome: "value-rejected"}
	}
	return nil
}

// failingRule reports an execution failure for every phase.
type failingRule struct{}

func (failingRule) Run(context.Context, Invocation) error {
	return &Error{Code: "RULE_FAILED", Message: "the rule failed"}
}
