package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/realization"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceSubmitCommandResolvesExplicitCapabilityBinding(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	app := result.World.Applications[0]
	pattern := applicationPatternByID(t, app.Patterns, "order_entry.submit_order")
	if pattern.Capability != "sales.order.submit" {
		t.Fatalf("command PatternのCapabilityが変わっています: %#v", pattern)
	}

	binding, err := realization.ResolveCustomGoBinding(result.World.Bindings, app.ID, pattern.Capability)
	if err != nil {
		t.Fatalf("sales.order.submitのBindingを解決できませんでした: %v", err)
	}
	if binding.Identity() != "order_entry/sales.order.submit" {
		t.Fatalf("binding identity=%q", binding.Identity())
	}
	if binding.ExpectedAdapterFilename() != "sales__order__submit.bind.go" {
		t.Fatalf("adapter filename=%q", binding.ExpectedAdapterFilename())
	}
	if binding.ExpectedHookName() != "TeiBindSalesOrderSubmit" {
		t.Fatalf("hook name=%q", binding.ExpectedHookName())
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "binding", ID: binding.Identity()})
	if len(refs) != 1 {
		t.Fatalf("Binding source references=%#v", refs)
	}
	const bindingPath = "applications/order-entry/realizations/submit-order.tei"
	if refs[0].ArtifactPath != bindingPath {
		t.Fatalf("Binding source path=%q want=%q", refs[0].ArtifactPath, bindingPath)
	}
	content, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(bindingPath)))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(content[refs[0].Range.Start.Offset:refs[0].Range.End.Offset]); got != "sales.order.submit" {
		t.Fatalf("Binding source anchor=%q", got)
	}
}

func TestBindingSourceIdentityIsApplicationScopedAndSurvivesFileRename(t *testing.T) {
	root := t.TempDir()
	writeBindingWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: binding_source_test\nsources:\n  - applications\n")
	writeBindingWorkspaceFile(t, root, "applications/first/application.tei", "application first_app {\n}\n")
	writeBindingWorkspaceFile(t, root, "applications/second/application.tei", "application second_app {\n}\n")
	writeBindingWorkspaceFile(t, root, "applications/first/realizations/submit.tei", "realize sales.order.submit with custom go\n")
	writeBindingWorkspaceFile(t, root, "applications/second/realizations/submit.tei", "realize sales.order.submit with custom go\n")

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Workspaceの読み込みに失敗しました: %v", err)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}

	firstSubject := sourceindex.Subject{Kind: "binding", ID: "first_app/sales.order.submit"}
	secondSubject := sourceindex.Subject{Kind: "binding", ID: "second_app/sales.order.submit"}
	firstRefs := idx.ReferencesFor(firstSubject)
	secondRefs := idx.ReferencesFor(secondSubject)
	if len(firstRefs) != 1 || len(secondRefs) != 1 {
		t.Fatalf("ApplicationごとのBinding sourceを分離できていません: first=%#v second=%#v", firstRefs, secondRefs)
	}
	if firstRefs[0].ArtifactPath == secondRefs[0].ArtifactPath {
		t.Fatalf("異なるApplicationのBinding sourceが同じartifactへ潰れています: %#v %#v", firstRefs[0], secondRefs[0])
	}

	oldPath := filepath.Join(root, "applications", "first", "realizations", "submit.tei")
	newPath := filepath.Join(root, "applications", "first", "realizations", "submit-order-renamed.tei")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	movedLoaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("rename後のWorkspaceを読み込めませんでした: %v", err)
	}
	movedIndex, err := workspace.BuildSourceIndex(root, movedLoaded)
	if err != nil {
		t.Fatalf("rename後のSource Indexを構築できませんでした: %v", err)
	}
	movedRefs := movedIndex.ReferencesFor(firstSubject)
	if len(movedRefs) != 1 {
		t.Fatalf("rename後のBinding source references=%#v", movedRefs)
	}
	if movedRefs[0].Subject != firstRefs[0].Subject {
		t.Fatalf("file renameでBinding identityが変わりました: before=%#v after=%#v", firstRefs[0].Subject, movedRefs[0].Subject)
	}
	if movedRefs[0].ArtifactPath == firstRefs[0].ArtifactPath {
		t.Fatalf("file rename後もartifact pathが更新されていません: %#v", movedRefs[0])
	}
}

func writeBindingWorkspaceFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
