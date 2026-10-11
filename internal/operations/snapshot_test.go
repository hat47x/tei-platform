package operations

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/ir"
	"tei/internal/modelref"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestWorkspaceProjectionFingerprintsKeepMeaningAndSourceSeparate(t *testing.T) {
	world := ir.World{}
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{{
		Path: "semantic/order.tei", Fingerprint: "sha256:first",
	}}}
	w1, s1, err := workspaceProjectionFingerprints(world, index)
	if err != nil || w1 == "" || s1 == "" {
		t.Fatalf("initial fingerprints: %q %q %v", w1, s1, err)
	}
	w2, s2, err := workspaceProjectionFingerprints(world, index)
	if err != nil || w1 != w2 || s1 != s2 {
		t.Fatalf("unchanged projection changed fingerprints: %q %q %v", w2, s2, err)
	}
	index.Artifacts[0].Path = "semantic/moved.tei"
	w3, s3, err := workspaceProjectionFingerprints(world, index)
	if err != nil || w1 != w3 || s1 == s3 {
		t.Fatalf("source move changed meaning or kept stale provenance: %q %q %v", w3, s3, err)
	}
	if workspaceTraceFingerprint(w1, s1, "graph") ==
		workspaceTraceFingerprint(w3, s3, "graph") {
		t.Fatal("workspace trace continuation ignored Source Index changes")
	}
	if workspaceTraceFingerprint(w1, s1, "graph") ==
		workspaceTraceFingerprint(w1, s1, "other-graph") {
		t.Fatal("workspace trace continuation ignored Graph changes")
	}
}

func TestTraceWorkspaceRejectsMovedAndEditedSourceOnContinuation(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "semantic")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("workspace: trace_snapshot\nsources:\n  - semantic\n")
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(sourceDir, "order.tei")
	moved := filepath.Join(sourceDir, "renamed.tei")
	definition := []byte("namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n")
	if err := os.WriteFile(original, definition, 0644); err != nil {
		t.Fatal(err)
	}
	key := modelref.Key{Scope: "test-projection", Subject: subject.Subject{
		Kind: "semantic-data", ID: "sales.order",
	}}
	observe := func() WorkspaceTrace {
		t.Helper()
		result, err := TraceWorkspace(root, key, modelref.TraceOptions{Mode: "summary"})
		if err != nil {
			t.Fatalf("trace: %v", err)
		}
		return result
	}
	first := observe()
	if first.SnapshotFingerprint == "" || first.WorldFingerprint == "" ||
		first.SourceIndexFingerprint == "" || first.Trace.SnapshotFingerprint == "" {
		t.Fatalf("fingerprints missing: %+v", first)
	}
	inspectFirst, err := InspectSubject(root, key.Subject)
	if err != nil {
		t.Fatal(err)
	}
	if inspectFirst.WorldFingerprint != first.WorldFingerprint ||
		inspectFirst.SourceIndexFingerprint != first.SourceIndexFingerprint {
		t.Fatalf("inspect and trace disagree on first projection")
	}
	if err := os.Rename(original, moved); err != nil {
		t.Fatal(err)
	}
	assertStale := func(old string) {
		t.Helper()
		_, err := TraceWorkspace(root, key, modelref.TraceOptions{
			Mode: "summary", ExpectedFingerprint: old,
		})
		var traceErr *modelref.TraceError
		if !errors.As(err, &traceErr) || traceErr.Code != "STALE_SNAPSHOT" {
			t.Fatalf("source changed but previous trace continuation accepted: %v", err)
		}
	}
	assertStale(first.SnapshotFingerprint)
	second := observe()
	if second.WorldFingerprint != first.WorldFingerprint ||
		second.Trace.SnapshotFingerprint != first.Trace.SnapshotFingerprint ||
		second.SourceIndexFingerprint == first.SourceIndexFingerprint ||
		second.SnapshotFingerprint == first.SnapshotFingerprint {
		t.Fatalf("moving Source must preserve meaning/graph and change Source snapshot: first=%+v second=%+v", first, second)
	}
	inspectSecond, err := InspectSubject(root, key.Subject)
	if err != nil {
		t.Fatal(err)
	}
	if inspectSecond.WorldFingerprint != second.WorldFingerprint ||
		inspectSecond.SourceIndexFingerprint != second.SourceIndexFingerprint {
		t.Fatalf("inspect and trace disagree after source move")
	}
	if len(second.Source.References) != 1 ||
		second.Source.References[0].ArtifactPath != "semantic/renamed.tei" {
		t.Fatalf("source navigation did not follow move: %+v", second.Source)
	}
	// Whitespace in a model file changes the artifact digest while retaining
	// the exact same Canonical World and typed graph.
	if err := os.WriteFile(moved, append(definition, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	assertStale(second.SnapshotFingerprint)
	third := observe()
	if third.WorldFingerprint != second.WorldFingerprint ||
		third.Trace.SnapshotFingerprint != second.Trace.SnapshotFingerprint ||
		third.SourceIndexFingerprint == second.SourceIndexFingerprint ||
		third.SnapshotFingerprint == second.SnapshotFingerprint {
		t.Fatalf("nonsemantic source edit was not tracked independently: second=%+v third=%+v", second, third)
	}
}
