package operations_test

import (
	"testing"

	"tei/internal/operations"
)

// TestCloseStepMovesOnePlaceAtATime checks that closing takes two steps, so the
// intermediate "closing" state is not skipped. X-TEA advanced the closing state
// from whichever screen the operator used (D:Session.java L1440–1980), so the
// middle state was whatever each screen did.
func TestCloseStepMovesOnePlaceAtATime(t *testing.T) {
	step, ok := operations.CloseStep(operations.PeriodOpen)
	if !ok || step.To != operations.PeriodClosing {
		t.Fatalf("step=%+v ok=%v", step, ok)
	}
	step, ok = operations.CloseStep(operations.PeriodClosing)
	if !ok || step.To != operations.PeriodClosed {
		t.Fatalf("step=%+v ok=%v", step, ok)
	}
	if _, ok := operations.CloseStep(operations.PeriodClosed); ok {
		t.Fatal("a closed period was given a further closing step")
	}
}

// TestNextStateRefusesAnUnknownStart checks that a state outside the declared ones
// is refused rather than treated as open.
func TestNextStateRefusesAnUnknownStart(t *testing.T) {
	if _, err := operations.NextState("archived", operations.PeriodClosed); err == nil {
		t.Fatal("an undeclared state was accepted")
	}
}

// TestNextStateRefusesASkippedStep checks that open cannot jump to closed: the
// closing run would then have no state to be in while it runs.
func TestNextStateRefusesASkippedStep(t *testing.T) {
	if _, err := operations.NextState(operations.PeriodOpen, operations.PeriodClosed); err == nil {
		t.Fatal("open jumped straight to closed")
	}
}

// TestNextStateAllowsReopening checks that a closed period can be reopened to open,
// because a correction after closing is a real event. It does not go back to
// closing: the run that finished is over.
func TestNextStateAllowsReopening(t *testing.T) {
	next, err := operations.NextState(operations.PeriodClosed, operations.PeriodOpen)
	if err != nil {
		t.Fatal(err)
	}
	if next != operations.PeriodOpen {
		t.Fatalf("next=%s", next)
	}
	if _, err := operations.NextState(operations.PeriodClosed, operations.PeriodClosing); err == nil {
		t.Fatal("a closed period moved back to closing")
	}
}

// TestTransitionsAreDeclared checks that the declared steps are the ones a caller
// is offered, so a screen does not have to know them by heart.
func TestTransitionsAreDeclared(t *testing.T) {
	transitions := operations.Transitions()
	if len(transitions) != 3 {
		t.Fatalf("transitions=%+v", transitions)
	}
	for _, transition := range transitions {
		if !transition.Allowed() {
			t.Fatalf("a declared transition is not allowed: %+v", transition)
		}
	}
	// The returned slice is a copy: a caller cannot change the declaration.
	transitions[0] = operations.Transition{}
	opening := operations.Transition{From: operations.PeriodOpen, To: operations.PeriodClosing}
	if !opening.Allowed() {
		t.Fatal("the declaration was changed by the caller")
	}
}
