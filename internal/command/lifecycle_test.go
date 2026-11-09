package command

import (
	"context"
	"errors"
	"testing"

	"tei/internal/semantic"
)

// #2564 → #1129: an unknown outcome must not be replayed by the same
// owner/generation. A takeover needs a higher generation and a new owner, and
// it does not claim success until Observe.
func TestLifecycleStopsReplayAfterUnknown(t *testing.T) {
	lifecycle := NewLifecycle()
	if _, err := lifecycle.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.MarkUnknown("cmd-1", "owner-a", 1, "response lost"); err != nil {
		t.Fatal(err)
	}
	if lifecycle.MayReplay("cmd-1") {
		t.Fatal("unknown outcome was reported as replayable")
	}
	if _, err := lifecycle.Dispatch("cmd-1", "owner-a", 1); err == nil {
		t.Fatal("the same owner/generation replayed an unknown command")
	}
	if _, err := lifecycle.Takeover("cmd-1", "owner-a", 2); err == nil {
		t.Fatal("takeover by the same owner was accepted")
	}
	if _, err := lifecycle.Takeover("cmd-1", "owner-b", 1); err == nil {
		t.Fatal("takeover with an equal generation was accepted")
	}
}

func TestLifecycleTakeoverRequiresObservationBeforeSuccess(t *testing.T) {
	lifecycle := NewLifecycle()
	if _, err := lifecycle.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.MarkUnknown("cmd-1", "owner-a", 1, "timeout"); err != nil {
		t.Fatal(err)
	}
	attempt, err := lifecycle.Takeover("cmd-1", "owner-b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.TakeoverFrom != 1 || attempt.Owner != "owner-b" || attempt.Stage != StageDispatched {
		t.Fatalf("unexpected takeover attempt: %+v", attempt)
	}
	if current, _ := lifecycle.Attempt("cmd-1"); current.Stage != StageDispatched || current.Outcome != "" {
		t.Fatalf("takeover claimed an outcome: %+v", current)
	}
	if err := lifecycle.Observe("cmd-1", "owner-b", 2, OutcomeRejected); err != nil {
		t.Fatal(err)
	}
	current, _ := lifecycle.Attempt("cmd-1")
	if current.Stage != StageObserved || current.Outcome != OutcomeRejected || current.TakeoverFrom != 1 {
		t.Fatalf("observation was not recorded: %+v", current)
	}
	if _, err := lifecycle.Dispatch("cmd-1", "owner-c", 3); err == nil {
		t.Fatal("an observed command was re-dispatched")
	}
}

func TestLifecycleObserveRequiresMatchingAttempt(t *testing.T) {
	lifecycle := NewLifecycle()
	if err := lifecycle.Observe("cmd-1", "owner-a", 1, OutcomeCommitted); err == nil {
		t.Fatal("observation without a dispatch was accepted")
	}
	if _, err := lifecycle.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.Dispatch("cmd-1", "owner-a", 2); err == nil {
		t.Fatal("a second dispatch while in flight was accepted")
	}
	if err := lifecycle.Observe("cmd-1", "owner-b", 1, OutcomeCommitted); err == nil {
		t.Fatal("observation from a different owner was accepted")
	}
	if err := lifecycle.Observe("cmd-1", "owner-a", 2, OutcomeCommitted); err == nil {
		t.Fatal("observation from a different generation was accepted")
	}
	if err := lifecycle.Observe("cmd-1", "owner-a", 1, ""); err == nil {
		t.Fatal("observation without an outcome was accepted")
	}
	if err := lifecycle.Observe("cmd-1", "owner-a", 1, OutcomeCommitted); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Observe("cmd-1", "owner-a", 1, OutcomeCommitted); err == nil {
		t.Fatal("an observed attempt was observed twice")
	}
}

func TestLifecycleRejectsTakeoverWithoutUnknownOutcome(t *testing.T) {
	lifecycle := NewLifecycle()
	if _, err := lifecycle.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.Takeover("cmd-1", "owner-b", 2); err == nil {
		t.Fatal("takeover without an unknown outcome was accepted")
	}
	if err := lifecycle.Observe("cmd-1", "owner-a", 1, OutcomeCommitted); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.Takeover("cmd-1", "owner-b", 2); err == nil {
		t.Fatal("takeover of an observed command was accepted")
	}
	if _, err := lifecycle.Takeover("cmd-absent", "owner-b", 2); err == nil {
		t.Fatal("takeover of an undispatched command was accepted")
	}
}

