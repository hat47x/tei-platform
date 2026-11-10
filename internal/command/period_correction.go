package command

import (
	"fmt"

	"tei/internal/applicationview"
	"tei/internal/core"
)

// CorrectionCheck reports whether the capability the request names is the
// correction capability the period declares.
//
// A period may declare `correction via <capability>`, meaning a change to a closed
// period must go through that capability. The declaration was validated (the
// capability must exist) but nothing read it at execution, so a change to a closed
// period could run through any capability. That is the D-9 shape: a declared value
// that does nothing at run time.
//
// The check is on the capability the request names, not on the period alone: the
// period says which capability may correct it, and the request says which one is
// running. It is called only when the period is closed, because an open period needs
// no correction.
func CorrectionCheck(plan applicationview.CommandPlan, capability core.CanonicalID) error {
	if plan.Period == nil {
		return nil
	}
	declared := plan.Period.Correction
	if declared == "" {
		// A period that declares no correction capability is not guarded here: it
		// either allows any capability or refuses the change with closed-rejects, and
		// both are declared elsewhere.
		return nil
	}
	if capability == declared {
		return nil
	}
	if capability == "" {
		return &ExecutionError{
			Code:    "COMMAND_PERIOD_CORRECTION_REQUIRED",
			Message: fmt.Sprintf("period %s is closed and allows a correction only through %s, but the request names no capability", plan.Period.ID, declared),
		}
	}
	return &ExecutionError{
		Code:    "COMMAND_PERIOD_CORRECTION_REQUIRED",
		Message: fmt.Sprintf("period %s allows a correction only through %s, but %s is running", plan.Period.ID, declared, capability),
	}
}
