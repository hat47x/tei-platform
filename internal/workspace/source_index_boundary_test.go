package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// BuildSourceIndex is public to callers within the Go module. A forged Result
// must not bypass Workspace physical containment merely because its path is
// lexically relative.
func TestSourceIndexRejectsExternalSymlinkFromForgedResult(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("foreign source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "alias.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := workspace.BuildSourceIndex(root, workspace.Result{
		ManagedFiles: []workspace.ManagedFile{{Path: "alias.txt", SourceRoot: "custom"}},
	})
	if err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("forged managed source escaped Workspace: %v", err)
	}
}

func TestSourceIndexRejectsExternalSymlinkInSourceMap(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.tei")
	if err := os.WriteFile(outside, []byte("foreign source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "alias.tei")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := workspace.BuildSourceIndex(root, workspace.Result{
		Sources: map[string]workspace.SourceLocation{
			"application:demo": {File: "alias.tei", Line: 1, Column: 1},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("forged Source map escaped Workspace: %v", err)
	}
}

func TestSourceIndexRejectsExternalSymlinkViaNestedDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "asset.md"), []byte("foreign source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "custom")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := workspace.BuildSourceIndex(root, workspace.Result{
		ManagedFiles: []workspace.ManagedFile{{Path: "custom/asset.md", SourceRoot: "custom"}},
	})
	if err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("nested symlink escaped Workspace: %v", err)
	}
}

func TestSourceIndexAcceptsNativeLocalManagedFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "native.md"), []byte("local source"), 0600); err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, workspace.Result{
		ManagedFiles: []workspace.ManagedFile{{Path: "native.md", SourceRoot: "custom"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Artifacts) != 1 || index.Artifacts[0].Path != "native.md" {
		t.Fatalf("local native artifact was not indexed: %+v", index.Artifacts)
	}
}

func TestSourceIndexRejectsExternalSourceContribution(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "foreign.txt")
	if err := os.WriteFile(outside, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "contribution.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := workspace.BuildSourceIndex(root, workspace.Result{
		SourceContributions: []workspace.SourceContribution{{
			File: "contribution.txt",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("source contribution escaped Workspace: %v", err)
	}
}

func TestSourceIndexRejectsSiblingWorkspacePrefix(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	outside := filepath.Join(parent, "workspace-other")
	for _, dir := range []string{root, outside} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "private.md"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "private.md"), filepath.Join(root, "alias.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := workspace.BuildSourceIndex(root, workspace.Result{
		ManagedFiles: []workspace.ManagedFile{{Path: "alias.md", SourceRoot: "custom"}},
	})
	if err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("sibling path prefix escaped Workspace: %v", err)
	}
}

func TestSourceIndexAllowsLocalSymlinkToRegularFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "native.md"), []byte("local"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("native.md", filepath.Join(root, "alias.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	index, err := workspace.BuildSourceIndex(root, workspace.Result{
		ManagedFiles: []workspace.ManagedFile{{Path: "alias.md", SourceRoot: "custom"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Artifacts) != 1 || index.Artifacts[0].Path != "alias.md" {
		t.Fatalf("local symlink artifact not preserved: %+v", index.Artifacts)
	}
}

func TestSourceIndexRejectsBrokenSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink("missing.md", filepath.Join(root, "alias.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := workspace.BuildSourceIndex(root, workspace.Result{
		ManagedFiles: []workspace.ManagedFile{{Path: "alias.md", SourceRoot: "custom"}},
	}); err == nil {
		t.Fatal("broken symlink was silently accepted")
	}
}

func TestSourceIndexRejectsNonregularSource(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "folder"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := workspace.BuildSourceIndex(root, workspace.Result{
		ManagedFiles: []workspace.ManagedFile{{Path: "folder", SourceRoot: "custom"}},
	})
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory was accepted as source artifact: %v", err)
	}
}
