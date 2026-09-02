package workspace_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type legacyFixtureVerifier func(context.Context, string, string) (bool, error)

func (f legacyFixtureVerifier) Verify(ctx context.Context, ref, presented string) (bool, error) {
	return f(ctx, ref, presented)
}

func TestReferenceLegacyAuthenticationBindingKeepsLegacyMeaningAndProvenance(t *testing.T) {
	root := legacyAuthRepoRoot(t)
	workspacePath := filepath.Join(root, "reference", "workspace")
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}

	app := legacyAuthApplicationByID(t, loaded.World.Applications, "order_entry")
	if !containsCanonicalID(app.SemanticUses, "security") {
		t.Fatalf("order_entryがsecurity意味空間を利用していません: %#v", app.SemanticUses)
	}

	binding, err := realization.ResolveCustomGoBinding(loaded.World.Bindings, app.ID, "security.authenticate")
	if err != nil {
		t.Fatalf("security.authenticateのBindingを解決できません: %v", err)
	}
	if binding.Identity() != "order_entry/security.authenticate" {
		t.Fatalf("binding identity=%q", binding.Identity())
	}
	if binding.ExpectedAdapterFilename() != "security__authenticate.bind.go" {
		t.Fatalf("adapter filename=%q", binding.ExpectedAdapterFilename())
	}
	if binding.ExpectedHookName() != "TeiBindSecurityAuthenticate" {
		t.Fatalf("hook name=%q", binding.ExpectedHookName())
	}

	index, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatalf("Source Indexを構築できません: %v", err)
	}
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "binding", ID: binding.Identity()})
	if len(refs) != 1 {
		t.Fatalf("Authentication Bindingのsource references=%#v", refs)
	}
	const bindingPath = "applications/order-entry/realizations/authenticate.tei"
	if refs[0].ArtifactPath != bindingPath {
		t.Fatalf("Binding source path=%q want=%q", refs[0].ArtifactPath, bindingPath)
	}
	bindingSource, err := os.ReadFile(filepath.Join(workspacePath, filepath.FromSlash(bindingPath)))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(bindingSource[refs[0].Range.Start.Offset:refs[0].Range.End.Offset]); got != "security.authenticate" {
		t.Fatalf("Binding source anchor=%q", got)
	}

	fixturePath := filepath.Join(root, "reference", "legacy-auth", "users.json")
	store, err := legacyauth.LoadFixtureStore(fixturePath)
	if err != nil {
		t.Fatalf("legacy認証fixtureを読み込めません: %v", err)
	}
	if store.SourcePath() != fixturePath {
		t.Fatalf("fixture source path=%q want=%q", store.SourcePath(), fixturePath)
	}

	user, found, err := store.FindByLogin(context.Background(), "legacy-order-editor")
	if err != nil || !found {
		t.Fatalf("legacy userを取得できません: found=%v err=%v", found, err)
	}
	if user.LoginID != "legacy-order-editor" ||
		user.SubjectID != "subject-order-editor" ||
		user.CredentialRef != "credential-ref://legacy/order-editor" ||
		user.LoginFailureCount != 1 ||
		user.OrganizationCode != "SALES" ||
		user.RoleCode != "ORDER_EDITOR" {
		t.Fatalf("legacy fieldの意味が失われています: %#v", user)
	}

	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	fixtureText := string(fixtureBytes)
	for _, forbidden := range []string{`"password"`, `"password_hash"`, `"credential_hash"`, `"token"`, `"secret"`} {
		if strings.Contains(fixtureText, forbidden) {
			t.Fatalf("Git管理するlegacy fixtureにsecret material用fieldが含まれています: %s", forbidden)
		}
	}

	capability := legacyAuthCapabilityByID(t, loaded.World.Semantics, "security.authenticate")
	adapter := legacyauth.Adapter{
		Store: store,
		Verifier: legacyFixtureVerifier(func(_ context.Context, ref, presented string) (bool, error) {
			if ref != "credential-ref://legacy/order-editor" {
				t.Fatalf("credential reference=%q", ref)
			}
			if presented != "runtime-only-credential" {
				t.Fatalf("presented credential=%q", presented)
			}
			return true, nil
		}),
		Now: func() time.Time {
			return time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
		},
	}
	result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, legacyauth.Input{
		LoginID:    "legacy-order-editor",
		Credential: "runtime-only-credential",
	})
	if err != nil {
		t.Fatalf("legacy fixtureからsecurity.authenticateを実行できません: %v", err)
	}
	if result.Mode != capabilityinvoke.ResultOutcome || result.Outcome != "authenticated" {
		t.Fatalf("authentication result=%#v", result)
	}
	if strings.Contains(fmt.Sprintf("%#v", result), "runtime-only-credential") {
		t.Fatalf("Business Resultへ実行時credentialが混入しています: %#v", result)
	}
}

func legacyAuthRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func legacyAuthApplicationByID(t *testing.T, applications []application.Application, id core.CanonicalID) application.Application {
	t.Helper()
	for _, app := range applications {
		if app.ID == id {
			return app
		}
	}
	t.Fatalf("Application %sがReference Workspaceに見つかりません", id)
	return application.Application{}
}

func legacyAuthCapabilityByID(t *testing.T, namespaces []semantic.Namespace, id core.CanonicalID) semantic.Capability {
	t.Helper()
	for _, namespace := range namespaces {
		for _, capability := range namespace.Capabilities {
			if capability.ID == id {
				return capability
			}
		}
	}
	t.Fatalf("Semantic Capability %sがReference Workspaceに見つかりません", id)
	return semantic.Capability{}
}

func containsCanonicalID(values []core.CanonicalID, wanted core.CanonicalID) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
