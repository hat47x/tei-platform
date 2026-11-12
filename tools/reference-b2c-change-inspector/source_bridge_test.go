package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const (
	b0SourcePath = "semantic/sales/order.tei"
	b0OrderA = "namespace sales\n\ndata Order {\n    attribute orderId type std.text\n}\n"
	b0OrderB = "namespace sales\n\ndata Order {\n    attribute orderId type std.code\n}\n"
)

// b0StrictRead intentionally reuses the product Workspace loader, the index
// built solely from Load-time bytes, and the product Subject projection.
// It is a test helper, NOT a new B2C Source Reader or public API.
func b0StrictRead(t *testing.T, root string) (workspace.Result, sourceindex.Index, operations.SubjectInspection) {
	t.Helper()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	got, err := operations.InspectionFromLoad(loaded, index, subject)
	if err != nil {
		t.Fatal(err)
	}
	if got.Effect != "read-only" || got.Subject != subject || got.SourceStatus != "indexed" ||
		got.Workspace == nil || got.Workspace.Name == "" || len(got.Sources.References) == 0 {
		t.Fatalf("inspection was not bound to a real indexed subject: %+v", got)
	}
	return loaded, index, got
}

func b0Artifact(t *testing.T, got operations.SubjectInspection, path string) sourceindex.Artifact {
	t.Helper()
	for _, artifact := range got.Sources.Artifacts {
		if artifact.Path == path {
			return artifact
		}
	}
	t.Fatalf("subject has no managed source artifact %s: %+v", path, got.Sources)
	return sourceindex.Artifact{}
}

func b0Write(t *testing.T, root, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(b0SourcePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestB0PinnedSourceReadSeparatesBytesFromCurrentness verifies the exact
// managed-source bytes of one Load. A->B->A may restore the content revision,
// but a content digest cannot attest acceptance, current grant, read epoch or
// what external Native/Browser sources were in the required closure.
func TestB0PinnedSourceReadSeparatesBytesFromCurrentness(t *testing.T) {
	root := t.TempDir()
	manifest := "workspace: b2c_source_fixture\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	b0Write(t, root, b0OrderA)
	loadedA, indexA, inspectionA := b0StrictRead(t, root)

	sum := sha256.Sum256([]byte(b0OrderA))
	expectedBytesFingerprint := "sha256:" + hex.EncodeToString(sum[:])
	if got := b0Artifact(t, inspectionA, b0SourcePath).Fingerprint; got != expectedBytesFingerprint {
		t.Fatalf("source fingerprint=%s, independently expected=%s", got, expectedBytesFingerprint)
	}
	revA := sourceindex.ModelRevision(indexA)
	if !strings.HasPrefix(revA, "sha256:") {
		t.Fatalf("managed source revision missing: %s", revA)
	}

	// Change disk contents AFTER the initial Load; the strict index must
	// continue to describe A, never silently re-read the now-current B.
	b0Write(t, root, b0OrderB)
	indexAgain, err := workspace.BuildSourceIndexFromLoad(root, loadedA)
	if err != nil {
		t.Fatal(err)
	}
	if sourceindex.ModelRevision(indexAgain) != revA {
		t.Fatalf("Load-time source changed by disk mutation")
	}
	_, indexB, inspectionB := b0StrictRead(t, root)
	if sourceindex.ModelRevision(indexB) == revA {
		t.Fatal("source-only change failed to invalidate managed source revision")
	}
	if b0Artifact(t, inspectionB, b0SourcePath).Fingerprint == expectedBytesFingerprint {
		t.Fatal("updated source kept the original bytes fingerprint")
	}

	// A -> B -> A reuses a content fingerprint, not an acceptance or grant.
	b0Write(t, root, b0OrderA)
	_, indexA2, _ := b0StrictRead(t, root)
	if sourceindex.ModelRevision(indexA2) != revA {
		t.Fatal("identical managed source bytes did not recover their revision")
	}
	if _, err := workspace.BuildSourceIndexFromLoad(t.TempDir(), loadedA); err == nil {
		t.Fatal("Load-time source index accepted a different workspace root")
	}
}

// TestB0RealWorkspaceSourceObservationNotClosure proves that the existing
// managed Reference workspace can supply one actual canonical subject and
// source occurrence without declaring complete B2C impact coverage.
func TestB0RealWorkspaceSourceObservationNotClosure(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, index, err := func() (workspace.Result, sourceindex.Index, error) {
		l, e := workspace.Load(root)
		if e != nil {
			return workspace.Result{}, sourceindex.Index{}, e
		}
		i, e := workspace.BuildSourceIndexFromLoad(root, l)
		return l, i, e
	}()
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"}
	inspection, err := operations.InspectionFromLoad(loaded, index, subject)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Subject != subject || inspection.SourceStatus != "indexed" ||
		len(inspection.Sources.References) == 0 || len(inspection.Sources.Artifacts) == 0 {
		t.Fatalf("Source observation is missing: %+v", inspection)
	}
	// Even a real indexed Query in a Load-bound Source snapshot does NOT prove
	// the presence of a Native frontend's dynamic imports or its complete
	// dependent consumer graph. Do not set DeclaredScopeClosed=true here.
	in := input{
		Schema: inputSchema, Scope: "native-pdp-undiscovered",
		BaseRevision: sourceindex.ModelRevision(index),
		CandidateRevision: sourceindex.ModelRevision(index),
		DeclaredScopeClosed: false,
		Nodes: []node{{ID: "query:order_entry.recent_orders", Layer: "content"}},
		Changed: []string{"query:order_entry.recent_orders"},
		Required: []string{"query:order_entry.recent_orders"},
		Unobserved: []string{"native-client-dynamic-edge"},
	}
	report, err := inspect(in)
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage != "partial-or-unverified" ||
		report.Disposition != "incomplete-scope-needs-review" {
		t.Fatalf("a real Source read became a whole-native impact certificate: %+v", report)
	}
}
