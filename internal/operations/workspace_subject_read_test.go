package operations

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func documentDraftFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(relative, body string) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: subject_read_test\nformatVersion: 1\nsources:\n  - semantic\n")
	write("semantic/sales/order.tei", "namespace sales\n\ndata Order {\n    attribute orderId type std.text\n}\n")
	return root
}

// TestReadWorkspaceSubjectAnswersOneLoad pins that the shared read returns the
// same canonical projection InspectSubject returns, plus the managed Source
// revision, from a single Workspace load.
func TestReadWorkspaceSubjectAnswersOneLoad(t *testing.T) {
	root := documentDraftFixture(t)
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	read, err := ReadWorkspaceSubject(root, subject)
	if err != nil {
		t.Fatalf("ReadWorkspaceSubject: %v", err)
	}
	inspection, err := InspectSubject(root, subject)
	if err != nil {
		t.Fatalf("InspectSubject: %v", err)
	}
	if read.Inspection.SchemaVersion != inspection.SchemaVersion ||
		read.Inspection.Effect != inspection.Effect ||
		string(read.Inspection.Canonical) != string(inspection.Canonical) ||
		read.Inspection.WorldFingerprint != inspection.WorldFingerprint ||
		read.Inspection.SourceIndexFingerprint != inspection.SourceIndexFingerprint ||
		read.Inspection.SourceStatus != inspection.SourceStatus ||
		len(read.Inspection.Sources.References) != len(inspection.Sources.References) {
		t.Fatalf("shared read disagrees with InspectSubject:\n got=%+v\nwant=%+v",
			read.Inspection, inspection)
	}
	if read.Inspection.Workspace == nil || read.Inspection.Workspace.Name != "subject_read_test" {
		t.Fatalf("workspace identity=%+v", read.Inspection.Workspace)
	}
	if !strings.HasPrefix(read.ModelRevision, "sha256:") {
		t.Fatalf("modelRevision=%q", read.ModelRevision)
	}
	if read.ModelRevision != sourceindex.ModelRevision(mustIndex(t, root)) {
		t.Fatal("model revision is not the managed Source revision")
	}
}

// TestReadWorkspaceSubjectKeepsInspectionErrorCodes pins the failure codes a
// document projection depends on: a missing element is not a generic failure.
func TestReadWorkspaceSubjectKeepsInspectionErrorCodes(t *testing.T) {
	root := documentDraftFixture(t)
	_, err := ReadWorkspaceSubject(root, sourceindex.Subject{Kind: "semantic-data", ID: "sales.nothing"})
	var inspectionErr *InspectionError
	if !errors.As(err, &inspectionErr) || inspectionErr.Code != InspectionNotFound {
		t.Fatalf("err=%v", err)
	}
	_, err = ReadWorkspaceSubject(root, sourceindex.Subject{Kind: "semantic-period", ID: "sales.close"})
	if !errors.As(err, &inspectionErr) || inspectionErr.Code != InspectionUnsupportedKind {
		t.Fatalf("err=%v", err)
	}
}

// TestInspectionFromLoadReadsNoFilesystem pins that the second entry point is a
// pure projection over a load the caller already holds.
func TestInspectionFromLoadReadsNoFilesystem(t *testing.T) {
	root := documentDraftFixture(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("BuildSourceIndex: %v", err)
	}
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	fromLoad, err := InspectionFromLoad(loaded, idx, subject)
	if err != nil {
		t.Fatalf("InspectionFromLoad: %v", err)
	}
	direct, err := InspectSubject(root, subject)
	if err != nil {
		t.Fatalf("InspectSubject: %v", err)
	}
	if string(fromLoad.Canonical) != string(direct.Canonical) ||
		fromLoad.SourceIndexFingerprint != direct.SourceIndexFingerprint {
		t.Fatalf("from-load projection differs:\n got=%+v\nwant=%+v", fromLoad, direct)
	}
}

func mustIndex(t *testing.T, root string) sourceindex.Index {
	t.Helper()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}


