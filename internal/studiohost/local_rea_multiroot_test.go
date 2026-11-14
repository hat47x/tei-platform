package studiohost

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Two different actual Workspace roots can carry byte-identical producer
// Evidence without thereby becoming the same Worktree, Principal or Grant.
// Even if a Host binding is rewritten to match mutated file bytes, the R2
// parser must refuse a broken producer delivery instead of accepting it.
func TestLocalREAReaderKeepsSameProducerEvidenceSeparateAcrossRoots(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json"))
	if err != nil {
		t.Fatal(err)
	}
	rootA, rootB := t.TempDir(), t.TempDir()
	for _, item := range []struct {
		root string
		raw  []byte
	}{{rootA, raw}, {rootB, raw}} {
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
		right.Value.Attempt.ReceivedDigest != a.Digest ||
		left.Value.Attempt.SourceRevision != right.Value.Attempt.SourceRevision ||
		left.Value.Observation.NodeID != right.Value.Observation.NodeID ||
		left.Value.Relation.RelationID != right.Value.Relation.RelationID {
		t.Fatalf("producer identity or host-bound raw bytes were confused: left=%+v right=%+v", left.Value.Attempt, right.Value.Attempt)
	}

	// A mutated saved delivery with the *same* producer Evidence ID must not
	// cross from Root B into Root A, even when both allowlisted roots exist.
	otherRaw := append(append([]byte{}, raw...), '\n')
	if err := os.WriteFile(filepath.Join(rootB, "analysis.json"), otherRaw, 0600); err != nil {
		t.Fatal(err)
	}
	mutated := reader.Read("workspace-2", "shared-selector", localREANodeID, localREARelationID)
	if mutated.OK || mutated.Value != nil || mutated.Failure == nil || mutated.Failure.Code != "OPERATION_FAILED" {
		t.Fatalf("mutated Evidence was accepted for second root: %+v", mutated)
	}
	// Even updating the Host's raw-byte pin to the modified file does not
	// turn a broken producer delivery into usable Evidence. Keep this
	// separate from the simple stale-Host-Pin refusal above.
	changedDigest := sha256.Sum256(otherRaw)
	rebound := b
	rebound.Digest = fmt.Sprintf("sha256:%x", changedDigest)
	reboundReader, err := NewLocalREAReader(host, []LocalREABinding{a, rebound})
	if err != nil {
		t.Fatal(err)
	}
	repinned := reboundReader.Read("workspace-2", "shared-selector", localREANodeID, localREARelationID)
	if repinned.OK || repinned.Value != nil || repinned.Failure == nil || repinned.Failure.Code != "OPERATION_FAILED" {
		t.Fatalf("repinning a changed producer delivery must not repair it: %+v", repinned)
	}
	stillValid := reader.Read("workspace-1", "shared-selector", localREANodeID, localREARelationID)
	if !stillValid.OK || stillValid.Failure != nil || stillValid.Value == nil ||
		stillValid.Value.Attempt.ReceivedDigest != localREADigest {
		t.Fatalf("mutation of other root invalidated or retargeted valid root: %+v", stillValid)
	}
	// Restoring the exact captured byte is a new local read. It does not
	// prove that a source revision or Grant stayed current during A-B-A.
	if err := os.WriteFile(filepath.Join(rootB, "analysis.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	restored := reader.Read("workspace-2", "shared-selector", localREANodeID, localREARelationID)
	if !restored.OK || restored.Value == nil || restored.Value.Attempt.ReceivedDigest != localREADigest {
		t.Fatalf("restored saved Evidence should be readable but not prove current Source: %+v", restored)
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
