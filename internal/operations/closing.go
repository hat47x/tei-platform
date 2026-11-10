package operations

import (
	"fmt"
)

// Transition is one step of a period's lifecycle: a period moves from one state
// to another, and the step says who may take it.
//
// X-TEA advanced the closing state by writing the period table from whichever
// screen the operator used (D:Session.java L1440–1980), so the allowed steps were
// whatever each screen did and the intermediate "closing" state was not a
// declaration. Here the steps are listed, and a caller can see which move it is
// asking for.
type Transition struct {
	From PeriodState
	To   PeriodState
}

// periodTransitions lists every step a period may take. Closing is two steps on
// purpose: "closing" means the closing run is under way, and "closed" means it
// finished. Skipping the middle state hides a long run from the operator, and
// reopening from "closing" drops a partly-finished run without saying so.
var periodTransitions = []Transition{
	{From: PeriodOpen, To: PeriodClosing},
	{From: PeriodClosing, To: PeriodClosed},
	// A closed period can be reopened, because a correction after closing is a
	// real business event. It goes back to "open", not to "closing": the closing
	// run that finished is over, and a new one starts when the operator asks.
	{From: PeriodClosed, To: PeriodOpen},
}

// Allowed reports whether the step is one the period may take.
func (t Transition) Allowed() bool {
	for _, allowed := range periodTransitions {
		if allowed == t {
			return true
		}
	}
	return false
}

// Transitions returns the steps a period may take, in the order they are declared.
// The caller uses this to offer the moves without knowing them by heart.
func Transitions() []Transition {
	return append([]Transition(nil), periodTransitions...)
}

// NextState returns the state a period moves to when the caller asks for the given
// step, checking that the current state is the one the step starts from.
//
// The check is on the current state, not on the requested target alone: a caller
// that says "close" for a period that is already closed is refused, so a repeated
// request cannot be mistaken for a fresh one.
func NextState(current PeriodState, to PeriodState) (PeriodState, error) {
	switch current {
	case PeriodOpen, PeriodClosing, PeriodClosed:
	default:
		return "", fmt.Errorf("period state %q is not open, closing, or closed", current)
	}
	step := Transition{From: current, To: to}
	if !step.Allowed() {
		return "", fmt.Errorf("a period cannot move from %s to %s", current, to)
	}
	return to, nil
}

// CloseStep returns the step that moves a period one place toward closed: open
// goes to closing, and closing goes to closed. A closed period has no further
// step, so the caller is told rather than given the same state again.
func CloseStep(current PeriodState) (Transition, bool) {
	switch current {
	case PeriodOpen:
		return Transition{From: PeriodOpen, To: PeriodClosing}, true
	case PeriodClosing:
		return Transition{From: PeriodClosing, To: PeriodClosed}, true
	default:
		return Transition{}, false
	}
}
