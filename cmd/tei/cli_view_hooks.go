package main

import (
	"context"

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
func runDeclaredHooksWithRule(ctx context.Context, world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string, rule hook.ReferenceRule) (*cliHookReport, error) {
	runner := hook.Runner{World: world, Implementation: rule}
	report, err := runner.RunReachable(ctx, applicationID, patternID, attributeValues(inputs), hook.ReferenceReach{})
	if report == nil {
		return nil, err
	}
	return &cliHookReport{PhaseReport: report, Pattern: patternID, Note: cliHookNote}, err
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
