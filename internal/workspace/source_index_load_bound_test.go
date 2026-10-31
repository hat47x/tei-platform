package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

const loadBoundReadme = "databases/sales-core/README.md"

func copyReferenceWorkspace(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.CopyFS(root, os.DirFS(filepath.Join("..", "..", "reference", "workspace"))); err != nil {
		t.Fatalf("copy reference Workspace: %v", err)
	}
	return root
}

func artifactFor(t *testing.T, idx sourceindex.Index, path string) sourceindex.Artifact {
	t.Helper()
	for _, a := range idx.Artifacts {
		if a.Path == path {
			return a
		}
	}
	t.Fatalf("Artifact %s missing from Source Index", path)
	return sourceindex.Artifact{}
}

func mustArtifact(t *testing.T, path string, content []byte) sourceindex.Artifact {
	t.Helper()
	a, err := sourceindex.NewArtifact(path, content)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// (a) Every ManagedFile is captured, with the same path form as the snapshot.
func TestLoadCapturesEveryManagedFile(t *testing.T) {
	root := copyReferenceWorkspace(t)
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ManagedFiles) == 0 {
		t.Fatal("no managed files")
	}
	for _, m := range result.ManagedFiles {
		_, inSnap := result.sourceSnapshot[m.Path]
		_, inExtra := result.sourceManagedExtra[m.Path]
		if !inSnap && !inExtra {
			t.Errorf("managed file %s is in neither Load-time map", m.Path)
		}
		if inSnap && inExtra {
			t.Errorf("managed file %s is in both Load-time maps", m.Path)
		}
	}
	if _, ok := result.sourceManagedExtra[loadBoundReadme]; !ok {
		t.Fatalf("%s should be captured as an unparsed managed file", loadBoundReadme)
	}
}

// (b) A file rewritten between Load and index build must not change the
// Artifact fingerprint of the strict builder. The legacy builder re-reads the
// file (the pre-fix behaviour); that is pinned so the RED condition is visible.
func TestBuildSourceIndexFromLoadIgnoresLaterFileChange(t *testing.T) {
	root := copyReferenceWorkspace(t)
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(loadBoundReadme))
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := mustArtifact(t, loadBoundReadme, original)
	if err := os.WriteFile(path, []byte("tampered after Load\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	legacy, err := BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(artifactFor(t, legacy, loadBoundReadme), want) {
		t.Fatal("legacy BuildSourceIndex no longer re-reads unparsed files; its behaviour must stay unchanged")
	}

	strict, err := BuildSourceIndexFromLoad(root, result)
	if err != nil {
		t.Fatal(err)
	}
	if got := artifactFor(t, strict, loadBoundReadme); !reflect.DeepEqual(got, want) {
		t.Fatalf("Artifact is not bound to Load-time bytes: got %+v want %+v", got, want)
	}
}

// (c) Unknown-root asset: bytes captured at Load, not re-read.
func TestBuildSourceIndexFromLoadUnknownRootAsset(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	dir := filepath.Join(root, "custom-assets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: demo\nsources:\n  - custom-assets\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "asset.md")
	if err := os.WriteFile(file, []byte("native asset"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("changed after Load"), 0o600); err != nil {
		t.Fatal(err)
	}
	idx, err := BuildSourceIndexFromLoad(root, result)
	if err != nil {
		t.Fatal(err)
	}
	want := mustArtifact(t, "custom-assets/asset.md", []byte("native asset"))
	if got := artifactFor(t, idx, "custom-assets/asset.md"); !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown-root asset not bound to Load-time bytes: %+v", got)
	}
}

// IndexedLoadedSnapshot must stay parser-bound (not loosened by the capture).
func TestIndexedLoadedSnapshotStillRejectsUnparsedAssets(t *testing.T) {
	root := copyReferenceWorkspace(t)
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := IndexedLoadedSnapshot(root, result); err == nil || !strings.Contains(err.Error(), "has no parsed Load snapshot") {
		t.Fatalf("IndexedLoadedSnapshot must stay parser-bound: %v", err)
	}
}

// (d) Fail closed without touching the filesystem.
func TestBuildSourceIndexFromLoadFailsClosed(t *testing.T) {
	root := copyReferenceWorkspace(t)
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("no snapshot", func(t *testing.T) {
		bare := Result{ManagedFiles: result.ManagedFiles}
		if _, err := BuildSourceIndexFromLoad(root, bare); err == nil || !strings.Contains(err.Error(), "not produced by Load") {
			t.Fatalf("Result without Load bytes was accepted: %v", err)
		}
		// Legacy builder keeps working for hand-built Results.
		if _, err := BuildSourceIndex(root, Result{}); err != nil {
			t.Fatalf("legacy builder regressed: %v", err)
		}
	})

	t.Run("wrong root", func(t *testing.T) {
		other := copyReferenceWorkspace(t)
		if _, err := BuildSourceIndexFromLoad(other, result); err == nil || !strings.Contains(err.Error(), "another Workspace") {
			t.Fatalf("Result for another root was accepted: %v", err)
		}
	})

	t.Run("path outside both maps", func(t *testing.T) {
		// The file exists on disk, so only the strict rule can refuse it.
		const rel = "databases/sales-core/not-managed.md"
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		forged := result
		forged.ManagedFiles = append(append([]ManagedFile(nil), result.ManagedFiles...),
			ManagedFile{Path: rel, SourceRoot: "databases"})
		if _, err := BuildSourceIndexFromLoad(root, forged); err == nil || !strings.Contains(err.Error(), rel) ||
			!strings.Contains(err.Error(), "not covered by the Load-time bytes") {
			t.Fatalf("uncovered managed path was accepted: %v", err)
		}
		forged = result
		forged.Sources = map[string]SourceLocation{"x:y": {File: rel, Line: 1, Column: 1}}
		if _, err := BuildSourceIndexFromLoad(root, forged); err == nil || !strings.Contains(err.Error(), "not covered by the Load-time bytes") {
			t.Fatalf("uncovered Source reference was accepted: %v", err)
		}
	})
}
