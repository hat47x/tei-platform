package operations

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/workspace"
)

// PeriodRequirement is the capability a period close must run before the period
// moves to the next state.
//
// The declaration existed (`period ... close-requires <capability>`) and validation
// checked that the capability is declared, but nothing read it when the period was
// closed: a period could name a close capability and close without running it. That
// is the D-9 shape — a declaration present in the source and inert at run time.
type PeriodRequirement struct {
	Period     core.CanonicalID
	Capability core.CanonicalID
	// Correction is the capability a correction after closing must run. It is kept
	// beside the close capability so a caller that needs one does not have to read
	// the period twice.
	Correction core.CanonicalID
}

// PeriodRequirements reads the close and correction capabilities a workspace's
// periods declare.
//
// A period that declares neither is absent from the result, so a caller sees "no
// requirement" as an absence rather than a default capability.
func PeriodRequirements(world ir.World) map[core.CanonicalID]PeriodRequirement {
	out := map[core.CanonicalID]PeriodRequirement{}
	for _, namespace := range world.Semantics {
		for _, period := range namespace.Periods {
			if period.CloseRequires == "" && period.Correction == "" {
				continue
			}
			out[period.ID] = PeriodRequirement{
				Period:     period.ID,
				Capability: period.CloseRequires,
				Correction: period.Correction,
			}
		}
	}
	return out
}

// PeriodRequirementsFromWorkspace loads the workspace and reads the requirements, so
// a caller that has a path does not also have to load the world.
func PeriodRequirementsFromWorkspace(root, applicationFilter string) (map[core.CanonicalID]PeriodRequirement, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return nil, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	_ = applicationFilter
	return PeriodRequirements(loaded.World), nil
}

// CheckCloseRequirement reports the capability a close must run, and refuses the
// close when the period declared one and the caller did not run it.
//
// The check is on the transition, not on the period alone: moving open to closing is
// the close, and it must run the declared capability. A period that declares no
// capability has nothing to check.
func CheckCloseRequirement(requirement PeriodRequirement, from, to PeriodState, ranCapability core.CanonicalID) error {
	if requirement.Capability == "" {
		return nil
	}
	// Only the step that completes the close is guarded. Moving open to closing is
	// the run; moving closing to closed finishes it, and that step needs the same
	// capability to have run. Both steps require it, so a caller cannot reach
	// "closed" by way of an unrun close.
	if ranCapability == requirement.Capability {
		return nil
	}
	return fmt.Errorf("%w: period %s declares close-requires %s, but %s was run for %s to %s",
		ErrRequirementNotRun, requirement.Period, requirement.Capability, ranCapability, from, to)
}

// CheckReopenRequirement refuses a reopen that did not run the period's declared
// correction capability.
//
// Reopening a closed period is a correction after closing: it takes the period back
// to open so its data can be changed. The command path already asks for the
// declared correction capability (COMMAND_PERIOD_CORRECTION_REQUIRED), but the
// period state can be advanced by the operations path too, and that path checked
// only the close requirement. A reopen could therefore move a closed period to open
// without the correction capability ever running — the declaration would hold on
// one entrance and not the other.
//
// Only the closed-to-open step is guarded, and only when the period declares a
// correction capability: a period that declares none has nothing to require here.
func CheckReopenRequirement(requirement PeriodRequirement, from, to PeriodState, ranCapability core.CanonicalID) error {
	if requirement.Correction == "" {
		return nil
	}
	if from != PeriodClosed || to != PeriodOpen {
		return nil
	}
	if ranCapability == requirement.Correction {
		return nil
	}
	return fmt.Errorf("%w: period %s reopens only through correction capability %s, but %s was run",
		ErrRequirementNotRun, requirement.Period, requirement.Correction, ranCapability)
}

// ErrRequirementNotRun reports that a declared requirement was not satisfied, so a
// caller can tell it apart from an invalid transition.
var ErrRequirementNotRun = fmt.Errorf("declared period requirement was not run")
