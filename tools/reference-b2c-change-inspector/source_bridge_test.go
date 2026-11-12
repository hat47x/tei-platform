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
	b0OrderA     = "namespace sales\n\ndata Order {\n    attribute orderId type std.text\n}\n"
	b0OrderB     = "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"
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
	revB := sourceindex.ModelRevision(indexB)
	if revB == revA {
		t.Fatal("source-only change failed to invalidate managed source revision")
	}
	if b0Artifact(t, inspectionB, b0SourcePath).Fingerprint == expectedBytesFingerprint {
		t.Fatal("updated source kept the original bytes fingerprint")
	}

	// Use two actual managed Source revisions for the hypothetical Native
	// binding. This still cannot prove the native client's required closure.
	in := input{
		Schema: inputSchema, Scope: "native-pdp-undiscovered",
		BaseRevision: revA, CandidateRevision: revB,
		DeclaredScopeClosed: false,
		Nodes: []node{{ID: "semantic-data:sales.order", Layer: "business"}},
		Changed: []string{"semantic-data:sales.order"},
		Required: []string{"semantic-data:sales.order"},
		Unobserved: []string{"native-client-dynamic-edge"},
	}
	out, err := inspect(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Coverage != "partial-or-unverified" || len(out.Lanes) != 1 || out.Lanes[0] != "C3" ||
		out.Disposition != "known-interaction-business-or-release-impact" {
		t.Fatalf("real Source revisions were mistaken for native closure: %+v", out)
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
	// The existing inspection states its authority limits explicitly.
	// A real indexed Query does not confer release, effect, disclosure, or
	// complete native dependency coverage.
	if len(inspection.AuthorityLimits) == 0 {
		t.Fatal("inspection lacks read-only authority limits")
	}
}


// TestB0ManifestIdentityIsNotManagedSourceRevision detects an ABA-like
// misbinding in which the managed Source bytes are unchanged while the
// Workspace manifest changes. ModelRevision alone is not a Workspace identity.
func TestB0ManifestIdentityIsNotManagedSourceRevision(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "tei.workspace.yaml")
	writeManifest := func(name string) {
		t.Helper()
		content := "workspace: " + name + "\nformatVersion: 1\nsources:\n  - semantic\n"
		if err := os.WriteFile(manifestPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest("b2c_source_a")
	b0Write(t, root, b0OrderA)
	_, firstIndex, first := b0StrictRead(t, root)

	writeManifest("b2c_source_b")
	_, secondIndex, second := b0StrictRead(t, root)
	if sourceindex.ModelRevision(firstIndex) != sourceindex.ModelRevision(secondIndex) {
		t.Fatal("same managed Source bytes changed their content revision")
	}
	if first.Workspace.Name == second.Workspace.Name {
		t.Fatalf("manifest drift was not visible in actual Workspace identity: %v %v", first.Workspace, second.Workspace)
	}
	if b0Artifact(t, first, b0SourcePath).Fingerprint != b0Artifact(t, second, b0SourcePath).Fingerprint {
		t.Fatal("manifest-only change modified managed Source artifact fingerprint")
	}
	// The test intentionally does not claim an effective consumer Contract
	// or present-day authorization based on the equal content revision.
}
