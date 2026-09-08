package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestBusinessWorkspaceSourceRootIsPhysicalDiscoveryWithoutIRWorldEmbedding(t *testing.T) {
	root := t.TempDir()
	writeBusinessWorkspaceDiscoveryFileProbe(t, filepath.Join(root, "tei.workspace.yaml"), []byte("workspace: business_discovery_pressure\nsources:\n  - business\n"))

	candidatePath := filepath.Join(root, "business", "order-entry-procedure.json")
	writeBusinessWorkspaceDiscoveryFileProbe(t, candidatePath, []byte(`{"schemaVersion":"business-authoring-candidate-probe-v0","procedure":"order-entry.order-handling"}`))

	legacyPath := filepath.Join(root, "reference", "business-system-design", "legacy-order-entry-procedure.json")
	writeBusinessWorkspaceDiscoveryFileProbe(t, legacyPath, []byte(`{"artifactRole":"legacy-reference-evidence"}`))

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("workspace load: %v", err)
	}

	// An explicit future-compatible source root is a physical discovery boundary,
	// not an instruction to embed unknown Business meaning into ir.World.
	if len(loaded.World.Applications) != 0 || len(loaded.World.Databases) != 0 {
		t.Fatalf("unknown Business source root unexpectedly changed ir.World application/database meaning: %+v", loaded.World)
	}

	candidateRel := filepath.ToSlash(filepath.Join("business", "order-entry-procedure.json"))
	legacyRel := filepath.ToSlash(filepath.Join("reference", "business-system-design", "legacy-order-entry-procedure.json"))
	managed := map[string]workspace.ManagedFile{}
	for _, item := range loaded.ManagedFiles {
		managed[item.Path] = item
	}
	candidate, ok := managed[candidateRel]
	if !ok {
		t.Fatalf("declared Business source root artifact was not discovered: managed=%+v", loaded.ManagedFiles)
	}
	if candidate.SourceRoot != "business" {
		t.Fatalf("Business artifact source root=%q want=business", candidate.SourceRoot)
	}
	if _, ok := managed[legacyRel]; ok {
		t.Fatalf("legacy Reference Evidence outside declared source roots became managed authority candidate: %+v", managed[legacyRel])
	}

	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	artifact, ok := businessWorkspaceDiscoveryArtifactProbe(index.Artifacts, candidateRel)
	if !ok {
		t.Fatalf("declared Business managed artifact is missing from Source Index: %+v", index.Artifacts)
	}
	if artifact.Fingerprint == "" {
		t.Fatal("Business managed artifact fingerprint is empty")
	}
	if _, ok := businessWorkspaceDiscoveryArtifactProbe(index.Artifacts, legacyRel); ok {
		t.Fatal("legacy Reference Evidence outside source roots entered Source Index managed artifacts")
	}
}

func TestBusinessWorkspaceSourceMoveChangesPhysicalLocatorWithoutChangingWorldProbe(t *testing.T) {
	root := t.TempDir()
	writeBusinessWorkspaceDiscoveryFileProbe(t, filepath.Join(root, "tei.workspace.yaml"), []byte("workspace: business_discovery_move_pressure\nsources:\n  - business\n"))

	oldRel := filepath.ToSlash(filepath.Join("business", "procedures", "order-entry.json"))
	newRel := filepath.ToSlash(filepath.Join("business", "procedures", "archive", "order-entry.json"))
	content := []byte(`{"schemaVersion":"business-authoring-candidate-probe-v0","procedure":"order-entry.order-handling"}`)
	oldPath := filepath.Join(root, filepath.FromSlash(oldRel))
	newPath := filepath.Join(root, filepath.FromSlash(newRel))
	writeBusinessWorkspaceDiscoveryFileProbe(t, oldPath, content)

	before, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load before move: %v", err)
	}
	beforeIndex, err := workspace.BuildSourceIndex(root, before)
	if err != nil {
		t.Fatalf("index before move: %v", err)
	}
	beforeArtifact, ok := businessWorkspaceDiscoveryArtifactProbe(beforeIndex.Artifacts, oldRel)
	if !ok {
		t.Fatalf("missing Business artifact before move: %+v", beforeIndex.Artifacts)
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	after, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load after move: %v", err)
	}
	afterIndex, err := workspace.BuildSourceIndex(root, after)
	if err != nil {
		t.Fatalf("index after move: %v", err)
	}
	afterArtifact, ok := businessWorkspaceDiscoveryArtifactProbe(afterIndex.Artifacts, newRel)
	if !ok {
		t.Fatalf("missing Business artifact after move: %+v", afterIndex.Artifacts)
	}

	if !reflect.DeepEqual(before.World, after.World) {
		t.Fatalf("physical Business artifact move changed ir.World\nbefore=%#v\nafter=%#v", before.World, after.World)
	}
	if beforeArtifact.Fingerprint != afterArtifact.Fingerprint {
		t.Fatalf("same Business source bytes changed fingerprint after move: before=%s after=%s", beforeArtifact.Fingerprint, afterArtifact.Fingerprint)
	}
	if beforeArtifact.Path == afterArtifact.Path {
		t.Fatalf("physical locator did not move: before=%q after=%q", beforeArtifact.Path, afterArtifact.Path)
	}
}

func writeBusinessWorkspaceDiscoveryFileProbe(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func businessWorkspaceDiscoveryArtifactProbe(values []sourceindex.Artifact, path string) (sourceindex.Artifact, bool) {
	for _, value := range values {
		if value.Path == path {
			return value, true
		}
	}
	return sourceindex.Artifact{}, false
}
