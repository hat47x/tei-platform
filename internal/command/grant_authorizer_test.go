package command

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/semantic"
)

func grantAuthorizerRequest() Request {
	return Request{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Capability:  "sales.submit",
		Inputs:      map[string]string{"order": "ORD-1"},
	}
}

func grantSpecs() []permission.Spec {
	return []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "sales/orders"}}
}

// TestGrantAuthorizerRequiresEveryDeclaredPermission checks that a partial
// grant is not enough: the missing permission is named in the denial.
func TestGrantAuthorizerRequiresEveryDeclaredPermission(t *testing.T) {
	required := []permission.Spec{
		{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "sales/orders"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "billing"},
	}
	authorizer := GrantAuthorizer{
		Declared: map[core.CanonicalID][]permission.Spec{"sales.submit": required},
		Grants:   grantSpecs(),
	}
	err := authorizer.Authorize(context.Background(), grantAuthorizerRequest())
	if !errors.Is(err, ErrPermissionNotGranted) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "network") {
		t.Fatalf("denial does not name the missing permission: %v", err)
	}
}

// TestGrantAuthorizerAcceptsCoveringGrants checks that a covering grant is
// enough and that a read-write grant covers a write.
func TestGrantAuthorizerAcceptsCoveringGrants(t *testing.T) {
	authorizer := GrantAuthorizer{
		Declared: map[core.CanonicalID][]permission.Spec{"sales.submit": grantSpecs()},
		Grants: []permission.Spec{
			{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "sales/orders"},
		},
	}
	if err := authorizer.Authorize(context.Background(), grantAuthorizerRequest()); err != nil {
		t.Fatalf("err=%v", err)
	}
}

// TestGrantAuthorizerRefusesUndeclaredCapability checks that silence is not a
// grant: a capability with no declaration is refused.
func TestGrantAuthorizerRefusesUndeclaredCapability(t *testing.T) {
	authorizer := GrantAuthorizer{
		Grants: []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "sales/orders"}},
	}
	err := authorizer.Authorize(context.Background(), grantAuthorizerRequest())
	if !errors.Is(err, ErrPermissionNotGranted) {
		t.Fatalf("err=%v", err)
	}
}

// TestGrantAuthorizerRefusesInvalidDeclaration checks that a declaration that
// cannot be understood is refused, not treated as satisfied.
func TestGrantAuthorizerRefusesInvalidDeclaration(t *testing.T) {
	authorizer := GrantAuthorizer{
		Declared: map[core.CanonicalID][]permission.Spec{"sales.submit": {{Kind: permission.KindFilesystem, Access: permission.AccessConnect, Scope: "sales/orders"}}},
		Grants:   []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessConnect, Scope: "sales/orders"}},
	}
	err := authorizer.Authorize(context.Background(), grantAuthorizerRequest())
	if err == nil || errors.Is(err, ErrPermissionNotGranted) {
		t.Fatalf("invalid declaration err=%v", err)
	}
}

// TestGrantAuthorizerRunsBeforeTheExecutor checks that a denied command never
// reaches the executor, so a menu or a Pattern cannot reach the data path.
func TestGrantAuthorizerRunsBeforeTheExecutor(t *testing.T) {
	executor := &grantRecordingExecutor{}
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   executor,
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: GrantAuthorizer{
			Declared: map[core.CanonicalID][]permission.Spec{"sales.submit": grantSpecs()},
		},
	}
	_, err := invoker.Invoke(context.Background(), grantAuthorizerRequest())
	var executionError *ExecutionError
	if !errors.As(err, &executionError) || executionError.Code != "COMMAND_UNAUTHORIZED" {
		t.Fatalf("err=%v", err)
	}
	if executor.called {
		t.Fatal("denied command reached the executor")
	}
}

type grantRecordingExecutor struct {
	called bool
}

func (e *grantRecordingExecutor) Execute(context.Context, Request) (Result, error) {
	e.called = true
	return Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}, nil
}
