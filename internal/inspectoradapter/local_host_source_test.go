package inspectoradapter

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
)

// The chosen Provider ID and Subject are identical for two workspaces.
// This is not evidence that the Provider read the correct Source root.
// Each Host-selected instance must retain the root it was actually bound to,
// and Source-only revisions must be visible without a new Host selection.
func TestLocalInspectorSelectedRouteKeepsWorkspaceAndSourceIdentity(t *testing.T) {
	const manifestA = "workspace: selected_alpha\nformatVersion: 1\nsources:\n  - semantic\n"
	const manifestB = "workspace: selected_beta\nformatVersion: 1\nsources:\n  - semantic\n"
	const sourceA = "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"
	const sourceB = "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n    attribute customerId type std.identifier\n}\n"
	makeWorkspace := func(manifest, source string) string {
		t.Helper()
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "semantic"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "semantic", "first.tei"), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
		return root
	}
	rootA := makeWorkspace(manifestA, sourceA)
	rootB := makeWorkspace(manifestB, sourceB)

	host, err := newLocalInspectorHostFromObservedDigest(inspectorArtifact)
	if err != nil {
		t.Fatal(err)
	}
	selectedA, err := host.Bind(rootA)
	if err != nil {
		t.Fatal(err)
	}
	selectedB, err := host.Bind(rootB)
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	firstA, err := selectedA.Inspect(subject)
	if err != nil {
		t.Fatal(err)
	}
	firstB, err := selectedB.Inspect(subject)
	if err != nil {
		t.Fatal(err)
	}
	if firstA.Subject != subject || firstB.Subject != subject ||
		firstA.WorldFingerprint == firstB.WorldFingerprint ||
		bytes.Equal(firstA.Canonical, firstB.Canonical) ||
		firstA.SourceStatus != "indexed" || firstB.SourceStatus != "indexed" ||
		len(firstA.Sources.References) == 0 || len(firstB.Sources.References) == 0 ||
		firstA.Workspace == nil || firstB.Workspace == nil ||
		*firstA.Workspace == *firstB.Workspace {
		t.Fatalf("selected Inspector crossed Workspace boundaries: A=%+v B=%+v", firstA, firstB)
	}
	// The Source path alone changes; the canonical World does not. The
	// selected Provider stays the same and must observe a fresh Source Index.
	if err := os.Rename(
		filepath.Join(rootA, "semantic", "first.tei"),
		filepath.Join(rootA, "semantic", "moved.tei"),
	); err != nil {
		t.Fatal(err)
	}
	secondA, err := selectedA.Inspect(subject)
	if err != nil {
		t.Fatal(err)
	}
	secondB, err := selectedB.Inspect(subject)
	if err != nil {
		t.Fatal(err)
	}
	if secondA.WorldFingerprint != firstA.WorldFingerprint ||
		!bytes.Equal(secondA.Canonical, firstA.Canonical) ||
		secondA.SourceIndexFingerprint == firstA.SourceIndexFingerprint ||
		secondA.SourceIndexFingerprint == "" ||
		len(secondA.Sources.References) == 0 ||
		secondA.Sources.References[0].ArtifactPath == firstA.Sources.References[0].ArtifactPath ||
		secondA.Workspace == nil || *secondA.Workspace != *firstA.Workspace {
		t.Fatalf("Source-only movement not observed by selected Provider: before=%+v after=%+v", firstA, secondA)
	}
	if secondB.WorldFingerprint != firstB.WorldFingerprint ||
		secondB.SourceIndexFingerprint != firstB.SourceIndexFingerprint ||
		!bytes.Equal(secondB.Canonical, firstB.Canonical) {
		t.Fatalf("unmodified Workspace B changed with Workspace A: before=%+v after=%+v", firstB, secondB)
	}
}

func TestLocalInspectorSameCanonicalDifferentWorkspaceIdentity(t *testing.T) {
	// A pair of workspaces can have byte-identical semantic declarations and
	// return the same Canonical/World fingerprint while differing in manifest
	// identity. An identical Provider, Subject and result body do not establish
	// an identical workspace, current Source admission or authorization.
	makeWorkspace := func(name string) string {
		t.Helper()
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "semantic"), 0755); err != nil {
			t.Fatal(err)
		}
		manifest := "workspace: " + name + "\nformatVersion: 1\nsources:\n  - semantic\n"
		source := "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"
		if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "semantic", "order.tei"), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
		return root
	}
	host, err := newLocalInspectorHostFromObservedDigest(inspectorArtifact)
	if err != nil {
		t.Fatal(err)
	}
	var results []string
	var manifests []string
	for _, name := range []string{"alpha", "beta"} {
		selected, err := host.Bind(makeWorkspace(name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := selected.Inspect(sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"})
		if err != nil || got.Workspace == nil || got.Workspace.Name != name {
			t.Fatalf("workspace %q not observed: %+v %v", name, got, err)
		}
		results = append(results, string(got.Canonical)+"/"+got.WorldFingerprint)
		manifests = append(manifests, got.Workspace.Manifest.SHA256)
	}
	if results[0] != results[1] || manifests[0] == "" || manifests[1] == "" ||
		manifests[0] == manifests[1] {
		t.Fatalf("same Canonical World failed to distinguish Workspace manifests: results=%v manifests=%v", results, manifests)
	}
}
