package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
)

func writeManagedSnapshotFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func snapshotFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	writeManagedSnapshotFixture(t, root, "tei.workspace.yaml", "workspace: snapshot\nsources:\n  - semantic\n  - docs\n")
	writeManagedSnapshotFixture(t, root, "semantic/model.tei", "namespace snapshot\n")
	writeManagedSnapshotFixture(t, root, "docs/asset.md", "original bytes\n")
	return root, "docs/asset.md"
}

// A forward-compatible source root is managed by Load even when no domain
// parser consumes it. Its bytes must be captured at Load time so Source Index
// does not re-read the file after Load and IndexedLoadedSnapshot can bind the
// artifact to the same bytes.
func TestLoadCapturesForwardCompatibleManagedAsset(t *testing.T) {
	root, asset := snapshotFixture(t)
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	content, ok := loaded.sourceSnapshot[asset]
	if !ok {
		t.Fatalf("managed asset is absent from the Load snapshot: %+v", loaded.ManagedFiles)
	}
	if string(content) != "original bytes\n" {
		t.Fatalf("snapshot content=%q", content)
	}
	index, err := BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, artifact := range index.Artifacts {
		if artifact.Path == asset {
			found = true
			if artifact.Fingerprint != sourceindex.Fingerprint(content) {
				t.Fatalf("artifact was not tied to Load bytes: %+v", artifact)
			}
		}
	}
	if !found {
		t.Fatalf("managed asset is absent from the Source Index: %+v", index.Artifacts)
	}
	_, read, err := IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatalf("IndexedLoadedSnapshot rejected a complete Load snapshot: %v", err)
	}
	readBack, err := read(asset)
	if err != nil {
		t.Fatal(err)
	}
	if string(readBack) != "original bytes\n" {
		t.Fatalf("read=%q", readBack)
	}
}

// The snapshot is a Load-time authoring record, not the current filesystem.
// Modifying the asset after Load must not change the bound artifact or the
// reader, while a new Load observes the new bytes.
func TestForwardCompatibleManagedAssetKeepsLoadTimeBytes(t *testing.T) {
	root, asset := snapshotFixture(t)
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(asset)), []byte("changed after load\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	index, err := BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	original := sourceindex.Fingerprint([]byte("original bytes\n"))
	changed := sourceindex.Fingerprint([]byte("changed after load\n"))
	for _, artifact := range index.Artifacts {
		if artifact.Path != asset {
			continue
		}
		if artifact.Fingerprint != original {
			t.Fatalf("Source Index re-read the file after Load: %+v", artifact)
		}
	}
	_, read, err := IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	readBack, err := read(asset)
	if err != nil {
		t.Fatal(err)
	}
	if string(readBack) != "original bytes\n" {
		t.Fatalf("IndexedLoadedSnapshot read post-Load bytes: %q", readBack)
	}

	reloaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	reloadedIndex, err := BuildSourceIndex(root, reloaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range reloadedIndex.Artifacts {
		if artifact.Path == asset && artifact.Fingerprint != changed {
			t.Fatalf("a new Load did not observe the new bytes: %+v", artifact)
		}
	}
}
