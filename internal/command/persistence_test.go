package command

import (
	"context"
	"errors"
	"testing"

	"tei/internal/semantic"
)

type recordingObserver struct {
	called  bool
	effect  semantic.DataEffect
	request Request
	err     error
}

func (o *recordingObserver) Observe(_ context.Context, request Request, declared semantic.DataEffect) error {
	o.called = true
	o.request = request
	o.effect = declared
	return o.err
}

// TestInvokerReportsPersistenceNotApplicableWithoutADeclaredEffect checks that
// a command with no declared effect does not claim a persistence observation.
func TestInvokerReportsPersistenceNotApplicableWithoutADeclaredEffect(t *testing.T) {
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      effectlessCommandWorld(),
		Executor:   fakeExecutor{result: Result{}},
		UnitOfWork: &MemoryUnitOfWork{},
	}
	result, err := invoker.Invoke(context.Background(), requestForEffectlessWorld())
	if err != nil {
		t.Fatal(err)
	}
	if result.Persistence != PersistenceNotApplicable {
		t.Fatalf("persistence=%q, want not-applicable", result.Persistence)
	}
}

// TestInvokerReportsPersistenceNotObservedWithoutAnObserver checks that a
// successful Commit with no observer is recorded as not-observed, never as
// observed and never as "not persisted".
func TestInvokerReportsPersistenceNotObservedWithoutAnObserver(t *testing.T) {
	invoker := Invoker{
		Authorizer:   ReferenceAuthorizer{},
		World:        commandWorld(),
		Executor:     fakeExecutor{result: Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}},
		UnitOfWork:   &MemoryUnitOfWork{},
		DurableStore: true,
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.Persistence != PersistenceNotObserved {
		t.Fatalf("persistence=%q, want not-observed", result.Persistence)
	}
}

// TestInvokerObservesPersistenceWithAnIndependentObserver checks that an
// observer's success becomes observed, and that it receives the declared effect
// rather than the executor's report.
func TestInvokerObservesPersistenceWithAnIndependentObserver(t *testing.T) {
	observer := &recordingObserver{}
	invoker := Invoker{
		Authorizer:          ReferenceAuthorizer{},
		World:               commandWorld(),
		Executor:            fakeExecutor{result: Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}},
		UnitOfWork:          &MemoryUnitOfWork{},
		DurableStore:        true,
		PersistenceObserver: observer,
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.Persistence != PersistenceObserved {
		t.Fatalf("persistence=%q, want observed", result.Persistence)
	}
	if !observer.called || observer.effect.Kind != semantic.DataEffectUpdate || observer.effect.Data != "sales.order" {
		t.Fatalf("observer=%+v", observer)
	}
}

// TestInvokerKeepsAnObserverFailureAsNotObserved checks that a failed
// independent read is recorded as not-observed, not as a Command failure: the
// Commit already happened.
func TestInvokerKeepsAnObserverFailureAsNotObserved(t *testing.T) {
	observer := &recordingObserver{err: errors.New("read replica is behind")}
	invoker := Invoker{
		Authorizer:          ReferenceAuthorizer{},
		World:               commandWorld(),
		Executor:            fakeExecutor{result: Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}},
		UnitOfWork:          &MemoryUnitOfWork{},
		DurableStore:        true,
		PersistenceObserver: observer,
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.Persistence != PersistenceNotObserved {
		t.Fatalf("persistence=%q, want not-observed", result.Persistence)
	}
}

// TestInvokerDoesNotObserveAfterARolledBackExecution checks that a failed
// execution never runs the independent observer; there is nothing to observe.
func TestInvokerDoesNotObserveAfterARolledBackExecution(t *testing.T) {
	observer := &recordingObserver{}
	invoker := Invoker{
		Authorizer:          ReferenceAuthorizer{},
		World:               commandWorld(),
		Executor:            fakeExecutor{err: errors.New("write failed")},
		UnitOfWork:          &MemoryUnitOfWork{},
		PersistenceObserver: observer,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err == nil {
		t.Fatal("failed execution was accepted")
	}
	if observer.called {
		t.Fatal("observer ran after a failed execution")
	}
}
