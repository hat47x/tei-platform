package operations_test

import (
	"errors"
	"testing"

	"tei/internal/operations"
)

// TestReopenRequiresTheDeclaredCorrectionCapability checks that a closed period which
// declares `correction via <capability>` is not reopened without that capability.
//
// The command path already asks for it, but the period state can also be advanced by
// the operations path, which checked only the close requirement. A reopen could
// therefore move a closed period to open without the correction ever running: the
// declaration held on one entrance and not the other.
func TestReopenRequiresTheDeclaredCorrectionCapability(t *testing.T) {
	requirement := operations.PeriodRequirement{
		Period:     "sales.month",
		Capability: "sales.period.close",
		Correction: "sales.period.correct",
	}
	// A reopen with no capability is refused.
	if err := operations.CheckReopenRequirement(requirement, operations.PeriodClosed, operations.PeriodOpen, ""); err == nil {
		t.Fatal("a reopen without the correction capability was accepted")
	}
	// A reopen naming the close capability is not the correction, so it is refused.
	if err := operations.CheckReopenRequirement(requirement, operations.PeriodClosed, operations.PeriodOpen, "sales.period.close"); err == nil {
		t.Fatal("the close capability was accepted as a correction")
	}
	// The refusal is the shared "requirement not run" error, so a caller can tell it
	// apart from an invalid transition.
	err := operations.CheckReopenRequirement(requirement, operations.PeriodClosed, operations.PeriodOpen, "")
	if !errors.Is(err, operations.ErrRequirementNotRun) {
		t.Fatalf("the refusal is not the requirement error: %v", err)
	}
	// The declared correction capability lets the reopen through.
	if err := operations.CheckReopenRequirement(requirement, operations.PeriodClosed, operations.PeriodOpen, "sales.period.correct"); err != nil {
		t.Fatalf("the declared correction capability was refused: %v", err)
	}
}

// TestReopenCheckLeavesTheCloseStepsAlone checks that the reopen check applies to
// the closed-to-open step only: closing must not be refused for lacking a correction
// capability it does not need.
func TestReopenCheckLeavesTheCloseStepsAlone(t *testing.T) {
	requirement := operations.PeriodRequirement{
		Period:     "sales.month",
		Correction: "sales.period.correct",
	}
	if err := operations.CheckReopenRequirement(requirement, operations.PeriodOpen, operations.PeriodClosing, ""); err != nil {
		t.Fatalf("the close step was refused by the reopen rule: %v", err)
	}
	if err := operations.CheckReopenRequirement(requirement, operations.PeriodClosing, operations.PeriodClosed, ""); err != nil {
		t.Fatalf("the finishing step was refused by the reopen rule: %v", err)
	}
}

// TestReopenCheckLeavesAPeriodWithoutCorrectionAlone checks that a period which
// declares no correction capability is not guarded: there is nothing to require.
func TestReopenCheckLeavesAPeriodWithoutCorrectionAlone(t *testing.T) {
	if err := operations.CheckReopenRequirement(operations.PeriodRequirement{Period: "sales.week"},
		operations.PeriodClosed, operations.PeriodOpen, ""); err != nil {
		t.Fatalf("a period with no correction declaration was refused: %v", err)
	}
}
