package studiohost

import (
	"context"

	"tei/internal/applicationview"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/operations"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// CommandPlanResponse is a read-only command plan for the Studio bridge.
type CommandPlanResponse struct {
	OK      bool                         `json:"ok"`
	Value   *applicationview.CommandPlan `json:"value,omitempty"`
	Failure *Failure                     `json:"failure,omitempty"`
}

// CommandRunValue is a reference execution of a command. It has no durable effect.
type CommandRunValue struct {
	Plan    applicationview.CommandPlan `json:"plan"`
	Mode    string                      `json:"mode"`
	Outcome *semantic.Outcome           `json:"outcome,omitempty"`
	// OutcomeUndetermined states why no outcome was returned although the
	// capability declares some.
	OutcomeUndetermined string               `json:"outcomeUndetermined,omitempty"`
	Effect              *semantic.DataEffect `json:"effect,omitempty"`
	// Committed reports that the configured UnitOfWork returned from Commit.
	// The reference host uses MemoryUnitOfWork, which does not write data (#2998).
	Committed bool `json:"committed"`
	// Persisted is true only when an independent observer saw the declared
	// effect after Commit. A reference execution has no observer.
	Persisted bool `json:"persisted"`
	// Persistence separates "not applicable" (reference) from "not observed"
	// and "observed".
	Persistence string `json:"persistence"`
	// Durability names the storage boundary: "none" for the reference executor.
	// EffectReverted is true when the effect was applied and then put back
	// because an after-change phase refused.
	EffectReverted bool                      `json:"effectReverted,omitempty"`
	Durability     string                    `json:"durability"`
	Records        []command.OperationRecord `json:"records,omitempty"`
	// Differences keeps the applied operations the declaration does not
	// describe, so the view can show the gap beside the effect.
	Differences []command.Difference `json:"differences,omitempty"`
	// AppliedUnobserved is true when the executor did not enumerate its
	// operations, so `differences` being empty is "not observed".
	AppliedUnobserved bool `json:"appliedUnobserved,omitempty"`
	// ReportedEffectMissing is true when the capability declared an effect but
	// the executor reported none. It is recorded, not rejected (#2996).
	ReportedEffectMissing bool `json:"reportedEffectMissing,omitempty"`
	// AuthorizationMode names how the run was authorized. The Studio reference
	// host uses reference-allow-all when no capability allow list is supplied.
	AuthorizationMode string `json:"authorizationMode,omitempty"`
	// Hooks records the pattern hooks this run consulted before the command.
	// It mirrors the CLI report: a phase that could not be evaluated is named
	// with a reason instead of being shown as a pass.
	Hooks *HookRunReport `json:"hooks,omitempty"`
}

// CommandRunResponse is the result of a reference execution.
type CommandRunResponse struct {
	OK      bool             `json:"ok"`
	Value   *CommandRunValue `json:"value,omitempty"`
	Failure *Failure         `json:"failure,omitempty"`
}

func (h *ReadOnly) loadWorkspace(workspaceID string) (workspace.Result, *Failure) {
	root, failure := h.workspace(workspaceID)
	if failure != nil {
		return workspace.Result{}, failure
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return workspace.Result{}, &Failure{Code: "WORKSPACE_LOAD_FAILED", Stage: "command", Message: err.Error()}
	}
	return loaded, nil
}

// PlanCommand resolves a command's inputs without executing it.
func (h *ReadOnly) PlanCommand(workspaceID, applicationID, patternID, capabilityID string, inputs map[string]string) CommandPlanResponse {
	loaded, failure := h.loadWorkspace(workspaceID)
	if failure != nil {
		return CommandPlanResponse{Failure: failure}
	}
	plan, err := applicationview.BuildCommandPlan(loaded.World, core.CanonicalID(applicationID), core.CanonicalID(patternID), core.CanonicalID(capabilityID), inputs)
	if err != nil {
		return CommandPlanResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "command", Message: err.Error()}}
	}
	return CommandPlanResponse{OK: true, Value: &plan}
}

// ExecuteCommand runs the reference executor and unit of work. It does not
// persist data or call an external service.
func (h *ReadOnly) ExecuteCommand(workspaceID, applicationID, patternID, capabilityID string, inputs map[string]string) CommandRunResponse {
	return h.ExecuteCommandWithPeriods(workspaceID, applicationID, patternID, capabilityID, inputs, nil)
}

// ExecuteCommandWithPeriods runs the reference executor with caller-supplied
// period states. A closed period rejects effects listed in closedRejects.
func (h *ReadOnly) ExecuteCommandWithPeriods(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string) CommandRunResponse {
	return h.ExecuteCommandWithPolicy(workspaceID, applicationID, patternID, capabilityID, inputs, periods, nil)
}

