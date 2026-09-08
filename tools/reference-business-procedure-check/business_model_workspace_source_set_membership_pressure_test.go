package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func TestBusinessWorkspaceSourceRootMembershipChangesManagedAuthorityReachability(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "tei.workspace.yaml")
	candidateRel := filepath.ToSlash(filepath.Join("business", "order-entry-procedure.json"))
	candidatePath := filepath.Join(root, filepath.FromSlash(candidateRel))

	writeBusinessWorkspaceDiscoveryFileProbe(t, candidatePath, []byte(`{"schemaVersion":"business-authoring-candidate-probe-v0","procedure":"order-entry.order-handling"}`))
	writeBusinessWorkspaceDiscoveryFileProbe(t, manifestPath, []byte("workspace: business_membership_pressure\nsources:\n  - business\n"))

	before, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load before source-root change: %v", err)
	}
	if !businessWorkspaceManagedPathProbe(before.ManagedFiles, candidateRel) {
		t.Fatalf("declared Business source was not managed before membership change: %+v", before.ManagedFiles)
	}
	beforeIndex, err := workspace.BuildSourceIndex(root, before)
	if err != nil {
		t.Fatalf("build source index before membership change: %v", err)
	}
	beforeArtifact, ok := businessWorkspaceDiscoveryArtifactProbe(beforeIndex.Artifacts, candidateRel)
	if !ok || beforeArtifact.Fingerprint == "" {
		t.Fatalf("declared Business source missing from Source Index before membership change: %+v", beforeIndex.Artifacts)
	}

	// Keep the candidate bytes untouched on disk and change only which source
	// root the workspace declares authoritative/reachable. Physical existence
	// must not keep the old file in ManagedFiles or the Source Index.
	writeBusinessWorkspaceDiscoveryFileProbe(t, manifestPath, []byte("workspace: business_membership_pressure\nsources:\n  - other\n"))

	after, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load after source-root change: %v", err)
	}
	if businessWorkspaceManagedPathProbe(after.ManagedFiles, candidateRel) {
		t.Fatalf("Business source remained managed after its source root left workspace membership: %+v", after.ManagedFiles)
	}
	afterIndex, err := workspace.BuildSourceIndex(root, after)
	if err != nil {
		t.Fatalf("build source index after membership change: %v", err)
	}
	if _, ok := businessWorkspaceDiscoveryArtifactProbe(afterIndex.Artifacts, candidateRel); ok {
		t.Fatalf("physically present but no-longer-declared Business source remained in Source Index: %+v", afterIndex.Artifacts)
	}
	if _, err := os.Stat(candidatePath); err != nil {
		t.Fatalf("Business source should remain physically present while authority reachability changes: %v", err)
	}
}

func TestBusinessWorkspaceSourceAddRemoveChangesManagedSourceSet(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "tei.workspace.yaml")
	writeBusinessWorkspaceDiscoveryFileProbe(t, manifestPath, []byte("workspace: business_add_remove_pressure\nsources:\n  - business\n"))

	primaryRel := filepath.ToSlash(filepath.Join("business", "order-entry-procedure.json"))
	secondaryRel := filepath.ToSlash(filepath.Join("business", "follow-up-procedure.json"))
	primaryPath := filepath.Join(root, filepath.FromSlash(primaryRel))
	secondaryPath := filepath.Join(root, filepath.FromSlash(secondaryRel))
	writeBusinessWorkspaceDiscoveryFileProbe(t, primaryPath, []byte(`{"schemaVersion":"business-authoring-candidate-probe-v0","procedure":"order-entry.order-handling"}`))

	initial, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("initial workspace load: %v", err)
	}
	if !businessWorkspaceManagedPathProbe(initial.ManagedFiles, primaryRel) || businessWorkspaceManagedPathProbe(initial.ManagedFiles, secondaryRel) {
		t.Fatalf("unexpected initial Business source set: %+v", initial.ManagedFiles)
	}

	writeBusinessWorkspaceDiscoveryFileProbe(t, secondaryPath, []byte(`{"schemaVersion":"business-authoring-candidate-probe-v0","procedure":"order-entry.follow-up"}`))
	added, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("workspace load after source add: %v", err)
	}
	if !businessWorkspaceManagedPathProbe(added.ManagedFiles, secondaryRel) {
		t.Fatalf("new Business source did not enter managed source set: %+v", added.ManagedFiles)
	}
	addedIndex, err := workspace.BuildSourceIndex(root, added)
	if err != nil {
		t.Fatalf("source index after source add: %v", err)
	}
	if _, ok := businessWorkspaceDiscoveryArtifactProbe(addedIndex.Artifacts, secondaryRel); !ok {
		t.Fatalf("new Business source did not enter Source Index: %+v", addedIndex.Artifacts)
	}

	if err := os.Remove(secondaryPath); err != nil {
		t.Fatalf("remove secondary Business source: %v", err)
	}
	removed, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("workspace load after source remove: %v", err)
	}
	if businessWorkspaceManagedPathProbe(removed.ManagedFiles, secondaryRel) {
		t.Fatalf("removed Business source remained in managed source set: %+v", removed.ManagedFiles)
	}
	removedIndex, err := workspace.BuildSourceIndex(root, removed)
	if err != nil {
		t.Fatalf("source index after source remove: %v", err)
	}
	if _, ok := businessWorkspaceDiscoveryArtifactProbe(removedIndex.Artifacts, secondaryRel); ok {
		t.Fatalf("removed Business source remained in Source Index: %+v", removedIndex.Artifacts)
	}
}

func businessWorkspaceManagedPathProbe(values []workspace.ManagedFile, path string) bool {
	for _, value := range values {
		if value.Path == path {
			return true
		}
	}
	return false
}
