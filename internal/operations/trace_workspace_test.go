package operations

import (
	"errors"
	"path/filepath"
	"os"
	"runtime"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

func TestTraceWorkspaceReadOnlyCrossViewAndSource(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	key := modelref.Key{Scope: "reference-observation", Subject: subject.Subject{
		Kind: "semantic-relation", ID: "sales.order.customer",
	}}
	summary, err := TraceWorkspace(root, key, modelref.TraceOptions{Mode: "summary"})
	if err != nil {
		t.Fatalf("trace summary: %v", err)
	}
	if summary.Trace.Total < 2 || len(summary.Trace.Rows) != 0 ||
		summary.Trace.SnapshotFingerprint == "" ||
		summary.Source.Status != "located" || len(summary.Source.References) == 0 ||
		len(summary.RelatedSources) != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	located := false
	for _, reference := range summary.Source.References {
		if reference.ArtifactPath == "semantic/sales/order.tei" {
			located = true
		}
	}
	if !located {
		t.Fatalf("lost source provenance: %+v", summary.Source)
	}
	full, err := TraceWorkspace(root, key, modelref.TraceOptions{
		Mode: "full", Direction: "both",
		ExpectedFingerprint: summary.Trace.SnapshotFingerprint,
	})
	if err != nil {
		t.Fatalf("trace full: %v", err)
	}
	if full.Trace.Total != summary.Trace.Total ||
		full.Trace.Unresolved != summary.Trace.Unresolved ||
		len(full.Trace.Rows) != full.Trace.Total ||
		len(full.RelatedSources) == 0 {
		t.Fatalf("full/summary mismatch: %+v", full)
	}
	pattern := modelref.Key{Scope: key.Scope, Subject: subject.Subject{
		Kind: "application-pattern", ID: "order_entry.customer_reference",
	}}
	customer := modelref.Key{Scope: key.Scope, Subject: subject.Subject{
		Kind: "semantic-data", ID: "customer.customer",
	}}
	foundIncoming, foundOutgoing := false, false
	for _, edge := range full.Trace.Rows {
		if edge.Status != "resolved" || edge.ResolvedTo == nil {
			continue
		}
		if edge.Edge.From == pattern && *edge.ResolvedTo == key {
			foundIncoming = true
		}
		if edge.Edge.From == key && *edge.ResolvedTo == customer {
			foundOutgoing = true
		}
	}
	if !foundIncoming || !foundOutgoing {
		t.Fatalf("missing declared application/relation/data references: %+v", full.Trace.Rows)
	}
	for _, nav := range full.RelatedSources {
		if nav.Key == key {
			t.Fatalf("duplicate root navigation in neighbor list: %+v", nav)
		}
	}
}

func TestTraceWorkspaceMissingAndStaleFingerprint(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	key := modelref.Key{Scope: "reference-observation", Subject: subject.Subject{
		Kind: "semantic-data", ID: "unknown-data",
	}}
	_, err := TraceWorkspace(root, key, modelref.TraceOptions{Mode: "summary"})
	var semantic *modelref.TraceError
	if !errors.As(err, &semantic) || semantic.Code != "SUBJECT_NOT_FOUND" {
		t.Fatalf("missing subject not rejected: %v", err)
	}
	key.Subject.ID = "sales.order"
	_, err = TraceWorkspace(root, key, modelref.TraceOptions{
		Mode: "full", ExpectedFingerprint: "sha256:stale",
	})
	if !errors.As(err, &semantic) || semantic.Code != "STALE_SNAPSHOT" {
		t.Fatalf("stale trace not rejected: %v", err)
	}
	_, err = TraceWorkspace(root, modelref.Key{Subject: key.Subject}, modelref.TraceOptions{})
	if !errors.As(err, &semantic) || semantic.Code != "INVALID_SUBJECT" {
		t.Fatalf("missing scope accepted: %v", err)
	}
}

func TestTraceWorkspaceLoadFailureStage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "not-present")
	key := modelref.Key{Scope: "reference", Subject: subject.Subject{
		Kind: "semantic-data", ID: "sales.order",
	}}
	_, err := TraceWorkspace(root, key, modelref.TraceOptions{})
	var failure *ValidationFailure
	if !errors.As(err, &failure) || failure.Stage != StageLoad {
		t.Fatalf("missing Workspace did not produce load failure: %v", err)
	}
}

func TestReadWorkspaceSourceRejectsAbsoluteAndEscapedPaths(t *testing.T) {
	root := t.TempDir()
	reader, err := readWorkspaceSource(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside.tei", "/etc/passwd", ""} {
		if _, err := reader(path); err == nil {
			t.Fatalf("unsafe source path accepted: %q", path)
		}
	}
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated Windows permissions")
	}
	outside := t.TempDir()
	target := filepath.Join(outside, "outside.tei")
	if err := os.WriteFile(target, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "escape.tei")); err != nil {
		t.Skipf("platform cannot create symlink: %v", err)
	}
	if _, err := reader("escape.tei"); err == nil {
		t.Fatal("symlink escaped Workspace read boundary")
	}
}
