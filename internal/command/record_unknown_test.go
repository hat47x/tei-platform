package command

import (
	"context"
	"errors"
	"testing"
)

type failingPostExecutionRecorder struct{}

func (failingPostExecutionRecorder) Record(context.Context, OperationRecord) error {
	return errors.New("operation record unavailable")
}

type failingObserveStageLifecycle struct{ *Lifecycle }

func (failingObserveStageLifecycle) Observe(string, string, int, string) error {
	return errors.New("observation journal unavailable")
}

type failingUnknownStageLifecycle struct{ *Lifecycle }

func (failingUnknownStageLifecycle) MarkUnknown(string, string, int, string) error {
	return errors.New("unknown journal unavailable")
}

// RC-03: even a successful reference UoW rollback does not establish that an
// independent Executor/Provider transaction was rolled back. Do not claim a
// clean rejection when the post-execution operation record fails.
func TestRecorderFailureAfterExecutorIsUnknown(t *testing.T) {
	lifecycle := NewLifecycle()
	executor := &sequentialExecutor{results: []Result{submittedResult(), submittedResult()}}
	boundary := LifecycleInvoker{
		Lifecycle: lifecycle,
		Invoker: Invoker{
			World:      commandWorld(),
			Executor:   executor,
			Authorizer: ReferenceAuthorizer{},
			UnitOfWork: &MemoryUnitOfWork{},
			Recorder:   failingPostExecutionRecorder{},
		},
	}
	options := DispatchOptions{CommandID: "record-unknown", Owner: "owner-a", Generation: 1}
	_, err := boundary.Invoke(context.Background(), request(), options)
	if code := executionCode(t, err); code != "COMMAND_RECORD_FAILED" {
		t.Fatalf("code=%s, want COMMAND_RECORD_FAILED", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("post-execution recorder failure lost effect uncertainty")
	}
	attempt, ok := lifecycle.Attempt(options.CommandID)
	if !ok || attempt.Stage != StageUnknown {
		t.Fatalf("attempt=%+v, found=%t; want unknown", attempt, ok)
	}
	if lifecycle.MayReplay(options.CommandID) {
		t.Fatal("unknown recorder failure was reported as replayable")
	}
	if _, err := boundary.Invoke(context.Background(), request(), options); err == nil {
		t.Fatal("the same owner/generation replayed a possibly committed command")
	}
	if executor.calls != 1 {
		t.Fatalf("executor calls=%d, want 1", executor.calls)
	}
}

// RC-05: the business Executor returned successfully, but writing the
// lifecycle observation failed. The result is not a clean failure or proof of
// a durable committed stage. The unobserved dispatched attempt still blocks
// a duplicate dispatch in the reference Lifecycle.
func TestLifecycleObserveFailurePreservesUnknownEffect(t *testing.T) {
	stages := &failingObserveStageLifecycle{Lifecycle: NewLifecycle()}
	boundary := LifecycleInvoker{
		Lifecycle: stages,
		Invoker: Invoker{
			World:      commandWorld(),
			Authorizer: ReferenceAuthorizer{},
			Executor:   fakeExecutor{result: submittedResult()},
		},
	}
	options := DispatchOptions{CommandID: "observe-failure", Owner: "owner-a", Generation: 1}
	result, err := boundary.Invoke(context.Background(), request(), options)
	if code := executionCode(t, err); code != "COMMAND_LIFECYCLE_RECORD_FAILED" {
		t.Fatalf("code=%s, want COMMAND_LIFECYCLE_RECORD_FAILED", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("post-execution lifecycle write failure lost effect uncertainty")
	}
	if result.Outcome == nil || result.Outcome.Name != "submitted" {
		t.Fatalf("business result was not retained: %+v", result)
	}
	attempt, ok := stages.Attempt(options.CommandID)
	if !ok || attempt.Stage != StageDispatched {
		t.Fatalf("attempt=%+v, found=%t; unrecorded outcome must not be marked observed", attempt, ok)
	}
	if stages.MayReplay(options.CommandID) {
		t.Fatal("dispatched command was advertised as replayable after observation failed")
	}
	if _, err := stages.Dispatch(options.CommandID, options.Owner, options.Generation); err == nil {
		t.Fatal("duplicate dispatch was permitted after observation failed")
	}
}

// RC-05: when the first error already carries an unknown effect, a second
// failure in MarkUnknown must not erase that property in the returned error.
func TestLifecycleUnknownRecordFailureKeepsUncertainty(t *testing.T) {
	stages := &failingUnknownStageLifecycle{Lifecycle: NewLifecycle()}
	boundary := LifecycleInvoker{
		Lifecycle: stages,
		Invoker: Invoker{
			World:      commandWorld(),
			Authorizer: ReferenceAuthorizer{},
			Executor: fakeExecutor{
				err: NewUnknownExecutionError("PROVIDER_PARTIAL", "commit reply unavailable", nil),
			},
		},
	}
	options := DispatchOptions{CommandID: "unknown-record-failure", Owner: "owner-a", Generation: 1}
	_, err := boundary.Invoke(context.Background(), request(), options)
	if code := executionCode(t, err); code != "COMMAND_LIFECYCLE_RECORD_FAILED" {
		t.Fatalf("code=%s, want COMMAND_LIFECYCLE_RECORD_FAILED", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("original provider uncertainty was lost after stage write failure")
	}
	attempt, ok := stages.Attempt(options.CommandID)
	if !ok || attempt.Stage != StageDispatched {
		t.Fatalf("attempt=%+v, found=%t; unknown could not be durably recorded", attempt, ok)
	}
	if stages.MayReplay(options.CommandID) {
		t.Fatal("dispatched command was advertised as replayable after MarkUnknown failed")
	}
	if _, err := stages.Dispatch(options.CommandID, options.Owner, options.Generation); err == nil {
		t.Fatal("duplicate dispatch was permitted after MarkUnknown failed")
	}
}

// A pre-execution authorization failure must still be a definite rejection;
// the conservative post-execution rule must not make every preflight failure
// an unknown effect.
func TestPreExecutorAuthorizationFailureStaysRejected(t *testing.T) {
	lifecycle := NewLifecycle()
	executor := &sequentialExecutor{}
	boundary := LifecycleInvoker{
		Lifecycle: lifecycle,
		Invoker: Invoker{
			World:    commandWorld(),
			Executor: executor,
		},
	}
	options := DispatchOptions{CommandID: "preflight-reject", Owner: "owner-a", Generation: 1}
	_, err := boundary.Invoke(context.Background(), request(), options)
	if code := executionCode(t, err); code != "COMMAND_AUTHORIZER_UNAVAILABLE" {
		t.Fatalf("code=%s, want COMMAND_AUTHORIZER_UNAVAILABLE", code)
	}
	if EffectMayBeUnknown(err) {
		t.Fatal("pre-execution authorization refusal became an unknown effect")
	}
	attempt, ok := lifecycle.Attempt(options.CommandID)
	if !ok || attempt.Stage != StageObserved || attempt.Outcome != OutcomeRejected {
		t.Fatalf("attempt=%+v, found=%t; want observed rejection", attempt, ok)
	}
	if executor.calls != 0 {
		t.Fatalf("executor calls=%d, want 0", executor.calls)
	}
}

// The MayReplay query must agree with Dispatch for an in-flight attempt.
func TestLifecycleDispatchedIsNotReplayable(t *testing.T) {
	lifecycle := NewLifecycle()
	if !lifecycle.MayReplay("in-flight") {
		t.Fatal("an absent command was not initially dispatchable")
	}
	if _, err := lifecycle.Dispatch("in-flight", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if lifecycle.MayReplay("in-flight") {
		t.Fatal("an in-flight command was advertised as replayable")
	}
	if _, err := lifecycle.Dispatch("in-flight", "owner-a", 1); err == nil {
		t.Fatal("an in-flight command was re-dispatched")
	}
}
