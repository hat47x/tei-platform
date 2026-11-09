package command

import (
	"context"
	"errors"
	"testing"

	"tei/internal/semantic"
)

type effectExecutor struct {
	effect *semantic.DataEffect
}

func (e effectExecutor) Execute(context.Context, Request) (Result, error) {
	return Result{Effect: e.effect}, nil
}

func TestInvokerRejectsUndeclaredEffect(t *testing.T) {
	world := commandWorld()
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   effectExecutor{effect: &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}},
		UnitOfWork: &MemoryUnitOfWork{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	var executionError *ExecutionError
	if !errors.As(err, &executionError) || executionError.Code != "UNDECLARED_COMMAND_EFFECT" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvokerAcceptsDeclaredEffect(t *testing.T) {
	world := commandWorld()
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   effectExecutor{effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}},
		UnitOfWork: &MemoryUnitOfWork{},
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
}
