package command

import (
	"context"
	"errors"
	"testing"

	"tei/internal/semantic"
)

type externalAuthorizer struct{}

func (externalAuthorizer) Authorize(context.Context, Request) error { return nil }

// TestInvokerRecordsReferenceAllowAll checks that the reference authorizer is
// named as such, so a reader does not mistake it for a real grant.
func TestInvokerRecordsReferenceAllowAll(t *testing.T) {
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}},
		UnitOfWork: &MemoryUnitOfWork{},
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.AuthorizationMode != AuthorizationReferenceAllowAll {
		t.Fatalf("authorizationMode=%q", result.AuthorizationMode)
	}
}

// TestInvokerRecordsAllowList checks that the capability allow list is named
// separately from allow-all.
func TestInvokerRecordsAllowList(t *testing.T) {
	invoker := Invoker{
		Authorizer: AllowListAuthorizer{"sales.submit"},
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}},
		UnitOfWork: &MemoryUnitOfWork{},
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.AuthorizationMode != AuthorizationAllowList {
		t.Fatalf("authorizationMode=%q", result.AuthorizationMode)
	}
}

// TestInvokerRecordsExternalAuthorizer checks that any other Authorizer is
// called external without claiming which rules it consulted.
func TestInvokerRecordsExternalAuthorizer(t *testing.T) {
	invoker := Invoker{
		Authorizer: externalAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}},
		UnitOfWork: &MemoryUnitOfWork{},
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.AuthorizationMode != AuthorizationExternal {
		t.Fatalf("authorizationMode=%q", result.AuthorizationMode)
	}
}

// TestInvokerDoesNotRecordAModeWhenAuthorizationIsMissing checks the control:
// an unconfigured Authorizer still refuses, so no mode is fabricated.
func TestInvokerDoesNotRecordAModeWhenAuthorizationIsMissing(t *testing.T) {
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{}},
		UnitOfWork: &MemoryUnitOfWork{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	var execution *ExecutionError
	if !errors.As(err, &execution) || execution.Code != "COMMAND_AUTHORIZER_UNAVAILABLE" {
		t.Fatalf("err=%v", err)
	}
}
