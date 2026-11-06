package studiohost

import (
	"context"
	"errors"

	"tei/internal/applicationview"
	"tei/internal/command"
	"tei/internal/core"
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
	Effect  *semantic.DataEffect        `json:"effect,omitempty"`
	// Committed reports that the configured UnitOfWork returned from Commit.
	// The reference host uses MemoryUnitOfWork, which does not write data (#2998).
	Committed bool `json:"committed"`
	// Persisted is false for a reference execution.
	Persisted bool `json:"persisted"`
	// Durability names the storage boundary: "none" for the reference executor.
	Durability string                    `json:"durability"`
	Records    []command.OperationRecord `json:"records,omitempty"`
	// Differences keeps the applied operations the declaration does not
	// describe, so the view can show the gap beside the effect.
	Differences []command.Difference `json:"differences,omitempty"`
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

func (h *ReadOnly) executeCommand(workspaceID, applicationID, patternID, capabilityID string, inputs, periods map[string]string, allowed []string, attempt *command.DispatchOptions) CommandRunResponse {
	loaded, failure := h.loadWorkspace(workspaceID)
	if failure != nil {
		return CommandRunResponse{Failure: failure}
	}
	plan, err := applicationview.BuildCommandPlan(loaded.World, core.CanonicalID(applicationID), core.CanonicalID(patternID), core.CanonicalID(capabilityID), inputs)
	if err != nil {
		return CommandRunResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "command", Message: err.Error()}}
	}
	invoker := command.Invoker{
		World:      loaded.World,
		Executor:   command.ReferenceExecutor{World: loaded.World},
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
		var execution *command.ExecutionError
		if errors.As(err, &execution) && execution.Code == "COMMAND_LIFECYCLE_REFUSED" {
			return CommandRunResponse{Failure: &Failure{Code: "COMMAND_LIFECYCLE_REFUSED", Stage: "command", Message: execution.Message}}
		}
		return CommandRunResponse{Failure: &Failure{Code: "COMMAND_FAILED", Stage: "command", Message: err.Error()}}
	}
	return CommandRunResponse{OK: true, Value: &CommandRunValue{
		Plan:        plan,
		Mode:        "reference",
		Outcome:     result.Outcome,
		Effect:      result.Effect,
		Committed:   true,
		Persisted:   false,
		Durability:  "none",
		Records:     recorder.Records,
		Differences: recordDifferences(recorder.Records),
	}}
}
