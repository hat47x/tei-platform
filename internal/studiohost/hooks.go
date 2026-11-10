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
// hookHalf selects which side of the change a hook run covers, the same split the
// CLI uses: a run that applies its change consults the before-* phases first,
// applies, then consults the after-* phases.
type hookHalf int

const (
	hookHalfAll hookHalf = iota
	hookHalfBefore
	hookHalfAfter
)

// runHooks runs every declared phase a reference execution can reach.
func (h *ReadOnly) runHooks(world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string) (*HookRunReport, error) {
	return h.runHooksHalf(world, applicationID, patternID, inputs, hook.ReferenceRule{}, hook.ReferenceReach{}, hookHalfAll)
}

// runHooksHalf runs one half of the declared phases, with the reach the caller
// says applies.
func (h *ReadOnly) runHooksHalf(world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string, rule hook.Implementation, reach hook.Reach, half hookHalf) (*HookRunReport, error) {
	runner := hook.Runner{World: world, Implementation: rule}
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
	var report *hook.PhaseReport
	var err error
	switch half {
	case hookHalfBefore:
		report, err = runner.RunBeforeChange(context.Background(), applicationID, patternID, studioAttributeValues(inputs), reach)
	case hookHalfAfter:
		report, err = runner.RunAfterChange(context.Background(), applicationID, patternID, studioAttributeValues(inputs), reach)
	default:
		report, err = runner.RunReachable(context.Background(), applicationID, patternID, studioAttributeValues(inputs), reach)
	}
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

// runAfterHalfRevertingOnRefusal consults the after-change phases, and puts the
// store back when one of them refuses.
//
// The effect was applied before this runs, so a refusal here comes after the
// change. Leaving it in place would keep something the declaration did not
// accept; the store is restored to the snapshot taken before the run, and the
// caller is told the effect was reverted.
//
// The reference rule never refuses, so Studio's own runs do not reach the
// revert today. The path exists and is tested with a refusing rule, because the
// rule a plugin supplies is what makes it reachable.
func (h *ReadOnly) runAfterHalfRevertingOnRefusal(world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string, rule hook.Implementation, store *command.MemoryStore, snapshot command.StoreSnapshot, before *HookRunReport) (*HookRunReport, bool, error) {
	after, err := h.runHooksHalf(world, applicationID, patternID, inputs, rule, hook.AppliedChangeReach{}, hookHalfAfter)
	if err != nil {
		store.Restore(snapshot)
		return after, true, err
	}
	return after, false, nil
}
