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
