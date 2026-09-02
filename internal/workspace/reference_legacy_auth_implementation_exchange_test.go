package workspace_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

type legacyExchangeFunction func(context.Context, string, string) (legacyauth.FunctionResult, error)

func (f legacyExchangeFunction) Authenticate(ctx context.Context, login, credential string) (legacyauth.FunctionResult, error) {
	return f(ctx, login, credential)
}

func TestReferenceLegacyAuthenticationExchangesImplementationWithoutChangingApplicationBinding(t *testing.T) {
	root := legacyAuthRepoRoot(t)
	workspacePath := filepath.Join(root, "reference", "workspace")
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}

	app := legacyAuthApplicationByID(t, loaded.World.Applications, "order_entry")
	binding, err := realization.ResolveCustomGoBinding(loaded.World.Bindings, app.ID, legacyauth.AuthenticationCapability)
	if err != nil {
		t.Fatalf("security.authenticateのBindingを解決できません: %v", err)
	}
	if binding.Identity() != "order_entry/security.authenticate" {
		t.Fatalf("binding identity=%q", binding.Identity())
	}
	if binding.ExpectedHookName() != "TeiBindSecurityAuthenticate" {
		t.Fatalf("binding hook=%q", binding.ExpectedHookName())
	}

	capability := legacyAuthCapabilityByID(t, loaded.World.Semantics, legacyauth.AuthenticationCapability)
	input := legacyauth.Input{LoginID: "legacy-order-editor", Credential: "runtime-only-credential"}

	fixturePath := filepath.Join(root, "reference", "legacy-auth", "users.json")
	store, err := legacyauth.LoadFixtureStore(fixturePath)
	if err != nil {
		t.Fatalf("legacy user fixtureを読み込めません: %v", err)
	}
	directAdapter := legacyauth.Adapter{
		Store: store,
		Verifier: legacyFixtureVerifier(func(_ context.Context, ref, presented string) (bool, error) {
			if ref != "credential-ref://legacy/order-editor" {
				t.Fatalf("credential reference=%q", ref)
			}
			if presented != input.Credential {
				t.Fatalf("presented credential=%q", presented)
			}
			return true, nil
		}),
		Now: func() time.Time {
			return time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
		},
	}
	directResult, err := capabilityinvoke.Invoke(context.Background(), capability, directAdapter, input)
	if err != nil {
		t.Fatalf("既存user recordを読む実現方式を実行できません: %v", err)
	}

	mappingPath := filepath.Join(root, "reference", "legacy-auth", "authenticate-function.json")
	mapping, err := legacyauth.LoadFunctionMapping(mappingPath)
	if err != nil {
		t.Fatalf("legacy authentication function mappingを読み込めません: %v", err)
	}
	if mapping.Source != "legacy_security.authenticate_user" {
		t.Fatalf("legacy function source=%q", mapping.Source)
	}
	if mapping.SourcePath() != mappingPath {
		t.Fatalf("legacy function mapping source path=%q want=%q", mapping.SourcePath(), mappingPath)
	}

	functionCalls := 0
	functionAdapter := legacyauth.FunctionAdapter{
		Mapping: mapping,
		Function: legacyExchangeFunction(func(_ context.Context, login, credential string) (legacyauth.FunctionResult, error) {
			functionCalls++
			if login != input.LoginID {
				t.Fatalf("legacy function login=%q", login)
			}
			if credential != input.Credential {
				t.Fatalf("legacy function credential=%q", credential)
			}
			return legacyauth.FunctionResult{ReturnCode: "0", SubjectID: "subject-order-editor"}, nil
		}),
	}
	functionResult, err := capabilityinvoke.Invoke(context.Background(), capability, functionAdapter, input)
	if err != nil {
		t.Fatalf("既存認証関数を呼ぶ実現方式を実行できません: %v", err)
	}
	if functionCalls != 1 {
		t.Fatalf("legacy authentication function calls=%d want=1", functionCalls)
	}

	if directResult.Capability != functionResult.Capability ||
		directResult.Mode != functionResult.Mode ||
		directResult.Outcome != functionResult.Outcome {
		t.Fatalf("実現方式の交換でBusiness Result契約が変わっています: direct=%#v function=%#v", directResult, functionResult)
	}
	directSubject, ok := directResult.Payload.(legacyauth.AuthenticatedSubject)
	if !ok {
		t.Fatalf("direct payload=%#v", directResult.Payload)
	}
	functionSubject, ok := functionResult.Payload.(legacyauth.AuthenticatedSubject)
	if !ok {
		t.Fatalf("function payload=%#v", functionResult.Payload)
	}
	if directSubject.SubjectID != functionSubject.SubjectID || directSubject.SubjectID != "subject-order-editor" {
		t.Fatalf("実現方式の交換でsubject identityが変わっています: direct=%#v function=%#v", directSubject, functionSubject)
	}

	// Application側のBindingは一つのままである。
	// 実装の違いは同じTeiBindSecurityAuthenticate hookの背後へ閉じ込めるため、
	// Driverはlegacy user schemaやlegacy return codeを知る必要がない。
	bindingAfterExchange, err := realization.ResolveCustomGoBinding(loaded.World.Bindings, app.ID, legacyauth.AuthenticationCapability)
	if err != nil {
		t.Fatalf("実現方式交換後のBindingを解決できません: %v", err)
	}
	if bindingAfterExchange.Identity() != binding.Identity() || bindingAfterExchange.ExpectedHookName() != binding.ExpectedHookName() {
		t.Fatalf("実現方式交換でApplication Bindingが変わっています: before=%#v after=%#v", binding, bindingAfterExchange)
	}
}
