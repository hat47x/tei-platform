package workspace

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPinnedSourceRootDoesNotFollowWorkspaceDirectoryReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows may disallow renaming an open directory; test symlink alias pivot instead")
	}
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

func TestReadSourceForLoadFallsBackToCheckedRootForDirectLoader(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "model.tei")
	if err := os.WriteFile(source, []byte("model"), 0600); err != nil {
		t.Fatal(err)
	}
	loaded := Result{sourceSnapshotRoot: root, sourceSnapshot: map[string][]byte{}}
	got, err := readSourceForLoad(root, &loaded, source)
	if err != nil || string(got) != "model" {
		t.Fatalf("direct loader did not use safe rooted fallback: %q, %v", got, err)
	}
	if string(loaded.sourceSnapshot["model.tei"]) != "model" {
		t.Fatalf("direct loader lost its Source snapshot: %+v", loaded.sourceSnapshot)
	}
	outside := filepath.Join(t.TempDir(), "outside.tei")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias.tei")
	if err := os.Symlink(outside, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := readSourceForLoad(root, &loaded, alias); err == nil {
		t.Fatal("direct loader's fallback followed an escaping symlink")
	}
	if _, err := readSourceForLoad(root, &Result{}, source); err == nil ||
		!strings.Contains(err.Error(), "no Workspace root") {
		t.Fatalf("loader without any Workspace boundary was accepted: %v", err)
	}
}

func TestPinnedWorkspaceRootSurvivesSymlinkAliasRetarget(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOOS == "wasip1" || runtime.GOOS == "plan9" {
		t.Skip("this target deliberately has no secure rooted reader")
	}
	parent := t.TempDir()
	before := filepath.Join(parent, "before")
	after := filepath.Join(parent, "after")
	for _, dir := range []string{before, after} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(before, "example.tei"), []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(after, "example.tei"), []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "current")
	if err := os.Symlink(before, alias); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	pinned, err := openWorkspaceRoot(alias)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(after, alias); err != nil {
		t.Fatal(err)
	}
	stillOriginal, err := readPinnedWorkspaceFile(pinned, "example.tei")
	if err != nil || string(stillOriginal) != "before" {
		t.Fatalf("pinned root followed modified Workspace alias: %q, %v", stillOriginal, err)
	}
	newAlias, err := readWorkspaceFile(alias, "example.tei")
	if err != nil || string(newAlias) != "after" {
		t.Fatalf("fixture did not redirect alias: %q, %v", newAlias, err)
	}
}
