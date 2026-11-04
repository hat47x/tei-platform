package designoutput

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func TestEditableSourcesIncludeOnlyDefinitionAndAuthoring(t *testing.T) {
	index := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{
			{Path: "semantic/order.tei", Fingerprint: "sha256:aaa"},
			{Path: "applications/app.tei", Fingerprint: "sha256:bbb"},
			{Path: "generated/main.go", Fingerprint: "sha256:ccc"},
		},
		References: []sourceindex.Reference{
			{ArtifactPath: "semantic/order.tei", Role: sourceindex.RoleDefinition},
			{ArtifactPath: "semantic/order.tei", Role: sourceindex.RoleDefinition},
			{ArtifactPath: "applications/app.tei", Role: sourceindex.RoleAuthoring},
			{ArtifactPath: "generated/main.go", Role: sourceindex.RoleImplementation},
			{ArtifactPath: "derived/report.md", Role: sourceindex.RoleDerived},
		},
	}
	got := editableSources(index)
	if len(got) != 2 {
		t.Fatalf("editableSources=%+v", got)
	}
	if got[0].Path != "applications/app.tei" || got[0].Fingerprint != "sha256:bbb" || got[0].Role != "authoring" {
		t.Fatalf("source[0]=%+v", got[0])
	}
	if got[1].Path != "semantic/order.tei" || got[1].Fingerprint != "sha256:aaa" || got[1].Role != "definition" {
		t.Fatalf("source[1]=%+v", got[1])
	}
}

func TestBundleShowsEditableSourcesAndReadOnlyArtifacts(t *testing.T) {
	root := referenceWorkspace(t)
	out := filepath.Join(t.TempDir(), "bundle")
	manifest, err := Write(root, out, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.EditableSources) == 0 {
		t.Fatal("no canonical source was exposed for editing")
	}
	artifacts := map[string]bool{}
	for _, artifact := range manifest.Artifacts {
		artifacts[artifact.Path] = true
	}
	for _, source := range manifest.EditableSources {
		if strings.HasPrefix(source.Path, "/") || strings.Contains(source.Path, "..") {
			t.Errorf("editable source %q is not workspace-relative", source.Path)
		}
		if artifacts[source.Path] {
			t.Errorf("derived artifact %q was exposed as an editable canonical source", source.Path)
		}
	}
	joined := strings.Join(manifest.Notes, "\n")
	if !strings.Contains(joined, "読み取り専用") || !strings.Contains(joined, "editableSources") {
		t.Fatalf("notes do not point at the editable canonical sources: %q", joined)
	}
}