// ExecuteCommandWithPolicy additionally applies a reference capability
// allow list before the executor runs.
func (h *ReadOnly) ExecuteCommandWithPolicy(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string, allowed []string) CommandRunResponse {
	return h.executeCommand(workspaceID, applicationID, patternID, capabilityID, inputs, periods, allowed, nil)
}

// ExecuteCommandWithLifecycle runs the same reference execution under the
// host-composed durable stage store. A repeated attempt is refused with
// COMMAND_LIFECYCLE_REFUSED instead of executing again.
func (h *ReadOnly) ExecuteCommandWithLifecycle(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string, allowed []string, commandID, owner string, generation int) CommandRunResponse {
	if h.CommandLifecycle == nil {
		return CommandRunResponse{Failure: &Failure{Code: "COMMAND_LIFECYCLE_UNAVAILABLE", Stage: "command", Message: "no command lifecycle is composed for this host"}}
	}
	if commandID == "" || owner == "" || generation < 1 {
		return CommandRunResponse{Failure: &Failure{Code: "INVALID_LIFECYCLE_ATTEMPT", Stage: "command", Message: "command id, owner, and a positive generation are required"}}
	}
	return h.executeCommand(workspaceID, applicationID, patternID, capabilityID, inputs, periods, allowed,
		&command.DispatchOptions{CommandID: commandID, Owner: owner, Generation: generation})
}

// recordDifferences returns the differences of the last operation record, so the
// view can show them without exposing the whole log.
func recordDifferences(records []command.OperationRecord) []command.Difference {
	if len(records) == 0 {
		return nil
	}
	return records[len(records)-1].Differences
}

// recordAppliedUnobserved reports whether the executor said what it applied.
// A command that declares no effect has nothing to compare, so the absence of
// records is not an unobserved difference check.
func recordReportedEffectMissing(records []command.OperationRecord) bool {
	if len(records) == 0 {
		return false
	}
	return records[len(records)-1].ReportedEffectMissing
}

func recordAppliedUnobserved(records []command.OperationRecord, effectDeclared bool) bool {
	if !effectDeclared {
		return false
	}
	if len(records) == 0 {
		return true
	}
	return records[len(records)-1].AppliedUnobserved
}

// ExecuteCommandWithReferenceStore runs the command against an in-memory
// reference store, so the after-change phases have a point to run at.
//
// The reference execution writes nothing durable: the store is created for this
// run and dropped when it ends, and the response keeps saying so. The change is
// still real for the length of the run, which is what `after-*` needs; without a
// store there is no after-change point and those phases are reported as skipped.
func (h *ReadOnly) ExecuteCommandWithReferenceStore(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string, allowed []string) CommandRunResponse {
	return h.executeCommandWithStore(workspaceID, applicationID, patternID, capabilityID, inputs, periods, allowed, nil, true)
}

func (h *ReadOnly) executeCommand(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string, allowed []string, attempt *command.DispatchOptions) CommandRunResponse {
	return h.executeCommandWithStore(workspaceID, applicationID, patternID, capabilityID, inputs, periods, allowed, attempt, false)
}

