package workspace_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceLegacyAuthenticationBindingConnectsOrderEntryWithoutPersistingSecret(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	var appFound bool
	for _, app := range loaded.World.Applications {
		if app.ID != "order_entry" {
			continue
		}
		appFound = true
		if !containsSemanticUse(app.SemanticUses, "security") {
			t.Fatalf("order_entryがsecurity Semanticを利用していません: %#v", app.SemanticUses)
		}
		binding, err := realization.ResolveCustomGoBinding(loaded.World.Bindings, app.ID, legacyauth.AuthenticationCapability)
		if err != nil {
			t.Fatalf("security.authenticateのBindingを解決できませんでした: %v", err)
		}
		if got, want := binding.Identity(), "order_entry/security.authenticate"; got != want {
			t.Fatalf("binding identity=%q want=%q", got, want)
		}
		if got, want := binding.ExpectedAdapterFilename(), "security__authenticate.bind.go"; got != want {
			t.Fatalf("adapter filename=%q want=%q", got, want)
		}
		if got, want := binding.ExpectedHookName(), "TeiBindSecurityAuthenticate"; got != want {
			t.Fatalf("hook name=%q want=%q", got, want)
		}
	}
	if !appFound {
		t.Fatal("order_entry Applicationが見つかりません")
	}

	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	subject := sourceindex.Subject{Kind: "binding", ID: "order_entry/security.authenticate"}
	refs := idx.ReferencesFor(subject)
	if len(refs) != 1 {
		t.Fatalf("Authentication Binding source references=%#v", refs)
	}
	const bindingPath = "applications/order-entry/realizations/authenticate.tei"
	if refs[0].ArtifactPath != bindingPath {
		t.Fatalf("Binding source path=%q want=%q", refs[0].ArtifactPath, bindingPath)
	}
	content, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(bindingPath)))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(content[refs[0].Range.Start.Offset:refs[0].Range.End.Offset]); got != "security.authenticate" {
		t.Fatalf("Binding source anchor=%q", got)
	}

	const runtimeSecret = "runtime-only-credential-value"
	adapter := legacyauth.Adapter{
		Store: referenceLegacyUserStore{},
		Verifier: referenceCredentialVerifier{
			wantCredential: runtimeSecret,
		},
	}
	result, err := adapter.Invoke(context.Background(), legacyauth.AuthenticationCapability, legacyauth.Input{
		LoginID:    "legacy-user-001",
		Credential: runtimeSecret,
	})
	if err != nil {
		t.Fatalf("Reference legacy authenticationを実行できませんでした: %v", err)
	}
	if result.Outcome != "authenticated" {
		t.Fatalf("authentication result=%#v", result)
	}

	worldJSON, err := json.Marshal(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(worldJSON), runtimeSecret) || strings.Contains(string(resultJSON), runtimeSecret) || strings.Contains(string(content), runtimeSecret) {
		t.Fatal("実行時credentialがCanonical World、Business Result、またはBinding sourceへ残っています")
	}
}

func containsSemanticUse(ids []core.CanonicalID, want core.CanonicalID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

type referenceLegacyUserStore struct{}

func (referenceLegacyUserStore) FindByLogin(_ context.Context, login string) (legacyauth.User, bool, error) {
	if login != "legacy-user-001" {
		return legacyauth.User{}, false, nil
	}
	return legacyauth.User{
		LoginID:          login,
		SubjectID:        "subject:legacy-user-001",
		CredentialRef:    "legacy-credential:legacy-user-001",
		OrganizationCode: "SALES",
		RoleCode:         "ORDER_OPERATOR",
	}, true, nil
}

type referenceCredentialVerifier struct {
	wantCredential string
}

func (v referenceCredentialVerifier) Verify(_ context.Context, ref, credential string) (bool, error) {
	return ref == "legacy-credential:legacy-user-001" && credential == v.wantCredential, nil
}
