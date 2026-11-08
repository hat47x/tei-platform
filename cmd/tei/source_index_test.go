package main

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/sourceindex"
)

func TestBuildSourceIndexInspectionReturnsWholeIndex(t *testing.T) {
	root := testRepositoryRoot(t)
	got, _, err := buildSourceIndexInspection(filepath.Join(root, "reference", "workspace"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	index, ok := got.(sourceIndexIndexView)
	if !ok {
		t.Fatalf("inspection type=%T", got)
	}
	if len(index.Artifacts) == 0 || len(index.References) == 0 {
		t.Fatalf("empty source index=%#v", index)
	}
}

// An index with no references must still answer an empty JSON array. A nil
// slice serializes as null, and a caller that iterates result.references then
// fails only on the Workspace that has no semantic declarations (#3456).
func TestSourceIndexWritesEmptyListsAsArrays(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs-only")
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: design_docs\nformatVersion: 1\nsources:\n  - docs\n")
	writeWorkspaceFile(t, root, "docs/architecture.md", "# Architecture\n")

	raw, mode, err := buildSourceIndexInspection(root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if mode != resultModeIndex {
		t.Fatalf("mode=%q", mode)
	}
	view, ok := raw.(sourceIndexIndexView)
	if !ok {
		t.Fatalf("inspection type=%T", raw)
	}
	if len(view.Artifacts) == 0 {
		t.Fatalf("the docs source produced no artifacts: %#v", view)
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Artifacts  []any `json:"artifacts"`
		References []any `json:"references"`
	}
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	if body.Artifacts == nil || body.References == nil {
		t.Fatalf("an empty list was written as null: %s", encoded)
	}
}

// The subject mode must write [] too when the subject has no references.
func TestSourceIndexSubjectViewWritesEmptyReferencesAsArray(t *testing.T) {
	raw, mode, err := buildSourceIndexInspection(
		filepath.Join(testRepositoryRoot(t), "reference", "workspace"),
		"not-a-kind", "nothing")
	if err != nil {
		t.Fatal(err)
	}
	if mode != resultModeSubject {
		t.Fatalf("mode=%q", mode)
	}
	view, ok := raw.(sourceIndexSubjectView)
	if !ok {
		t.Fatalf("inspection type=%T", raw)
	}
	if len(view.References) != 0 {
		t.Fatalf("subject unexpectedly has references: %#v", view.References)
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Artifacts  []any `json:"artifacts"`
		References []any `json:"references"`
	}
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	if body.References == nil || body.Artifacts == nil {
		t.Fatalf("an empty list was written as null: %s", encoded)
	}
}

func TestBuildSourceIndexInspectionFiltersQuerySubjectAndKeepsFingerprints(t *testing.T) {
	root := testRepositoryRoot(t)
	got, _, err := buildSourceIndexInspection(
		filepath.Join(root, "reference", "workspace"),
		"query",
		"order_entry.recent_orders",
	)
	if err != nil {
		t.Fatal(err)
	}
	view, ok := got.(sourceIndexSubjectView)
	if !ok {
		t.Fatalf("inspection type=%T", got)
	}
	if view.Subject != (sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"}) {
		t.Fatalf("subject=%#v", view.Subject)
	}
	if len(view.References) != 2 {
		t.Fatalf("references=%#v", view.References)
	}
	roles := map[sourceindex.Role]bool{}
	for _, ref := range view.References {
		roles[ref.Role] = true
	}
	if !roles[sourceindex.RoleDefinition] || !roles[sourceindex.RoleAuthoring] {
		t.Fatalf("roles=%#v", roles)
	}
	if len(view.Artifacts) != 2 {
		t.Fatalf("artifacts=%#v", view.Artifacts)
	}
	for _, artifact := range view.Artifacts {
		if artifact.Fingerprint == "" {
			t.Fatalf("artifact lacks fingerprint: %#v", artifact)
		}
	}
}

func TestBuildSourceIndexInspectionRequiresSubjectPair(t *testing.T) {
	if _, _, err := buildSourceIndexInspection("unused", "query", ""); err == nil {
		t.Fatal("subject kind without subject id was accepted")
	}
	if _, _, err := buildSourceIndexInspection("unused", "", "order_entry.recent_orders"); err == nil {
		t.Fatal("subject id without subject kind was accepted")
	}
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