func TestEffectMayBeUnknown(t *testing.T) {
	if EffectMayBeUnknown(errors.New("plain")) {
		t.Fatal("a plain error was treated as an unknown effect")
	}
	if EffectMayBeUnknown(&ExecutionError{Code: "COMMAND_EXECUTION_FAILED"}) {
		t.Fatal("a rollback-safe execution error was treated as unknown")
	}
	if !EffectMayBeUnknown(NewUnknownExecutionError("PROVIDER_PARTIAL", "partial write", nil)) {
		t.Fatal("an explicitly unknown execution error was not detected")
	}
}

type sequentialExecutor struct {
	results []Result
	errs    []error
	calls   int
}

func (e *sequentialExecutor) Execute(context.Context, Request) (Result, error) {
	index := e.calls
	e.calls++
	if index < len(e.errs) && e.errs[index] != nil {
		return Result{}, e.errs[index]
	}
	if index < len(e.results) {
		return e.results[index], nil
	}
	return Result{}, nil
}

func submittedResult() Result {
	return Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}
}

func TestLifecycleInvokerRecordsObservedSuccess(t *testing.T) {
	lifecycle := NewLifecycle()
	invoker := LifecycleInvoker{
		Lifecycle: lifecycle,
		Invoker: Invoker{
			Authorizer: ReferenceAuthorizer{},
			World:      commandWorld(),
			Executor:   fakeExecutor{result: submittedResult()},
		},
	}
	result, err := invoker.Invoke(context.Background(), request(), DispatchOptions{
		CommandID: "cmd-1", Owner: "owner-a", Generation: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome == nil || result.Outcome.Name != "submitted" {
		t.Fatalf("result=%+v", result)
	}
	attempt, _ := lifecycle.Attempt("cmd-1")
	if attempt.Stage != StageObserved || attempt.Outcome != "submitted" {
		t.Fatalf("attempt=%+v", attempt)
	}
}

func TestLifecycleInvokerRecordsRejectedExecution(t *testing.T) {
	lifecycle := NewLifecycle()
	transaction := &fakeTransaction{}
	invoker := LifecycleInvoker{
		Lifecycle: lifecycle,
		Invoker: Invoker{
			Authorizer: ReferenceAuthorizer{},
			World:      commandWorld(),
			Executor:   fakeExecutor{err: errors.New("validation stopped the command")},
			UnitOfWork: fakeUnitOfWork{transaction: transaction},
		},
	}
	_, err := invoker.Invoke(context.Background(), request(), DispatchOptions{
		CommandID: "cmd-1", Owner: "owner-a", Generation: 1,
	})
	if code := executionCode(t, err); code != "COMMAND_EXECUTION_FAILED" {
		t.Fatalf("code=%s", code)
	}
	attempt, _ := lifecycle.Attempt("cmd-1")
	if attempt.Stage != StageObserved || attempt.Outcome != OutcomeRejected {
		t.Fatalf("attempt=%+v", attempt)
	}
	if transaction.rollbacks != 1 {
		t.Fatalf("rollbacks=%d", transaction.rollbacks)
	}
}

func TestLifecycleInvokerMarksCommitFailureUnknown(t *testing.T) {
	lifecycle := NewLifecycle()
	transaction := &fakeTransaction{commitErr: errors.New("connection reset after send")}
	executor := &sequentialExecutor{results: []Result{submittedResult(), submittedResult()}}
	invoker := LifecycleInvoker{
		Lifecycle: lifecycle,
		Invoker: Invoker{
			Authorizer: ReferenceAuthorizer{},
			World:      commandWorld(),
			Executor:   executor,
			UnitOfWork: fakeUnitOfWork{transaction: transaction},
		},
	}
	options := DispatchOptions{CommandID: "cmd-1", Owner: "owner-a", Generation: 1}
	_, err := invoker.Invoke(context.Background(), request(), options)
	if code := executionCode(t, err); code != "COMMAND_COMMIT_FAILED" {
		t.Fatalf("code=%s", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("a commit failure was not marked as a possibly applied effect")
	}
	attempt, _ := lifecycle.Attempt("cmd-1")
	if attempt.Stage != StageUnknown {
		t.Fatalf("commit failure stage=%s, want unknown", attempt.Stage)
	}
	if lifecycle.MayReplay("cmd-1") {
		t.Fatal("unknown command was reported as replayable")
	}
	// The same owner/generation cannot retry; success is not claimed.
	if _, err := invoker.Invoke(context.Background(), request(), options); err == nil {
		t.Fatal("the same attempt was replayed after an unknown outcome")
	}
	if attempt, _ := lifecycle.Attempt("cmd-1"); attempt.Stage != StageUnknown {
		t.Fatalf("refused replay changed the stage to %s", attempt.Stage)
	}

	// A new owner takes over with a higher generation and observes the outcome.
	transaction.commitErr = nil
	takeover := DispatchOptions{CommandID: "cmd-1", Owner: "owner-b", Generation: 2, Takeover: true}
	result, err := invoker.Invoke(context.Background(), request(), takeover)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome == nil || result.Outcome.Name != "submitted" {
		t.Fatalf("takeover result=%+v", result)
	}
	attempt, _ = lifecycle.Attempt("cmd-1")
	if attempt.Stage != StageObserved || attempt.Owner != "owner-b" || attempt.Generation != 2 || attempt.TakeoverFrom != 1 {
		t.Fatalf("takeover attempt=%+v", attempt)
	}
}

func TestLifecycleInvokerMarksProviderUnknown(t *testing.T) {
	lifecycle := NewLifecycle()
	invoker := LifecycleInvoker{
		Lifecycle: lifecycle,
		Invoker: Invoker{
			Authorizer: ReferenceAuthorizer{},
			World:      commandWorld(),
			Executor:   fakeExecutor{err: NewUnknownExecutionError("PROVIDER_PARTIAL", "write may have been applied", nil)},
		},
	}
	_, err := invoker.Invoke(context.Background(), request(), DispatchOptions{
		CommandID: "cmd-1", Owner: "owner-a", Generation: 1,
	})
	if code := executionCode(t, err); code != "COMMAND_EXECUTION_FAILED" {
		t.Fatalf("code=%s", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("the provider's unknown-effect marker was dropped at the command boundary")
	}
	attempt, _ := lifecycle.Attempt("cmd-1")
	if attempt.Stage != StageUnknown {
		t.Fatalf("stage=%s, want unknown", attempt.Stage)
	}
}

func TestLifecycleInvokerRequiresLifecycle(t *testing.T) {
	invoker := LifecycleInvoker{Invoker: Invoker{World: commandWorld(), Executor: fakeExecutor{}, Authorizer: ReferenceAuthorizer{}}}
	_, err := invoker.Invoke(context.Background(), request(), DispatchOptions{CommandID: "cmd-1", Owner: "owner-a", Generation: 1})
	if code := executionCode(t, err); code != "COMMAND_LIFECYCLE_UNAVAILABLE" {
		t.Fatalf("code=%s", code)
	}
}

type rollbackFailureTransaction struct {
	commitErr   error
	rollbackErr error
}

func (t rollbackFailureTransaction) Commit(context.Context) error   { return t.commitErr }
func (t rollbackFailureTransaction) Rollback(context.Context) error { return t.rollbackErr }

func TestInvokeMarksRollbackFailureUnknown(t *testing.T) {
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{err: errors.New("write failed")},
		UnitOfWork: fakeUnitOfWork{transaction: &fakeTransaction{}}, // rollback succeeds
	}
	_, err := invoker.Invoke(context.Background(), request())
	if EffectMayBeUnknown(err) {
		t.Fatal("a successful rollback was treated as an unknown effect")
	}

	failing := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{err: errors.New("write failed")},
		UnitOfWork: rollbackFailureUnitOfWork{transaction: rollbackFailureTransaction{rollbackErr: errors.New("connection lost")}},
	}
	_, err = failing.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_ROLLBACK_FAILED" {
		t.Fatalf("code=%s", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("a failed rollback was not treated as an unknown effect")
	}
}

type rollbackFailureUnitOfWork struct {
	transaction rollbackFailureTransaction
}

func (u rollbackFailureUnitOfWork) Begin(context.Context) (Transaction, error) {
	return u.transaction, nil
}
