package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/sourceindex"
)

func TestBuildSourceIndexInspectionReturnsWholeIndex(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildSourceIndexInspection(filepath.Join(root, "reference", "workspace"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	index, ok := got.(sourceindex.Index)
	if !ok {
		t.Fatalf("inspection type=%T", got)
	}
	if len(index.Artifacts) == 0 || len(index.References) == 0 {
		t.Fatalf("empty source index=%#v", index)
	}
}

func TestBuildSourceIndexInspectionFiltersQuerySubjectAndKeepsFingerprints(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildSourceIndexInspection(
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
	if _, err := buildSourceIndexInspection("unused", "query", ""); err == nil {
		t.Fatal("subject kind without subject id was accepted")
	}
	if _, err := buildSourceIndexInspection("unused", "", "order_entry.recent_orders"); err == nil {
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
