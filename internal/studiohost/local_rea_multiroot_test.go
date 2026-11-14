package studiohost

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Two different actual Workspace roots may contain different saved bytes with
// the same producer-declared Evidence ID, node and source revision. The Host
// must keep its chosen root and raw-byte pin, not infer an identical Worktree
// or Current Grant from the external identity.
func TestLocalREAReaderKeepsSameProducerEvidenceSeparateAcrossRoots(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json"))
	if err != nil {
		t.Fatal(err)
	}
	otherRaw := append(append([]byte{}, raw...), '\n')
	otherDigest := sha256.Sum256(otherRaw)
	rootA, rootB := t.TempDir(), t.TempDir()
	for _, item := range []struct {
		root string
		raw  []byte
	}{{rootA, raw}, {rootB, otherRaw}} {
		if err := os.WriteFile(filepath.Join(item.root, "analysis.json"), item.raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	host, err := NewReadOnly([]string{rootA, rootB})
	if err != nil {
		t.Fatal(err)
	}
	a := LocalREABinding{
		WorkspaceID: "workspace-1", Selector: "shared-selector", File: "analysis.json",
		Version: "6.8.0", EvidenceID: localREAEvidenceID, Digest: localREADigest,
	}
	b := a
	b.WorkspaceID = "workspace-2"
	b.Digest = fmt.Sprintf("sha256:%x", otherDigest)
	if b.Digest == a.Digest {
		t.Fatal("fixture must have distinct saved-byte revisions")
	}
	reader, err := NewLocalREAReader(host, []LocalREABinding{a, b})
	if err != nil {
		t.Fatal(err)
	}
	left := reader.Read("workspace-1", "shared-selector", localREANodeID, localREARelationID)
	right := reader.Read("workspace-2", "shared-selector", localREANodeID, localREARelationID)
	if !left.OK || !right.OK || left.Value == nil || right.Value == nil {
		t.Fatalf("valid separate roots: left=%+v right=%+v", left.Failure, right.Failure)
	}
	if left.Value.Attempt.EvidenceID != right.Value.Attempt.EvidenceID ||
		left.Value.Attempt.EvidenceID != localREAEvidenceID ||
		left.Value.Attempt.ReceivedDigest != a.Digest ||
		right.Value.Attempt.ReceivedDigest != b.Digest ||
		left.Value.Attempt.SourceRevision != right.Value.Attempt.SourceRevision ||
		left.Value.Observation.NodeID != right.Value.Observation.NodeID ||
		left.Value.Relation.RelationID != right.Value.Relation.RelationID {
		t.Fatalf("producer identity or host-bound raw bytes were confused: left=%+v right=%+v", left.Value.Attempt, right.Value.Attempt)
	}

	// Swapping the contents between two still-allowlisted roots cannot make
	// either original selector silently follow the identical Evidence ID.
	if err := os.WriteFile(filepath.Join(rootA, "analysis.json"), otherRaw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "analysis.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, workspaceID := range []string{"workspace-1", "workspace-2"} {
		got := reader.Read(workspaceID, "shared-selector", localREANodeID, localREARelationID)
		if got.OK || got.Value != nil || got.Failure == nil || got.Failure.Code != "OPERATION_FAILED" {
			t.Fatalf("swapped producer Evidence was accepted for %s: %+v", workspaceID, got)
		}
	}
}

// A changed Workspace root must not invalidate an unrelated pinned root, or
// accidentally retarget a selector to that other root's Evidence.
func TestLocalREAReaderRootReplacementDoesNotRetargetOtherWorkspace(t *testing.T) {
	rootA, replacement := rootReplacementPair(t)
	rootB := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{rootA, rootB} {
		if err := os.WriteFile(filepath.Join(root, "analysis.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	host, err := NewReadOnly([]string{rootA, rootB})
	if err != nil {
		t.Fatal(err)
	}
	a := LocalREABinding{
		WorkspaceID: "workspace-1", Selector: "shared-selector", File: "analysis.json",
		Version: "6.8.0", EvidenceID: localREAEvidenceID, Digest: localREADigest,
	}
	b := a
	b.WorkspaceID = "workspace-2"
	reader, err := NewLocalREAReader(host, []LocalREABinding{a, b})
	if err != nil {
		t.Fatal(err)
	}
	replaceWorkspaceRoot(t, rootA, replacement)
	changed := reader.Read("workspace-1", "shared-selector", localREANodeID, localREARelationID)
	if changed.OK || changed.Value != nil || changed.Failure == nil || changed.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("changed first root was not rejected: %+v", changed)
	}
	other := reader.Read("workspace-2", "shared-selector", localREANodeID, localREARelationID)
	if !other.OK || other.Failure != nil || other.Value == nil || other.Value.Attempt.ReceivedDigest != localREADigest {
		t.Fatalf("valid second root was damaged or retargeted: %+v", other)
	}
}
