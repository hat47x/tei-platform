package studiohost

import (
	"context"

	"tei/internal/command"
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
	// CommandOutcome is the Command-side name this refusal corresponds to, when
	// the hook named one that Command also uses. It is empty when the hook's
	// outcome has no Command counterpart.
	CommandOutcome string `json:"commandOutcome,omitempty"`
}

const hookRunNote = "the reference rule evaluates the declared reads only; running it does not accept the rule, observe the command effect, or grant authority"

// runHooks runs the hooks a pattern declares that the reference execution can
// reach, before the reference command runs.
func (h *ReadOnly) runHooks(world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string) (*HookRunReport, error) {
	runner := hook.Runner{World: world, Implementation: hook.ReferenceRule{}}
	// A hook that declares `uses` promises a call. Give the runner the boundary
	// that can keep it, the same way the CLI does, so Studio does not report a
	// phase as executed while its declared call never happened.
	runner.Caller = hook.CommandCaller{
		Invoker: command.Invoker{
			World:      world,
			Executor:   command.ReferenceExecutor{World: world},
			UnitOfWork: &command.MemoryUnitOfWork{},
			Authorizer: command.ReferenceAuthorizer{},
		},
		Application: applicationID,
		Pattern: func(capability core.CanonicalID) (core.CanonicalID, bool) {
			pattern, err := hook.CommandPatternFor(world, applicationID, capability)
			if err != nil {
				return "", false
			}
			return pattern, true
		},
	}
	report, err := runner.RunReachable(context.Background(), applicationID, patternID, studioAttributeValues(inputs), hook.ReferenceReach{})
	if report == nil {
		return nil, err
	}
	out := &HookRunReport{PhaseReport: report, Pattern: patternID, Note: hookRunNote}
	// Only a refusal names an outcome. The name is mapped through the declared
	// correspondence; a name with no Command counterpart stays empty rather than
	// being reported as a Command refusal reason.
	if name, ok := hook.CommandOutcomeFor(report.Outcome); ok {
		out.CommandOutcome = name
	}
	return out, err
}

// studioHookCallInputs binds, for each capability a pattern's hooks declare in
// `uses`, the inputs the Command boundary receives. The hook's own inputs are
// used as-is; this function does not invent a mapping from a hook's reads to a
// capability's parameters. An unbound capability is reported as not called.
func studioHookCallInputs(world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string) map[core.CanonicalID]map[string]string {
	bound := map[core.CanonicalID]map[string]string{}
	for i := range world.Applications {
		if world.Applications[i].ID != applicationID {
			continue
		}
		for _, pattern := range world.Applications[i].Patterns {
			if pattern.ID != patternID || pattern.Interaction == nil {
				continue
			}
			for _, declaration := range pattern.Interaction.Hooks {
				for _, capability := range declaration.Uses {
					if _, done := bound[capability]; done {
						continue
					}
					values := make(map[string]string, len(inputs))
					for name, value := range inputs {
						values[name] = value
					}
					bound[capability] = values
				}
			}
		}
	}
	return bound
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
