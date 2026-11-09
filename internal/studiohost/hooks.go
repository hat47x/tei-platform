package studiohost

import (
	"context"

	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// HookRunReport records the pattern hooks the Studio reference execution
// consulted, and which of them it could not evaluate.
//
// The phase policy lives in internal/hook (Runner.RunReachable), so Studio and
// the CLI do not each carry their own copy. This type adds the boundary note
// that a Studio reader needs.
type HookRunReport struct {
	*hook.PhaseReport
	Pattern core.CanonicalID `json:"pattern"`
	Note    string           `json:"note"`
}

const hookRunNote = "the reference rule evaluates the declared reads only; running it does not accept the rule, observe the command effect, or grant authority"

// runHooks runs the hooks a pattern declares that the reference execution can
// reach, before the reference command runs.
func (h *ReadOnly) runHooks(world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string) (*HookRunReport, error) {
	runner := hook.Runner{World: world, Implementation: hook.ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), applicationID, patternID, studioAttributeValues(inputs), hook.ReferenceReach{})
	if report == nil {
		return nil, err
	}
	return &HookRunReport{PhaseReport: report, Pattern: patternID, Note: hookRunNote}, err
}

// studioAttributeValues keys the caller's inputs by canonical attribute name so
// the hook boundary can look up what each phase declares.
func studioAttributeValues(inputs map[string]string) map[core.CanonicalID]string {
	values := make(map[core.CanonicalID]string, len(inputs))
	for name, value := range inputs {
		values[core.CanonicalID(name)] = value
	}
	return values
}
