package main

import (
	"context"
	"errors"

	"tei/internal/application"
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
type cliHookReport struct {
	// Pattern is the pattern whose interaction declares the hooks.
	Pattern core.CanonicalID `json:"pattern"`
	// Executed and Warned list the phases that ran, in execution order.
	Executed []string `json:"executed,omitempty"`
	Warned   []string `json:"warned,omitempty"`
	// Skipped lists the declared phases this run does not reach: after-change
	// phases (the reference run writes nothing) and phases whose declared reads
	// the caller did not supply.
	Skipped []string `json:"skipped,omitempty"`
	// NotEvaluated lists the phases that were skipped because a declared read
	// had no value, so a reader does not mistake them for a pass.
	NotEvaluated []string `json:"notEvaluated,omitempty"`
	// Reasons names why a phase was not evaluated, keyed by phase.
	Reasons map[string]string `json:"reasons,omitempty"`
	// Refused names the phase that stopped the run, when one did.
	Refused string `json:"refused,omitempty"`
	// Outcome is the business outcome a refusal corresponds to, when the hook
	// named one. It is empty for an execution failure.
	Outcome string `json:"outcome,omitempty"`
	// Note states the boundary: running a declared hook is not evidence that a
	// rule plugin decided it, and it does not make the command effect observed.
	Note string `json:"note"`
}

const cliHookNote = "the reference rule evaluates the declared reads only; running it does not accept the rule, observe the command effect, or grant authority"

const (
	cliHookSkipAfterChange = "the reference run writes nothing, so there is no after-change point to run this phase at"
	cliHookSkipNoSource    = "the reference run has no data source for the declared read, so this phase was not evaluated"
	cliHookSkipUnsupported = "the runner order does not name this phase, so it was not evaluated"
)

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
func runDeclaredHooksWithRule(ctx context.Context, world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string, rule hook.ReferenceRule) (*cliHookReport, error) {
	declarations := declaredHooks(world, applicationID, patternID)
	if len(declarations) == 0 {
		return nil, nil
	}
	report := &cliHookReport{
		Pattern: patternID,
		Note:    cliHookNote,
		Reasons: map[string]string{},
	}
	runner := hook.Runner{
		World:          world,
		Implementation: rule,
	}
	for _, phase := range hook.PhaseOrder {
		declaration, declared := declarations[phase]
		if !declared {
			continue
		}
		if !beforeChange(phase) {
			report.Skipped = append(report.Skipped, phase)
			report.Reasons[phase] = cliHookSkipAfterChange
			continue
		}
		reads, missing := readsFromInputs(declaration.Reads, inputs)
		if missing != "" {
			report.Skipped = append(report.Skipped, phase)
			report.NotEvaluated = append(report.NotEvaluated, phase)
			report.Reasons[phase] = cliHookSkipNoSource + ": " + missing
			continue
		}
		invocation := hook.Invocation{
			Application: applicationID,
			Pattern:     patternID,
			Phase:       phase,
			Reads:       reads,
		}
		// RunWithDeclaredReads applies hook.ReadsFor, so a value the phase does
		// not declare is refused here rather than widening the hook's view.
		err := runner.RunWithDeclaredReads(ctx, invocation)
		if err == nil {
			report.Executed = append(report.Executed, phase)
			continue
		}
		var warning *hook.Warning
		if errors.As(err, &warning) {
			report.Warned = append(report.Warned, phase)
			continue
		}
		report.Refused = phase
		var notSatisfied *hook.NotSatisfied
		if errors.As(err, &notSatisfied) {
			report.Outcome = notSatisfied.Outcome
		}
		return report, err
	}
	// A phase the runner order does not name would be skipped by the loop
	// without a trace. Record it instead, so a declaration is never silently
	// dropped.
	for phase := range declarations {
		if knownPhase(phase) {
			continue
		}
		report.Skipped = append(report.Skipped, phase)
		report.Reasons[phase] = cliHookSkipUnsupported
	}
	return report, nil
}

// knownPhase reports whether the runner order names a phase.
func knownPhase(phase string) bool {
	for _, known := range hook.PhaseOrder {
		if known == phase {
			return true
		}
	}
	return false
}

// declaredHooks returns the hook declarations of a pattern by phase.
func declaredHooks(world ir.World, applicationID, patternID core.CanonicalID) map[string]*application.Hook {
	for i := range world.Applications {
		if world.Applications[i].ID != applicationID {
			continue
		}
		for _, pattern := range world.Applications[i].Patterns {
			if pattern.ID != patternID || pattern.Interaction == nil {
				continue
			}
			out := make(map[string]*application.Hook, len(pattern.Interaction.Hooks))
			for j := range pattern.Interaction.Hooks {
				phase := pattern.Interaction.Hooks[j].Phase
				if _, exists := out[phase]; exists {
					continue
				}
				out[phase] = &pattern.Interaction.Hooks[j]
			}
			return out
		}
	}
	return nil
}

// readsFromInputs fills a hook's declared reads from the command inputs. The
// first attribute without a value is returned so the reason can name it; an
// empty string means every declared read had a value.
func readsFromInputs(declared []core.CanonicalID, inputs map[string]string) (map[core.CanonicalID]string, string) {
	values := make(map[core.CanonicalID]string, len(declared))
	for _, attribute := range declared {
		value, ok := inputs[string(attribute)]
		if !ok {
			return nil, string(attribute)
		}
		values[attribute] = value
	}
	return values, ""
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

// beforeChange reports whether a phase occurs before the change. These are the
// phases a reference run can reach.
func beforeChange(phase string) bool {
	switch phase {
	case "before-read", "before-create", "before-update", "before-delete":
		return true
	}
	return false
}