func (h *ReadOnly) executeCommandWithStore(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string, allowed []string, attempt *command.DispatchOptions, applyToStore bool) CommandRunResponse {
	loaded, failure := h.loadWorkspace(workspaceID)
	if failure != nil {
		return CommandRunResponse{Failure: failure}
	}
	plan, err := applicationview.BuildCommandPlan(loaded.World, core.CanonicalID(applicationID), core.CanonicalID(patternID), core.CanonicalID(capabilityID), inputs)
	if err != nil {
		return CommandRunResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "command", Message: err.Error()}}
	}
	// A declared hook runs before the command when this reference execution can
	// reach it: a before-change phase whose declared reads the caller supplied.
	// Everything else is reported as skipped or not evaluated with a reason, so
	// a phase that did not run never reads as a phase that passed.
	var hookReport *HookRunReport
	var hookErr error
	var effectReverted bool
	if applyToStore {
		// The before-* phases decide before the change; the after-* phases are
		// consulted below, after the effect was applied.
		hookReport, hookErr = h.runHooksHalf(loaded.World, core.CanonicalID(applicationID), core.CanonicalID(patternID), inputs, hook.ReferenceRule{}, hook.ReferenceReach{}, hookHalfBefore)
	} else {
		hookReport, hookErr = h.runHooks(loaded.World, core.CanonicalID(applicationID), core.CanonicalID(patternID), inputs)
	}
	if hookErr != nil {
		// No input reaches this branch today: runHooks returns an error only
		// together with a report, and returns (nil, nil) for a pattern that
		// declares no hook (see TestCommandFailedHasNoKnownTrigger). The branch
		// stays as the boundary for a future runner that can fail without a
		// report; it is not a claim that the code is reachable.
		return CommandRunResponse{Failure: &Failure{Code: "COMMAND_FAILED", Stage: "command", Message: hookErr.Error()}}
	}
	referenceStore := command.NewMemoryStore()
	var executor command.Executor = command.ReferenceExecutor{World: loaded.World}
	if applyToStore {
		executor = command.ReferenceDataExecutor{World: loaded.World, Store: referenceStore}
	}
	invoker := command.Invoker{
		World:      loaded.World,
		Executor:   executor,
		UnitOfWork: &command.MemoryUnitOfWork{},
		Authorizer: command.ReferenceAuthorizer{},
	}
	if periods != nil {
		states := map[string]operations.PeriodState{}
		for id, value := range periods {
			state := operations.PeriodState(value)
			switch state {
			case operations.PeriodOpen, operations.PeriodClosing, operations.PeriodClosed:
				states[id] = state
			default:
				return CommandRunResponse{Failure: &Failure{Code: "INVALID_PERIOD_STATE", Stage: "command", Message: "period state must be open, closing, or closed"}}
			}
		}
		invoker.Periods = command.NewMapPeriodState(states)
	}
	recorder := &command.MemoryRecorder{}
	invoker.Recorder = recorder
	if len(allowed) > 0 {
		authorizer := make(command.AllowListAuthorizer, 0, len(allowed))
		for _, capability := range allowed {
			authorizer = append(authorizer, core.CanonicalID(capability))
		}
		invoker.Authorizer = authorizer
	}
	request := command.Request{
		Application: core.CanonicalID(applicationID),
		Pattern:     core.CanonicalID(patternID),
		Capability:  core.CanonicalID(capabilityID),
		Inputs:      inputs,
	}
	var result command.Result
	if attempt != nil {
		tracked := command.LifecycleInvoker{Invoker: invoker, Lifecycle: h.CommandLifecycle}
		result, err = tracked.Invoke(context.Background(), request, *attempt)
	} else {
		result, err = invoker.Invoke(context.Background(), request)
	}
	if err != nil {
		// The same rule the CLI uses: an ExecutionError already names the
		// reason. This entrance used to special-case only the lifecycle refusal,
		// so a missing input, a closed period or a failed commit all arrived as
		// COMMAND_FAILED (#3807).
		if code, message, ok := command.ReportedFailure(err); ok {
			return CommandRunResponse{Failure: &Failure{Code: code, Stage: "command", Message: message}}
		}
		return CommandRunResponse{Failure: &Failure{Code: "COMMAND_FAILED", Stage: "command", Message: err.Error()}}
	}
	if applyToStore {
		// The change is applied, so the after-* phases have a point to run at.
		// Their report is joined with the before-* one so the caller sees both
		// halves of the run. A refusal here comes after the change, so the store
		// is put back and the caller is told the effect was reverted.
		snapshot := referenceStore.Snapshot()
		after, reverted, afterErr := h.runAfterHalfRevertingOnRefusal(loaded.World, core.CanonicalID(applicationID), core.CanonicalID(patternID), inputs, hook.ReferenceRule{}, referenceStore, snapshot, hookReport)
		if after != nil && after.PhaseReport != nil {
			if hookReport == nil {
				hookReport = after
			} else {
				hookReport.PhaseReport.Merge(after.PhaseReport)
				if after.CommandOutcome != "" {
					hookReport.CommandOutcome = after.CommandOutcome
				}
			}
		}
		if afterErr != nil {
			return CommandRunResponse{Failure: &Failure{Code: "COMMAND_FAILED", Stage: "command", Message: afterErr.Error()}}
		}
		effectReverted = reverted
	}
	return CommandRunResponse{OK: true, Value: &CommandRunValue{
		Plan:                  plan,
		Mode:                  "reference",
		Outcome:               result.Outcome,
		OutcomeUndetermined:   result.OutcomeUndetermined,
		Effect:                result.Effect,
		Committed:             true,
		Persisted:             result.Persistence == command.PersistenceResourceObserved,
		Persistence:           string(result.Persistence),
		Durability:            "none",
		Records:               recorder.Records,
		Differences:           recordDifferences(recorder.Records),
		AppliedUnobserved:     recordAppliedUnobserved(recorder.Records, plan.Effect != nil),
		ReportedEffectMissing: recordReportedEffectMissing(recorder.Records),
		AuthorizationMode:     string(result.AuthorizationMode),
		Hooks:                 hookReport,
		EffectReverted:        effectReverted,
	}}
}