// TestReadWorkspaceSubjectRejectsSameOffsetRetargetAndTracksSourceOnlyDrift
// characterizes a real Loader/Index read for a UI or harness that pins an
// exact semantic Subject. A byte offset or matching canonical value alone is
// not a durable selection ticket or current edit authority.
func TestReadWorkspaceSubjectRejectsSameOffsetRetargetAndTracksSourceOnlyDrift(t *testing.T) {
	root := documentDraftFixture(t)
	semanticPath := filepath.Join(root, "semantic", "sales", "order.tei")
	original, err := os.ReadFile(semanticPath)
	if err != nil {
		t.Fatal(err)
	}
	oldSubject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	originalRead, err := ReadWorkspaceSubject(root, oldSubject)
	if err != nil {
		t.Fatalf("original subject read: %v", err)
	}
	originalRefs := mustIndex(t, root).ReferencesFor(oldSubject)
	if len(originalRefs) != 1 {
		t.Fatalf("original source references = %d, want one", len(originalRefs))
	}

	// A Source-only edit is a new managed Source revision even when the
	// semantic World, subject and canonical JSON remain unchanged.
	withTrailingWhitespace := append(append([]byte(nil), original...), '\n')
	if err := os.WriteFile(semanticPath, withTrailingWhitespace, 0o644); err != nil {
		t.Fatal(err)
	}
	sourceEdited, err := ReadWorkspaceSubject(root, oldSubject)
	if err != nil {
		t.Fatalf("read after source-only edit: %v", err)
	}
	if originalRead.ModelRevision == sourceEdited.ModelRevision ||
		originalRead.Inspection.SourceIndexFingerprint == sourceEdited.Inspection.SourceIndexFingerprint {
		t.Fatal("source-only revision drift was not visible to the pinned subject reader")
	}
	if originalRead.Inspection.WorldFingerprint != sourceEdited.Inspection.WorldFingerprint ||
		string(originalRead.Inspection.Canonical) != string(sourceEdited.Inspection.Canonical) {
		t.Fatal("source-only edit unexpectedly changed the typed subject")
	}

	// Both declaration names have exactly five bytes. The new subject's
	// Source range therefore occupies the same location as the old one.
	// A consumer that keeps only the source span would silently retarget.
	retargeted := strings.Replace(string(original), "data Order", "data Other", 1)
	if retargeted == string(original) || len(retargeted) != len(original) {
		t.Fatal("invalid same-offset negative fixture")
	}
	if err := os.WriteFile(semanticPath, []byte(retargeted), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ReadWorkspaceSubject(root, oldSubject)
	var missing *InspectionError
	if !errors.As(err, &missing) || missing.Code != InspectionNotFound {
		t.Fatalf("old pinned subject must not retarget to new declaration: %v", err)
	}
	newSubject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.other"}
	current, err := ReadWorkspaceSubject(root, newSubject)
	if err != nil {
		t.Fatalf("new subject read: %v", err)
	}
	currentRefs := mustIndex(t, root).ReferencesFor(newSubject)
	if len(currentRefs) != 1 || currentRefs[0].ArtifactPath != originalRefs[0].ArtifactPath ||
		currentRefs[0].Range != originalRefs[0].Range {
		t.Fatalf("same-position negative fixture did not preserve source location: old=%+v new=%+v",
			originalRefs, currentRefs)
	}
	if current.ModelRevision == originalRead.ModelRevision ||
		current.Inspection.Subject == originalRead.Inspection.Subject {
		t.Fatal("distinct Source and semantic Subject were collapsed by same byte position")
	}

	// A->B->A can recover the same descriptive revision, but does not prove
	// the intermediate state never existed or restore an expired edit grant.
	if err := os.WriteFile(semanticPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	restored, err := ReadWorkspaceSubject(root, oldSubject)
	if err != nil {
		t.Fatalf("restored subject read: %v", err)
	}
	if restored.ModelRevision != originalRead.ModelRevision ||
		restored.Inspection.WorldFingerprint != originalRead.Inspection.WorldFingerprint ||
		string(restored.Inspection.Canonical) != string(originalRead.Inspection.Canonical) {
		t.Fatal("restored bytes no longer yield the same descriptive subject read")
	}
}


// Two Workspaces can expose the same Canonical Subject and byte-identical
// semantic Source yet be distinct authoring and consumer scopes. A pinned
// target must not use world/source fingerprints as its Workspace identity.
func TestReadWorkspaceSubjectDoesNotCollapseSameModelAcrossWorkspaces(t *testing.T) {
	firstRoot := documentDraftFixture(t)
	secondRoot := documentDraftFixture(t)
	manifestPath := filepath.Join(secondRoot, "tei.workspace.yaml")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(manifest), "workspace: subject_read_test", "workspace: another_read_test", 1)
	if changed == string(manifest) {
		t.Fatal("second Workspace fixture did not change its declared identity")
	}
	if err := os.WriteFile(manifestPath, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	first, err := ReadWorkspaceSubject(firstRoot, subject)
	if err != nil {
		t.Fatalf("first Workspace read: %v", err)
	}
	second, err := ReadWorkspaceSubject(secondRoot, subject)
	if err != nil {
		t.Fatalf("second Workspace read: %v", err)
	}
	if first.Inspection.WorldFingerprint != second.Inspection.WorldFingerprint ||
		first.Inspection.SourceIndexFingerprint != second.Inspection.SourceIndexFingerprint ||
		first.ModelRevision != second.ModelRevision ||
		string(first.Inspection.Canonical) != string(second.Inspection.Canonical) {
		t.Fatal("only the Workspace declaration changed, but semantic Source or meaning also changed")
	}
	if first.Inspection.Workspace == nil || second.Inspection.Workspace == nil {
		t.Fatal("inspect omitted the authoring Workspace identity")
	}
	if first.Inspection.Workspace.Name == second.Inspection.Workspace.Name ||
		first.Inspection.Workspace.Manifest.SHA256 == second.Inspection.Workspace.Manifest.SHA256 {
		t.Fatal("different Workspaces collapsed into one identity despite equal semantic fingerprints")
	}
	// This test observes identity, not a current authorization, a security
	// namespace or a qualified cross-Workspace edit operation.
}
