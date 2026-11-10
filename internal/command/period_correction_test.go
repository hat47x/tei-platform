package command

import (
	"context"
	"testing"

	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// correctionWorld is a period that declares a correction capability and rejects
// updates once closed, so a test can tell the correction rule apart from
// closed-rejects.
func correctionWorld(correction string, closedRejects []string) ir.World {
	world := commandWorld()
	world.Semantics[0].Periods = []semantic.Period{{
		ID:            "sales.period",
		Data:          "sales.order",
		BusinessDate:  "sales.order.order_id",
		States:        []string{"open", "closed"},
		ClosedRejects: closedRejects,
		Correction:    core.CanonicalID(correction),
	}}
	return world
}

func closedInvoker(world ir.World) Invoker {
	state := NewMemoryPeriodState()
	state.SetClosed("sales.period", true)
	return Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}},
		Periods:    state,
	}
}

// TestInvokeRefusesACorrectionThroughAnotherCapability checks that a closed period
// which declares `correction via <capability>` refuses a change through any other
// capability. The declaration was validated but never read at execution, so any
// capability could change a closed period (D-9).
func TestInvokeRefusesACorrectionThroughAnotherCapability(t *testing.T) {
	// The period declares correction but does not reject update, so the refusal
	// below can only come from the correction rule.
	invoker := closedInvoker(correctionWorld("sales.correct", nil))
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_PERIOD_CORRECTION_REQUIRED" {
		t.Fatalf("code=%s", code)
	}
}

// TestCorrectionCheckAcceptsTheDeclaredCapability checks the rule itself: running the
// declared correction capability is accepted, so the rule names one capability rather
// than blocking every change.
//
// The rule is checked directly because a request for another capability does not
// reach the period check: the pattern must also declare that capability, and the
// command boundary refuses an undeclared one earlier.
func TestCorrectionCheckAcceptsTheDeclaredCapability(t *testing.T) {
	plan := applicationview.CommandPlan{
		Capability: "sales.correct",
		Period:     &applicationview.CommandPeriodRef{ID: "sales.period", Correction: "sales.correct"},
	}
	if err := CorrectionCheck(plan, "sales.correct"); err != nil {
		t.Fatalf("the declared correction capability was refused: %v", err)
	}
	if err := CorrectionCheck(plan, "sales.submit"); err == nil {
		t.Fatal("another capability was accepted as a correction")
	}
	if err := CorrectionCheck(plan, ""); err == nil {
		t.Fatal("a request naming no capability was accepted")
	}
}

// TestInvokeLeavesAnOpenPeriodAlone checks that the correction rule applies only to
// a closed period: an open period needs no correction.
func TestInvokeLeavesAnOpenPeriodAlone(t *testing.T) {
	state := NewMemoryPeriodState()
	state.SetClosed("sales.period", false)
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      correctionWorld("sales.correct", nil),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}},
		Periods:    state,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatalf("an open period was refused: %v", err)
	}
}

// TestInvokeLeavesAPeriodWithoutACorrectionAlone checks that a period which declares
// no correction capability is not guarded by the rule: closed-rejects decides.
func TestInvokeLeavesAPeriodWithoutACorrectionAlone(t *testing.T) {
	invoker := closedInvoker(correctionWorld("", nil))
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatalf("a period without a correction declaration was refused: %v", err)
	}
}
