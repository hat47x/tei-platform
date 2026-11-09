package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func writeIdentityFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Identity must report the name and manifest version the author declared, so
// a caller never has to infer which Workspace a read came from (#3181). The
// same Canonical ID can exist in two Workspaces with different meaning.
func TestIdentityReportsDeclaredNameAndManifest(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "tei.workspace.yaml", "workspace: identity_probe\nformatVersion: 1\nsources:\n  - semantic\n")
	result, err := Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got := result.Identity()
	if got.Name != "identity_probe" || got.FormatVersion != 1 {
		t.Fatalf("identity = %+v", got)
	}
	if got.Manifest.Path != "tei.workspace.yaml" || got.Manifest.SHA256 == "" {
		t.Fatalf("manifest identity = %+v", got.Manifest)
	}
	if same := result.ManifestIdentity(); same != got.Manifest {
		t.Fatalf("Identity and ManifestIdentity disagree: %+v vs %+v", got.Manifest, same)
	}
}

// A manifest without a name still loads: the name is identity, not a required
// source declaration. The result must report an empty name rather than
// inventing a path-derived one.
func TestIdentityWithoutNameIsEmptyNotEmptyInvented(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "tei.workspace.yaml", "formatVersion: 1\nsources:\n  - semantic\n")
	result, err := Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := result.Identity(); got.Name != "" || got.FormatVersion != 1 {
		t.Fatalf("identity = %+v", got)
	}
}

// Two Workspaces may declare the same model. Identity (name + manifest digest)
// separates them; the fingerprint identifies content, not the Workspace.
func TestIdentitySeparatesTwoWorkspacesWithTheSameModel(t *testing.T) {
	makeWorkspace := func(name string) Result {
		root := t.TempDir()
		writeIdentityFixture(t, root, "tei.workspace.yaml",
			"workspace: "+name+"\nformatVersion: 1\nsources:\n  - semantic\n")
		writeIdentityFixture(t, root, "semantic/order.tei",
			"namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n")
		result, err := Load(root)
		if err != nil {
			t.Fatalf("%s: load: %v", name, err)
		}
		return result
	}
	first, second := makeWorkspace("identity_a"), makeWorkspace("identity_b")
	if first.Identity().Name == second.Identity().Name {
		t.Fatalf("names collided: %+v / %+v", first.Identity(), second.Identity())
	}
	if first.Identity().Manifest.SHA256 == second.Identity().Manifest.SHA256 {
		t.Fatalf("manifests did not differ: %+v", first.Identity().Manifest)
	}
}

// A duplicate `workspace:` line is rejected instead of silently taking the
// last value, so the reported identity cannot be ambiguous.
func TestDuplicateWorkspaceNameIsRejected(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "tei.workspace.yaml",
		"workspace: first\nworkspace: second\nformatVersion: 1\nsources:\n  - semantic\n")
	if _, err := Load(root); err == nil {
		t.Fatal("duplicate workspace name was accepted")
	}
}
