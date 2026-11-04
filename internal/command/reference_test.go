package command

import (
	"context"
	"testing"
)

func TestReferenceExecutorSatisfiesInvokerContract(t *testing.T) {
	world := commandWorld()
	unitOfWork := &MemoryUnitOfWork{}
	invoker := Invoker{
		World:      world,
		Executor:   ReferenceExecutor{World: world},
		UnitOfWork: unitOfWork,
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome == nil || result.Outcome.Name != "submitted" {
		t.Fatalf("outcome=%+v", result.Outcome)
	}
	if result.Effect == nil || result.Effect.Kind != "update" {
		t.Fatalf("effect=%+v", result.Effect)
	}
	commits, rollbacks := unitOfWork.Counts()
	if commits != 1 || rollbacks != 0 {
		t.Fatalf("commits=%d rollbacks=%d", commits, rollbacks)
	}
}

func TestReferenceExecutorContractRollsBackOnExecutionFailure(t *testing.T) {
	world := commandWorld()
	unitOfWork := &MemoryUnitOfWork{}
	invoker := Invoker{
		World:      world,
		Executor:   fakeExecutor{err: context.Canceled},
		UnitOfWork: unitOfWork,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err == nil {
		t.Fatal("execution failure was accepted")
	}
	commits, rollbacks := unitOfWork.Counts()
	if commits != 0 || rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", commits, rollbacks)
	}
}

func TestReferenceUnitOfWorkReportsBeginFailure(t *testing.T) {
	unitOfWork := &MemoryUnitOfWork{FailBegin: true}
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   ReferenceExecutor{World: commandWorld()},
		UnitOfWork: unitOfWork,
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_TRANSACTION_BEGIN_FAILED" {
		t.Fatalf("code=%s", code)
	}
}
