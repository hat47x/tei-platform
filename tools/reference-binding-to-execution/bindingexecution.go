// Package bindingexecution is a read-only reference that closes the gap between
// the Environment's deployment-executor binding (validation and planning) and
// actual execution.
//
// The current product validates the binding and writes it into a plan (JSON),
// but the reference implementations that actually run an executor use a
// hard-coded CapabilityID and never consult the binding. This reference asks
// what it would take to resolve the binding and then execute, without inventing
// a second selection mechanism, a new resolver or any execution permission.
//
// It is deliberately narrow:
//   - it reads the accepted Workspace, resolves the binding with the existing
//     routing.RouteCommand, and executes through the existing routing.Router
//   - it refuses when the environment has no unambiguous binding
//   - it records what it actually ran, and nothing more
//
// It does not authenticate a caller, does not issue authority, and does not
// treat an executor acknowledgement as proof that external state exists.
package bindingexecution

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/deployment/executor"
	"tei/internal/deployment/routing"
	"tei/internal/environment"
	"tei/internal/workspace"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.binding-to-execution/v1"
	// Authority states what the report is allowed to claim.
	Authority = "derived-reference"

	RuleID      = "tei.reference.binding-to-execution"
	RuleVersion = "v1"
)

// rulePolicy is covered by the fingerprint.
const rulePolicy = `binding-to-execution/v1
selection: routing.RouteCommand over the Environment's deployment-executor binding
dispatch: routing.Router.Dispatch with an explicit executor registry
refused when: the binding is missing, ambiguous, or names an unregistered executor
after dispatch: ranCount is actual Execute entry; refusedCount is only before Execute
unverified: Execute returned error or an acknowledgement that failed binding validation
not verified: that the executor's acknowledgement reflects external state
not issued: authentication, authority, permission, or a durable receipt
ordering: commands are processed in ActionID order, as routing.RouteCommands does
`

// Refusal codes. They name the reason the reference did not run, which is
// distinct from an executor failure.
const (
	RefusedUnknownEnvironment = "UNKNOWN_ENVIRONMENT"
	RefusedBindingMissing     = "EXECUTOR_BINDING_MISSING"
	RefusedBindingAmbiguous   = "EXECUTOR_BINDING_AMBIGUOUS"
	RefusedNotRegistered      = "EXECUTOR_NOT_REGISTERED"
)

// Outcome is one command's result.
type Outcome struct {
	ActionID        string `json:"actionId"`
	ExecutorID      string `json:"executorId"`
	AttemptID       string `json:"attemptId,omitempty"`
	RawOutcome      string `json:"outcome,omitempty"`
	BindingResolved bool   `json:"bindingResolved"`
	// ExecutorInvoked records entry into Executor.Execute only. It does not
	// establish an external effect, a valid result or a durable receipt.
	ExecutorInvoked bool `json:"executorInvoked,omitempty"`
	// Error records a refusal or an executor error as text. It is never a
	// success.
	Error string `json:"error,omitempty"`
}

// Report is the whole run.
type Report struct {
	Schema          string    `json:"schema"`
	Authority       string    `json:"authority"`
	RuleID          string    `json:"ruleId"`
	RuleVersion     string    `json:"ruleVersion"`
	RuleFingerprint string    `json:"ruleFingerprint"`
	Environment     string    `json:"environment"`
	Outcomes        []Outcome `json:"outcomes"`
	// Ran counts entries into Executor.Execute, regardless of error or result
	// validity. Refused is strictly before that call. Unverified is a subset
	// of Ran: the executor was called but no valid acknowledgement followed.
	// None of these counters establishes an external business effect.
	Ran        int      `json:"ranCount"`
	Refused    int      `json:"refusedCount"`
	Unverified int      `json:"unverifiedCount"`
	Notes      []string `json:"notes"`
}

