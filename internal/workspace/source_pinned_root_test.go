package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinnedSourceRootDoesNotFollowWorkspaceDirectoryReplacement(t *testing.T) {
	parent := t.TempDir()
	current := filepath.Join(parent, "workspace")
	if err := os.Mkdir(current, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "model.tei"), []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := openWorkspaceRoot(current)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	renamed := filepath.Join(parent, "previous-workspace")
	if err := os.Rename(current, renamed); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(current, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "model.tei"), []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readPinnedWorkspaceFile(root, "model.tei")
	if err != nil || string(got) != "before" {
		t.Fatalf("a pinned Source read followed replaced Workspace: %q, %v", got, err)
	}
	newRoot, err := readWorkspaceFile(current, "model.tei")
	if err != nil || string(newRoot) != "after" {
		t.Fatalf("fixture did not replace the path: %q, %v", newRoot, err)
	}
}

func TestLoadRejectsManifestSymlinkLeavingWorkspace(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(manifest, []byte("workspace:\n  sources:\n    - semantic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(manifest, filepath.Join(root, "tei.workspace.yaml")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("Load read a Workspace manifest through an escaping symlink")
	}
}

func TestLoadAcceptsInternalManifestLinkAndReleasesRootHandle(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "inside.yaml"), []byte("workspace:\n  sources:\n    - semantic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("inside.yaml", filepath.Join(root, "tei.workspace.yaml")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.sourceReadRoot != nil {
		t.Fatal("Load Result retained a live OS root handle")
	}
	if loaded.sourceSnapshotRoot == "" || loaded.sourceSnapshot == nil {
		t.Fatalf("Load discarded source provenance: %+v", loaded)
	}
}

func TestReadSourceForLoadRequiresActiveRoot(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "model.tei")
	if err := os.WriteFile(source, []byte("model"), 0600); err != nil {
		t.Fatal(err)
	}
	loaded := Result{sourceSnapshotRoot: root, sourceSnapshot: map[string][]byte{}}
	if _, err := readSourceForLoad(root, &loaded, source); err == nil ||
		!strings.Contains(err.Error(), "active Workspace root") {
		t.Fatalf("unbound loader silently reread by path: %v", err)
	}
}
