package workspace_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/realization"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceLegacyAuthorizationBindingHasStableIdentityAndSource(t *testing.T) {
	root := authorizationBindingRepoRoot(t)
	workspacePath := filepath.Join(root, "reference", "workspace")
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}

	binding, err := realization.ResolveCustomGoBinding(loaded.World.Bindings, "order_entry", "security.authorize")
	if err != nil {
		t.Fatalf("security.authorizeのBindingを解決できません: %v", err)
	}
	if binding.Identity() != "order_entry/security.authorize" {
		t.Fatalf("binding identity=%q", binding.Identity())
	}
	if binding.ExpectedAdapterFilename() != "security__authorize.bind.go" {
		t.Fatalf("adapter filename=%q", binding.ExpectedAdapterFilename())
	}
	if binding.ExpectedHookName() != "TeiBindSecurityAuthorize" {
		t.Fatalf("hook name=%q", binding.ExpectedHookName())
	}

	index, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatalf("Source Indexを構築できません: %v", err)
	}
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "binding", ID: binding.Identity()})
	if len(refs) != 1 {
		t.Fatalf("Authorization Bindingのsource references=%#v", refs)
	}
	const sourcePath = "applications/order-entry/realizations/authorize.tei"
	if refs[0].ArtifactPath != sourcePath {
		t.Fatalf("Binding source path=%q want=%q", refs[0].ArtifactPath, sourcePath)
	}
	contents, err := os.ReadFile(filepath.Join(workspacePath, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(contents[refs[0].Range.Start.Offset:refs[0].Range.End.Offset]); got != "security.authorize" {
		t.Fatalf("Binding source anchor=%q", got)
	}
}

func authorizationBindingRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