// Run resolves the environment's binding for each command and dispatches
// through the existing Router. The registry is supplied by the caller (the
// Host owns it); this reference never constructs an executor itself.
func Run(
	root string, environmentID core.CanonicalID, cmds []executor.Command,
	executors map[core.CanonicalID]executor.Executor,
) (Report, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return Report{}, fmt.Errorf("load Workspace: %w", err)
	}
	env, ok := findEnvironment(loaded.World.Environments, environmentID)
	if !ok {
		return Report{
			Schema: Schema, Authority: Authority,
			RuleID: RuleID, RuleVersion: RuleVersion,
			RuleFingerprint: fingerprint(),
			Environment:     string(environmentID),
			Outcomes:        []Outcome{},
			Notes:           notes(),
		}, nil
	}
	report := Report{
		Schema: Schema, Authority: Authority,
		RuleID: RuleID, RuleVersion: RuleVersion,
		RuleFingerprint: fingerprint(),
		Environment:     string(environmentID),
		Outcomes:        []Outcome{},
		Notes:           notes(),
	}
	// Match routing.RouteCommands' ordering so a reader can compare the plan
	// and the run line by line.
	ordered := append([]executor.Command(nil), cmds...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ActionID < ordered[j].ActionID })

	for _, cmd := range ordered {
		// Ask the same question the plan asks, before anything runs.
		routed, routeErr := routing.RouteCommand(env, cmd)
		if routeErr != nil {
			report.Refused++
			report.Outcomes = append(report.Outcomes, Outcome{
				ActionID: cmd.ActionID,
				Error:    RefusedBindingMissing + ": " + routeErr.Error(),
			})
			continue
		}
		if _, ok := executors[routed.ExecutorID]; !ok {
			report.Refused++
			report.Outcomes = append(report.Outcomes, Outcome{
				ActionID:   cmd.ActionID,
				ExecutorID: string(routed.ExecutorID),
				Error:      RefusedNotRegistered + ": the binding names an executor this Host did not register",
			})
			continue
		}
		// Router.Dispatch may refuse an unregistered, mismatched or unsupported
		// Executor before Execute. Observe the actual Execute entry without
		// changing the existing selection logic or the caller-owned registry.
		witness := &executorInvocationWitness{Executor: executors[routed.ExecutorID]}
		router := routing.Router{Executors: map[core.CanonicalID]executor.Executor{
			routed.ExecutorID: witness,
		}}
		result, dispatchErr := router.Dispatch(context.Background(), env, cmd)
		if witness.invoked {
			report.Ran++
		}
		if dispatchErr != nil {
			if witness.invoked {
				report.Unverified++
			} else {
				report.Refused++
			}
			report.Outcomes = append(report.Outcomes, Outcome{
				ActionID:        cmd.ActionID,
				ExecutorID:      string(routed.ExecutorID),
				ExecutorInvoked: witness.invoked,
				Error:           dispatchErr.Error(),
			})
			continue
		}
		if !witness.invoked {
			return Report{}, fmt.Errorf("routing returned an acknowledgement without entering Executor.Execute for action %s", cmd.ActionID)
		}
		// A result from another executor is invalid, but Execute already
		// happened and may have changed external state.
		if validateErr := routing.ValidateResult(env, cmd, result); validateErr != nil {
			report.Unverified++
			report.Outcomes = append(report.Outcomes, Outcome{
				ActionID:        cmd.ActionID,
				ExecutorID:      string(routed.ExecutorID),
				ExecutorInvoked: true,
				Error:           validateErr.Error(),
			})
			continue
		}
		report.Outcomes = append(report.Outcomes, Outcome{
			ActionID:        cmd.ActionID,
			ExecutorID:      result.ExecutorID,
			AttemptID:       result.AttemptID,
			RawOutcome:      string(result.Outcome),
			BindingResolved: true,
			ExecutorInvoked: true,
		})
	}
	return report, nil
}

// executorInvocationWitness observes the call boundary only. It does not
// grant authority, inspect effects, retry or validate a provider artifact.
type executorInvocationWitness struct {
	executor.Executor
	invoked bool
}

func (w *executorInvocationWitness) Execute(ctx context.Context, cmd executor.Command) (executor.Result, error) {
	w.invoked = true
	return w.Executor.Execute(ctx, cmd)
}

func findEnvironment(envs []environment.Environment, id core.CanonicalID) (environment.Environment, bool) {
	for _, env := range envs {
		if env.ID == id {
			return env, true
		}
	}
	return environment.Environment{}, false
}

func fingerprint() string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))) }

func notes() []string {
	return []string{
		"ranCount counts Executor.Execute entry, refusedCount only pre-invocation and unverifiedCount post-invocation errors or invalid acknowledgements",
		"an acknowledgement is the executor accepting the intent, not proof that external state changed",
		"the executor registry is supplied by the Host; this reference constructs no executor",
		"the environment binding selects the executor; Supports is only a conformance check",
		"no authentication, authority, permission or durable receipt is issued here",
		"never infer no-effect, safe automatic retry or committed state from an unverified acknowledgement",
	}
}

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
