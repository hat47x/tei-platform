package command

import (
	"context"
	"testing"

	"tei/internal/semantic"
)

// #2997: an Invoker without an Authorizer must not run the executor. The
// reference path has to opt in with ReferenceAuthorizer so a host cannot
// silently omit an authorization decision.
func TestInvokeRefusesAnUnconfiguredAuthorizer(t *testing.T) {
	executor := &recordingExecutor{result: submittedResult()}
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   executor,
		UnitOfWork: &MemoryUnitOfWork{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_AUTHORIZER_UNAVAILABLE" {
		t.Fatalf("code=%s", code)
	}
	if executor.called {
		t.Fatal("executor ran without an authorization decision")
	}
}

func TestReferenceAuthorizerIsAnExplicitAllowDecision(t *testing.T) {
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   &recordingExecutor{result: submittedResult()},
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: ReferenceAuthorizer{},
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
}

// #2997: the authorization decision comes before the period check, so an
// unauthorized caller does not learn whether a period is closed.
func TestInvokeAuthorizesBeforeCheckingThePeriod(t *testing.T) {
	state := NewMemoryPeriodState()
	state.SetClosed("sales.period", true)
	invoker := Invoker{
		World:      periodWorld([]string{"update"}),
		Executor:   &recordingExecutor{result: submittedResult()},
		Periods:    state,
		Authorizer: denyAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_UNAUTHORIZED" {
		t.Fatalf("code=%s", code)
	}
}

// #2997: a declared period without a period state is not "no rejection"; the
// boundary cannot check the declaration, so it refuses instead of guessing.
func TestInvokeRefusesADeclaredPeriodWithoutState(t *testing.T) {
	invoker := Invoker{
		World:      periodWorld([]string{"update"}),
		Executor:   &recordingExecutor{result: submittedResult()},
		Authorizer: ReferenceAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_PERIOD_STATE_UNAVAILABLE" {
		t.Fatalf("code=%s", code)
	}
}

// #2996: an undeclared effect is detected after the executor ran. The boundary
// cannot know that a UnitOfWork rolled the write back, because the Executor
// contract does not require the executor to join that transaction.
func TestUndeclaredEffectIsMarkedUnknown(t *testing.T) {
	lifecycle := NewLifecycle()
	invoker := LifecycleInvoker{
		Lifecycle: lifecycle,
		Invoker: Invoker{
			World:      commandWorld(),
			Executor:   effectExecutor{effect: &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}},
			UnitOfWork: &MemoryUnitOfWork{},
			Authorizer: ReferenceAuthorizer{},
		},
	}
	_, err := invoker.Invoke(context.Background(), request(), DispatchOptions{CommandID: "cmd-1", Owner: "owner-a", Generation: 1})
	if code := executionCode(t, err); code != "UNDECLARED_COMMAND_EFFECT" {
		t.Fatalf("code=%s", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("an undeclared effect was recorded as a clean rejection")
	}
	attempt, _ := lifecycle.Attempt("cmd-1")
	if attempt.Stage != StageUnknown {
		t.Fatalf("stage=%s, want unknown", attempt.Stage)
	}
	if lifecycle.MayReplay("cmd-1") {
		t.Fatal("an unknown command was reported as replayable")
	}
}

func TestUndeclaredEffectWithoutUnitOfWorkIsAlsoUnknown(t *testing.T) {
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   effectExecutor{effect: &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}},
		Authorizer: ReferenceAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "UNDECLARED_COMMAND_EFFECT" {
		t.Fatalf("code=%s", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("an undeclared effect with no transaction was recorded as a clean rejection")
	}
}

func TestUndeclaredOutcomeIsMarkedUnknown(t *testing.T) {
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "unknown", Payload: "sales.order"}}},
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: ReferenceAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "UNDECLARED_COMMAND_OUTCOME" {
		t.Fatalf("code=%s", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("an undeclared outcome was recorded as a clean rejection")
	}
}

func TestOutsideDeclaredKindIsMarkedUnknown(t *testing.T) {
	invoker := Invoker{
		World: commandWorld(),
		Executor: fakeExecutor{result: Result{
			Effect:  &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
			Applied: []AppliedOperation{{Kind: semantic.DataEffectDelete, Data: "sales.order"}},
		}},
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: ReferenceAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "UNDECLARED_COMMAND_EFFECT" {
		t.Fatalf("code=%s", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("an applied operation outside the declaration was recorded as a clean rejection")
	}
}

func TestMissingAppliedOperationIsMarkedUnknown(t *testing.T) {
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Applied: []AppliedOperation{{Kind: semantic.DataEffectUpdate, Data: "sales.order_line"}}}},
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: ReferenceAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "MISSING_COMMAND_EFFECT" {
		t.Fatalf("code=%s", code)
	}
	if !EffectMayBeUnknown(err) {
		t.Fatal("a missing applied operation was recorded as a clean rejection")
	}
}

// recordingExecutor counts calls without returning an effect, so a refused
// call can be distinguished from an executed one.
type recordingExecutor struct {
	called bool
	result Result
}

func (e *recordingExecutor) Execute(context.Context, Request) (Result, error) {
	e.called = true
	return e.result, nil
}
