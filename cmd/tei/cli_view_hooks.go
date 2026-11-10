package main

import (
	"context"

	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// cliHookReport records the hooks the CLI consulted before a reference command,
// and which of them it could not evaluate.
//
// The CLI reference run has no data source: it does not read a database, so a
// hook that declares reads cannot be evaluated unless the caller supplied a
// value for every declared attribute. Recording that as "not evaluated" keeps
// the difference between "the hook passed" and "the hook never ran" visible; it
// does not let a skipped hook look like a satisfied one, and it does not
// fabricate a value the run did not read.
//
// The phase policy itself lives in internal/hook (Runner.RunReachable), so the
// CLI and Studio do not each carry their own copy of it. This type adds only
// the wording a CLI reader needs.
type cliHookReport struct {
	*hook.PhaseReport
	// Pattern is the pattern whose interaction declares the hooks.
	Pattern core.CanonicalID `json:"pattern"`
	// Note states the boundary: running a declared hook is not evidence that a
	// rule plugin decided it, and it does not make the command effect observed.
	Note string `json:"note"`
	// CommandOutcome is the Command-side name this refusal corresponds to, when
	// the hook named one that Command also uses. It is empty when the hook's
	// outcome has no Command counterpart (`not-satisfied`), so a caller does not
	// read "the hook was not satisfied" as a Command refusal reason.
	CommandOutcome string `json:"commandOutcome,omitempty"`
}

const cliHookNote = "the reference rule evaluates the declared reads only; running it does not accept the rule, observe the command effect, or grant authority"

// runDeclaredHooks runs the hooks a pattern declares that the reference run can
// reach, before the reference command.
//
// A phase runs only when it is a before-change phase and every declared read
// has a value from the command inputs. Everything else is recorded as skipped
// or not evaluated with a reason; the command still runs, because a reference
// run has no data effect that an unevaluated hook could have guarded.
//
// A hook that does run and refuses stops the command, so a failure the reference
// rule did decide is not overridden.
func runDeclaredHooks(ctx context.Context, world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string) (*cliHookReport, error) {
	// The CLI reference run applies no business rule of its own: the rule it
	// consults only checks that the declared reads exist, which hook.ReadsFor
	// already enforces. A host that wants a rule supplies one through
	// runDeclaredHooksWithRule.
	return runDeclaredHooksWithRule(ctx, world, applicationID, patternID, inputs, hook.ReferenceRule{})
}

// runDeclaredHooksWithRule is runDeclaredHooks with an explicit reference rule,
// so a caller (or a test) can decide what the hook considers unsatisfied.
func runDeclaredHooksWithRule(ctx context.Context, world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string, rule hook.Implementation) (*cliHookReport, error) {
	return runDeclaredHooksHalf(ctx, world, applicationID, patternID, inputs, rule, hook.ReferenceReach{}, hookHalfAll)
}

// hookHalf selects which side of the change a hook run covers.
//
// A run that applies its change cannot consult every phase in one pass: the
// before-* phases have to decide before the change and the after-* phases after
// it, with the change in between. The two named halves let the caller put the
// change where it belongs instead of the runner guessing a point.
type hookHalf int

const (
	// hookHalfAll consults every declared phase, which is what a reference run
	// does because it has no change to sit between the halves.
	hookHalfAll hookHalf = iota
	// hookHalfBefore consults the before-* phases only.
	hookHalfBefore
	// hookHalfAfter consults the after-* phases only.
	hookHalfAfter
)

// runDeclaredHooksHalf consults one half of the declared phases, with the reach
// the caller says applies.
func runDeclaredHooksHalf(ctx context.Context, world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string, rule hook.Implementation, reach hook.Reach, half hookHalf) (*cliHookReport, error) {
	runner := hook.Runner{World: world, Implementation: rule}
	// A hook that declares `uses` promises a call. Give the runner the boundary
	// that can keep it: the Command invoker, with the pattern resolution
	// CommandPatternFor performs. The capability's inputs are the hook's own
	// inputs, bound through the resolved command pattern.
	invoker := command.Invoker{
		World:      world,
		Executor:   command.ReferenceExecutor{World: world},
		UnitOfWork: &command.MemoryUnitOfWork{},
		Authorizer: command.ReferenceAuthorizer{},
	}
	runner.Caller = hook.CommandCaller{
		Invoker:     invoker,
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
		report, err = runner.RunBeforeChange(ctx, applicationID, patternID, attributeValues(inputs), reach)
	case hookHalfAfter:
		report, err = runner.RunAfterChange(ctx, applicationID, patternID, attributeValues(inputs), reach)
	default:
		report, err = runner.RunReachable(ctx, applicationID, patternID, attributeValues(inputs), reach)
	}
	if report == nil {
		return nil, err
	}
	out := &cliHookReport{PhaseReport: report, Pattern: patternID, Note: cliHookNote}
	// Only a refusal names an outcome. The name is mapped through the declared
	// correspondence; a name with no Command counterpart stays empty rather than
	// being reported as a Command refusal reason.
	if name, ok := hook.CommandOutcomeFor(report.Outcome); ok {
		out.CommandOutcome = name
	}
	return out, err
}

// hookCallInputs binds, for each capability a pattern's hooks declare in
// `uses`, the inputs the Command boundary receives. The hook's own inputs are
// used as-is: this function does not invent a mapping from a hook's reads to a
// capability's parameters, because that correspondence is a declaration the
// hook does not currently carry. An unbound capability is reported as not
// called rather than called with made-up values.
func hookCallInputs(world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string) map[core.CanonicalID]map[string]string {
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
					bound[capability] = copyInputs(inputs)
				}
			}
		}
	}
	return bound
}

