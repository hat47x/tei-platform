package legacyauth_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

func TestAuthenticationImplementationCanBeExchangedWithoutChangingCanonicalIdentity(t *testing.T) {
	capability := referenceCapability(t, legacyauth.AuthenticationCapability)
	loaded := loadReferenceWorkspaceForAuthenticationExchange(t)

	binding, err := realization.ResolveCustomGoBinding(
		loaded.World.Bindings,
		"order_entry",
		legacyauth.AuthenticationCapability,
	)
	if err != nil {
		t.Fatalf("Reference Bindingを解決できません: %v", err)
	}
	if binding.Identity() != "order_entry/security.authenticate" {
		t.Fatalf("Binding identity=%q", binding.Identity())
	}

	recordAdapter := legacyauth.Adapter{
		Store: storeFunc(func(context.Context, string) (legacyauth.User, bool, error) {
			return legacyauth.User{
				LoginID:       "legacy-user-001",
				SubjectID:     "subject-001",
				CredentialRef: "credential-ref-001",
			}, true, nil
		}),
		Verifier: verifierFunc(func(context.Context, string, string) (bool, error) {
			return true, nil
		}),
	}
	functionAdapter := legacyauth.FunctionAdapter{
		Mapping: loadReferenceFunctionMapping(t),
		Function: authenticateFunctionFunc(func(context.Context, string, string) (legacyauth.FunctionResult, error) {
			return legacyauth.FunctionResult{ReturnCode: "0", SubjectID: "subject-001"}, nil
		}),
	}

	input := legacyauth.Input{LoginID: "legacy-user-001", Credential: "runtime-presented-value"}
	recordResult, err := capabilityinvoke.Invoke(context.Background(), capability, recordAdapter, input)
	if err != nil {
		t.Fatalf("既存record経路の認証に失敗しました: %v", err)
	}
	functionResult, err := capabilityinvoke.Invoke(context.Background(), capability, functionAdapter, input)
	if err != nil {
		t.Fatalf("既存function経路の認証に失敗しました: %v", err)
	}

	if recordResult.Capability != legacyauth.AuthenticationCapability || functionResult.Capability != legacyauth.AuthenticationCapability {
		t.Fatalf("実装交換でCapability identityが変わっています: record=%s function=%s", recordResult.Capability, functionResult.Capability)
	}
	if recordResult.Mode != capabilityinvoke.ResultOutcome || functionResult.Mode != capabilityinvoke.ResultOutcome ||
		recordResult.Outcome != "authenticated" || functionResult.Outcome != "authenticated" {
		t.Fatalf("実装交換でBusiness Result契約が変わっています: record=%#v function=%#v", recordResult, functionResult)
	}

	recordSubject, ok := recordResult.Payload.(legacyauth.AuthenticatedSubject)
	if !ok {
		t.Fatalf("既存record経路のpayload=%#v", recordResult.Payload)
	}
	functionSubject, ok := functionResult.Payload.(legacyauth.AuthenticatedSubject)
	if !ok {
		t.Fatalf("既存function経路のpayload=%#v", functionResult.Payload)
	}
	if recordSubject.SubjectID != "subject-001" || functionSubject.SubjectID != recordSubject.SubjectID {
		t.Fatalf("実装交換でauthenticated subjectが変わっています: record=%#v function=%#v", recordSubject, functionSubject)
	}

	deniedRecordAdapter := legacyauth.Adapter{
		Store: storeFunc(func(context.Context, string) (legacyauth.User, bool, error) {
			return legacyauth.User{
				LoginID:       "legacy-user-001",
				SubjectID:     "subject-001",
				CredentialRef: "credential-ref-001",
			}, true, nil
		}),
		Verifier: verifierFunc(func(context.Context, string, string) (bool, error) {
			return false, nil
		}),
	}
	deniedFunctionAdapter := legacyauth.FunctionAdapter{
		Mapping: loadReferenceFunctionMapping(t),
		Function: authenticateFunctionFunc(func(context.Context, string, string) (legacyauth.FunctionResult, error) {
			return legacyauth.FunctionResult{ReturnCode: "10"}, nil
		}),
	}

	for name, adapter := range map[string]capabilityinvoke.Adapter{
		"既存record経路":   deniedRecordAdapter,
		"既存function経路": deniedFunctionAdapter,
	} {
		t.Run(name+"の拒否", func(t *testing.T) {
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, input)
			if err != nil {
				t.Fatalf("期待された拒否がExecution Errorになっています: %v", err)
			}
			if result.Capability != legacyauth.AuthenticationCapability || result.Outcome != "denied" {
				t.Fatalf("denied result=%#v", result)
			}
			denial, ok := result.Payload.(legacyauth.AuthenticationDenial)
			if !ok || denial.Reason != "credential_mismatch" {
				t.Fatalf("denied payload=%#v", result.Payload)
			}
		})
	}
}

func loadReferenceWorkspaceForAuthenticationExchange(t *testing.T) workspace.Result {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}
	return loaded
}
