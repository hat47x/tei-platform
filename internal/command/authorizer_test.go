package command

import (
	"context"
	"errors"
	"testing"
)

type denyAuthorizer struct{}

func (denyAuthorizer) Authorize(context.Context, Request) error {
	return errors.New("caller is not permitted")
}

type allowAuthorizer struct{}

func (allowAuthorizer) Authorize(context.Context, Request) error { return nil }

func TestInvokerAuthorizerDeniesBeforeExecution(t *testing.T) {
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   ReferenceExecutor{World: commandWorld()},
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: denyAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if err == nil {
		t.Fatal("denied command was executed")
	}
	var executionError *ExecutionError
	if !errors.As(err, &executionError) || executionError.Code != "COMMAND_UNAUTHORIZED" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvokerAuthorizerAllowsExecution(t *testing.T) {
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   ReferenceExecutor{World: commandWorld()},
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: allowAuthorizer{},
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
}
