package studiohost

import (
	"os"
	"path/filepath"
	"testing"
)

func rootReplacementPair(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	root := filepath.Join(tmp, "workspace-a")
	replacement := filepath.Join(tmp, "workspace-b")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(replacement, 0o700); err != nil {
		t.Fatal(err)
	}
	originalInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	replacementInfo, err := os.Stat(replacement)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(originalInfo, replacementInfo) {
		t.Fatal("replacement fixture must have a distinct directory identity")
	}
	return root, replacement
}

func replaceWorkspaceRoot(t *testing.T, root, replacement string) {
	t.Helper()
	moved := root + "-original"
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, root); err != nil {
		t.Fatal(err)
	}
	originalInfo, err := os.Stat(moved)
	if err != nil {
		t.Fatal(err)
	}
	replacementInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(originalInfo, replacementInfo) {
		t.Fatal("replacement path still identifies the original directory")
	}
}

func TestReadOnlyRejectsRootReplacement(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	h, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	replaceWorkspaceRoot(t, root, replacement)

	if got := h.ValidateWorkspace("workspace-1"); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("validation did not reject replacement root: %+v", got)
	}
	if got := h.CatalogWorkspace("workspace-1", ""); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("catalog did not reject replacement root: %+v", got)
	}
}

func TestFactoryIsNotCalledForReplacedWorkspaceRoot(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	factoryCalls := 0
	h, err := NewReadOnlyWithFactory([]string{root}, func(string) (RootServices, error) {
		factoryCalls++
		return RootServices{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	replaceWorkspaceRoot(t, root, replacement)

	if got := h.ValidateWorkspace("workspace-1"); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("factory validation did not reject replacement root: %+v", got)
	}
	if got := h.CatalogWorkspace("workspace-1", ""); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("factory catalog did not reject replacement root: %+v", got)
	}
	if factoryCalls != 0 {
		t.Fatalf("factory was called for replaced root %d times", factoryCalls)
	}
}

func TestReadOnlyRejectsUnauthorizedIDs(t *testing.T) {
	h, err := NewReadOnly([]string{filepath.Join("..", "..", "reference", "workspace")})
	if err != nil {
		t.Fatal(err)
	}
	list := h.ListWorkspaces()
	if !list.OK || len(list.Value) != 1 || list.Value[0].ID != "workspace-1" {
		t.Fatalf("unexpected allowlist: %+v", list)
	}
	list.Value[0].ID = "changed"
	if h.ListWorkspaces().Value[0].ID != "workspace-1" {
		t.Fatal("caller mutated the host's workspace list")
	}
	for _, id := range []string{"", "../reference/workspace", "workspace-2", " workspace-1"} {
		if got := h.ValidateWorkspace(id); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("validate %q: %+v", id, got)
		}
		if got := h.CatalogWorkspace(id, ""); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("catalog %q: %+v", id, got)
		}
		if got := h.InspectSubject(id, "semantic", "x"); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("inspect %q: %+v", id, got)
		}
	}
}

func TestRealOperationReadOnlyReference(t *testing.T) {
	h, err := NewReadOnly([]string{filepath.Join("..", "..", "reference", "workspace")})
	if err != nil {
		t.Fatal(err)
	}
	catalog := h.CatalogWorkspace("workspace-1", "")
	if !catalog.OK || catalog.Value == nil || catalog.Value.Effect != "read-only" {
		t.Fatalf("catalog: %+v", catalog)
	}
	validation := h.ValidateWorkspace("workspace-1")
	if !validation.OK || validation.Value == nil {
		t.Fatalf("validation: %+v", validation)
	}
	if len(catalog.Value.Entries) > 0 {
		subject := catalog.Value.Entries[0].Subject
		inspection := h.InspectSubject("workspace-1", subject.Kind, subject.ID)
		if !inspection.OK || inspection.Value == nil || inspection.Value.Effect != "read-only" {
			t.Fatalf("inspect %v: %+v", subject, inspection)
		}
	}
}

func TestRootConfigurationRefusesDuplicatesAndMissing(t *testing.T) {
	path := filepath.Join("..", "..", "reference", "workspace")
	if _, err := NewReadOnly(nil); err == nil {
		t.Fatal("empty root list accepted")
	}
	if _, err := NewReadOnly([]string{path, path}); err == nil {
		t.Fatal("duplicate root accepted")
	}
	if _, err := NewReadOnly([]string{filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("missing root accepted")
	}
}

func TestLoadFailureIsDistinguishedFromValidationDiagnostics(t *testing.T) {
	h, err := NewReadOnly([]string{t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	got := h.ValidateWorkspace("workspace-1")
	if got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_LOAD_FAILED" || got.Failure.Stage != "load" {
		t.Fatalf("unexpected failure: %+v", got)
	}
}
