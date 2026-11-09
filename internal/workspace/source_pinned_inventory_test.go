package workspace

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestManagedInventoryStaysWithPinnedRootAfterWorkspaceAliasRetarget(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOOS == "wasip1" || runtime.GOOS == "plan9" {
		t.Skip("no traversal-resistant Workspace root on this target")
	}
	parent := t.TempDir()
	before := filepath.Join(parent, "before")
	after := filepath.Join(parent, "after")
	for _, dir := range []string{filepath.Join(before, "assets"), filepath.Join(after, "assets")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(before, "assets", "original.md"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(after, "assets", "foreign.md"), []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "current")
	if err := os.Symlink(before, alias); err != nil {
		t.Skipf("directory aliases unavailable: %v", err)
	}
	rootHandle, err := openWorkspaceRoot(alias)
	if err != nil {
		t.Fatal(err)
	}
	defer rootHandle.Close()
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(after, alias); err != nil {
		t.Fatal(err)
	}
	managed, err := scanManagedFiles(rootHandle, []string{"assets"})
	if err != nil {
		t.Fatal(err)
	}
	if len(managed) != 1 || managed[0].Path != "assets/original.md" {
		t.Fatalf("managed inventory changed to replacement Workspace: %+v", managed)
	}
	got, err := readPinnedWorkspaceFile(rootHandle, managed[0].Path)
	if err != nil || string(got) != "original" {
		t.Fatalf("inventory and parsed source used different roots: %q, %v", got, err)
	}
}

func TestManagedInventoryRejectsAliasesWithSamePhysicalIdentity(t *testing.T) {
	for _, aliasKind := range []string{"symlink", "hardlink"} {
		t.Run(aliasKind, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "assets")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			original := filepath.Join(dir, "original.md")
			alias := filepath.Join(dir, "alias.md")
			if err := os.WriteFile(original, []byte("same file"), 0600); err != nil {
				t.Fatal(err)
			}
			var err error
			if aliasKind == "symlink" {
				err = os.Symlink("original.md", alias)
			} else {
				err = os.Link(original, alias)
			}
			if err != nil {
				t.Skipf("%s unavailable: %v", aliasKind, err)
			}
			rootHandle, err := openWorkspaceRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			defer rootHandle.Close()
			if _, err := scanManagedFiles(rootHandle, []string{"assets"}); err == nil ||
				!strings.Contains(err.Error(), "same physical file") {
				t.Fatalf("duplicate physical Source was accepted: %v", err)
			}
		})
	}
}

func TestManagedInventoryAllowsMissingRootAndRejectsEscapingEntries(t *testing.T) {
	root := t.TempDir()
	handle, err := openWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	empty, err := scanManagedFiles(handle, []string{"unused/nested"})
	if err != nil || len(empty) != 0 {
		t.Fatalf("missing optional root was not empty: %+v, %v", empty, err)
	}
	if err := os.Mkdir(filepath.Join(root, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "private.md")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "assets", "escape.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := scanManagedFiles(handle, []string{"assets"}); err == nil ||
		!strings.Contains(err.Error(), "outside the Workspace") {
		t.Fatalf("escaping Source was inventoried: %v", err)
	}
}

func TestManagedInventoryRejectsAbsoluteSymlinkWithinWorkspace(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "assets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(dir, "original.md")
	if err := os.WriteFile(original, []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(original, filepath.Join(dir, "alias.md")); err != nil {
		t.Skipf("absolute symlink unavailable: %v", err)
	}
	rootHandle, err := openWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer rootHandle.Close()
	if _, err := scanManagedFiles(rootHandle, []string{"assets"}); err == nil ||
		!strings.Contains(err.Error(), "outside the Workspace") {
		t.Fatalf("absolute source link was not rejected by rooted inventory: %v", err)
	}
}
