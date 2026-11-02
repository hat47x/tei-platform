package workspace

import (
	"os"
	"path/filepath"
	"strings"
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
// parser consumes it. Its bytes are captured separately at Load time so Source
// Index does not re-read the file after Load. sourceSnapshot keeps only parsed
// bytes, so IndexedLoadedSnapshot remains parser-bound and fails closed here.
func TestLoadCapturesForwardCompatibleManagedAsset(t *testing.T) {
	root, asset := snapshotFixture(t)
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.sourceSnapshot[asset]; ok {
		t.Fatalf("unparsed asset must not enter the parser-bound snapshot: %+v", loaded.sourceSnapshot)
	}
	content, ok := loaded.sourceManagedExtra[asset]
	if !ok {
		t.Fatalf("managed asset is absent from the Load-time managed capture: %+v", loaded.ManagedFiles)
	}
	if string(content) != "original bytes\n" {
		t.Fatalf("captured content=%q", content)
	}
	if _, ok := loaded.sourceManagedExtra["semantic/model.tei"]; ok {
		t.Fatal("parsed Source must stay in sourceSnapshot, not sourceManagedExtra")
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
	if _, _, err := IndexedLoadedSnapshot(root, loaded); err == nil ||
		!strings.Contains(err.Error(), "has no parsed Load snapshot") {
		t.Fatalf("IndexedLoadedSnapshot must stay parser-bound and fail closed, got %v", err)
	}
}

// The capture is a Load-time authoring record, not the current filesystem.
// Modifying the asset after Load must not change the bound artifact, while a
// new Load observes the new bytes.
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
		if artifact.Path == asset && artifact.Fingerprint != original {
			t.Fatalf("Source Index re-read the file after Load: %+v", artifact)
		}
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
