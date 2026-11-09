package hook_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
)

type recordingCaller struct {
	called      []core.CanonicalID
	returnValue any
	returnError error
}

func (c *recordingCaller) Call(_ context.Context, capability core.CanonicalID, _ map[string]string) (any, error) {
	c.called = append(c.called, capability)
	return c.returnValue, c.returnError
}

func declaredHook(uses ...core.CanonicalID) *application.Hook {
	return &application.Hook{Phase: "before-read", Uses: uses}
}

// TestDeclaredHookCallsTheDeclaredCapability checks that a declared capability
// reaches the caller.
func TestDeclaredHookCallsTheDeclaredCapability(t *testing.T) {
	inner := &recordingCaller{returnValue: "ok"}
	caller := hook.Declared{Inner: inner, Hook: declaredHook("sales.lookup")}
	value, err := caller.Call(context.Background(), "sales.lookup", nil)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if value != "ok" || len(inner.called) != 1 {
		t.Fatalf("value=%v called=%v", value, inner.called)
	}
}

// TestDeclaredHookRefusesAnUndeclaredUsecapability checks that the declaration,
// not the implementation, decides what a hook may reach.
func TestDeclaredHookRefusesAnUndeclaredUse(t *testing.T) {
	inner := &recordingCaller{returnValue: "ok"}
	caller := hook.Declared{Inner: inner, Hook: declaredHook("sales.lookup")}
	_, err := caller.Call(context.Background(), "sales.delete", nil)
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_UNDECLARED_CALL" {
		t.Fatalf("err=%v", err)
	}
	if len(inner.called) != 0 {
		t.Fatalf("undeclared capability reached the caller: %v", inner.called)
	}
	if !strings.Contains(err.Error(), "sales.delete") {
		t.Fatalf("refusal does not name the capability: %v", err)
	}
}

// TestDeclaredHookRefusesWithoutADeclaration checks that a hook without a
// declaration calls nothing: silence is not a permission.
func TestDeclaredHookRefusesWithoutADeclaration(t *testing.T) {
	inner := &recordingCaller{}
	caller := hook.Declared{Inner: inner}
	if _, err := caller.Call(context.Background(), "sales.lookup", nil); err == nil {
		t.Fatal("a hook without a declaration was allowed to call")
	}
	if len(inner.called) != 0 {
		t.Fatalf("called=%v", inner.called)
	}
}

// TestDeclaredHookReportsMissingCaller checks that a declared capability with no
// caller is reported as a boundary failure, not as a denial.
func TestDeclaredHookReportsMissingCaller(t *testing.T) {
	caller := hook.Declared{Hook: declaredHook("sales.lookup")}
	_, err := caller.Call(context.Background(), "sales.lookup", nil)
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_IMPLEMENTATION_UNAVAILABLE" {
		t.Fatalf("err=%v", err)
	}
}

// TestDeclaredHookPassesTheCallerError checks that the caller's own error is
// returned unchanged, so a hook failure keeps its cause.
func TestDeclaredHookPassesTheCallerError(t *testing.T) {
	inner := &recordingCaller{returnError: errors.New("provider failed")}
	caller := hook.Declared{Inner: inner, Hook: declaredHook("sales.lookup")}
	_, err := caller.Call(context.Background(), "sales.lookup", nil)
	if err == nil || !strings.Contains(err.Error(), "provider failed") {
		t.Fatalf("err=%v", err)
	}
}