// copyInputs returns a copy so a capability cannot widen the caller's map after
// the binding is recorded.
func copyInputs(inputs map[string]string) map[string]string {
	out := make(map[string]string, len(inputs))
	for name, value := range inputs {
		out[name] = value
	}
	return out
}

// attributeValues keys the command inputs by canonical attribute name so the
// hook boundary can look up what each phase declares.
func attributeValues(inputs map[string]string) map[core.CanonicalID]string {
	values := make(map[core.CanonicalID]string, len(inputs))
	for name, value := range inputs {
		values[core.CanonicalID(name)] = value
	}
	return values
}

// hookReportHolds reports whether a list of phases holds a phase.
func hookReportHolds(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// mergeHookReports joins the two halves of an applied run into the one report the
// CLI prints.
//
// The halves are separate runs, so each carries only what it saw. A refusal or a
// proposal belongs to the phase that produced it, and the order the halves were
// run in is the order the report lists.
func mergeHookReports(before, after *cliHookReport) *cliHookReport {
	if before == nil {
		return after
	}
	if after == nil {
		return before
	}
	out := &cliHookReport{
		PhaseReport: &hook.PhaseReport{
			Reasons: map[string]string{},
		},
		Pattern: before.Pattern,
		Note:    before.Note,
	}
	for _, source := range []*cliHookReport{before, after} {
		if source.PhaseReport == nil {
			continue
		}
		out.Executed = append(out.Executed, source.Executed...)
		out.Warned = append(out.Warned, source.Warned...)
		out.Skipped = append(out.Skipped, source.Skipped...)
		out.NotEvaluated = append(out.NotEvaluated, source.NotEvaluated...)
		for phase, reason := range source.Reasons {
			out.Reasons[phase] = reason
		}
		if source.Refused != "" {
			out.Refused = source.Refused
		}
		if source.Outcome != "" {
			out.Outcome = source.Outcome
		}
		for phase, produced := range source.Produced {
			if out.Produced == nil {
				out.Produced = map[string]hook.Produced{}
			}
			out.Produced[phase] = produced
		}
		if len(source.Uncalled) > 0 && out.Uncalled == nil {
			out.Uncalled = map[string][]core.CanonicalID{}
		}
		for phase, capabilities := range source.Uncalled {
			out.Uncalled[phase] = append(out.Uncalled[phase], capabilities...)
		}
		if len(source.Calls) > 0 && out.Calls == nil {
			out.Calls = map[string][]hook.CallOutcome{}
		}
		for phase, calls := range source.Calls {
			out.Calls[phase] = append(out.Calls[phase], calls...)
		}
	}
	out.CommandOutcome = before.CommandOutcome
	if after.CommandOutcome != "" {
		out.CommandOutcome = after.CommandOutcome
	}
	return out
}

// runAfterHalfRevertingOnRefusal consults the after-change phases, and puts the
// store back when one of them refuses.
//
// The effect was applied before this runs, so a refusal here comes after the
// change. Leaving the change in place would keep something the declaration did
// not accept; leaving the report silent about it would read as "nothing was
// applied". Both are avoided: the store is restored to the snapshot taken
// before the run, and the caller is told the effect was reverted.
//
// The reference rule never refuses, so the CLI's own runs do not reach the
// revert today. The path exists and is tested with a refusing rule, because the
// rule a plugin supplies is what makes it reachable — that is the boundary this
// function is written for.
func runAfterHalfRevertingOnRefusal(ctx context.Context, world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string, rule hook.Implementation, store *command.MemoryStore, snapshot command.StoreSnapshot, before *cliHookReport) (*cliHookReport, bool, error) {
	after, err := runDeclaredHooksHalf(ctx, world, applicationID, patternID, inputs, rule, hook.AppliedChangeReach{}, hookHalfAfter)
	merged := mergeHookReports(before, after)
	if err == nil {
		return merged, false, nil
	}
	store.Restore(snapshot)
	return merged, true, err
}
