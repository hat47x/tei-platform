package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

func sourceRootFixture(t *testing.T, rootName string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: demo\nsources:\n  - " + rootName + "\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestWorkspaceRejectsNonlocalSourceRoots(t *testing.T) {
	for _, sourceRoot := range []string{
		"../", "../outside", "a/../../outside", "/tmp/outside", ".",
		"./custom", "custom//nested", "custom/../semantic", "custom/",
		"..\\outside", "C:\\outside", "C:/outside",
	} {
		t.Run(sourceRoot, func(t *testing.T) {
			root := sourceRootFixture(t, sourceRoot)
			if _, err := workspace.Load(root); err == nil {
				t.Fatalf("nonlocal or ambiguous source root %q was accepted", sourceRoot)
			} else if !strings.Contains(err.Error(), "source root") {
				t.Fatalf("missing source-root diagnostic: %v", err)
			}
		})
	}
}

func TestLegacyWorkspaceManifestRejectsEscapingRoot(t *testing.T) {
	root := sourceRootFixture(t, "custom")
	if err := os.Remove(filepath.Join(root, "tei.workspace.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.tei"), []byte("workspace demo { source ../ }"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Load(root); err == nil {
		t.Fatal("legacy manifest bypassed source-root containment")
	}
}

func TestWorkspaceRejectsDuplicateAndNestedSourceRoots(t *testing.T) {
	for _, tc := range []struct {
		name, first, second string
	}{
		{"duplicate", "semantic", "semantic"},
		{"parent-first", "custom", "custom/objects"},
		{"child-first", "custom/objects", "custom"},
	} {
		for _, legacy := range []bool{false, true} {
			mode := "yaml"
			if legacy {
				mode = "tei"
			}
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				root := sourceRootFixture(t, tc.first)
				if legacy {
					if err := os.Remove(filepath.Join(root, "tei.workspace.yaml")); err != nil {
						t.Fatal(err)
					}
					source := "workspace demo { source " + tc.first + " source " + tc.second + " }"
					if err := os.WriteFile(filepath.Join(root, "tei.workspace.tei"), []byte(source), 0600); err != nil {
						t.Fatal(err)
					}
				} else {
					manifest := "workspace: demo\nsources:\n  - " + tc.first + "\n  - " + tc.second + "\n"
					if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := workspace.Load(root); err == nil || !strings.Contains(err.Error(), "overlap") {
					t.Fatalf("overlapping source roots were accepted: %v", err)
				}
			})
		}
	}
}

func TestWorkspaceAllowsSimilarButSeparateSourceRoots(t *testing.T) {
	root := sourceRootFixture(t, "custom/assets")
	manifest := "workspace: demo\nsources:\n  - custom/assets\n  - custom/assets-v2\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Load(root); err != nil {
		t.Fatalf("sibling source roots must be independent: %v", err)
	}
}

func TestUnknownLocalRootRemainsManaged(t *testing.T) {
	root := sourceRootFixture(t, "custom-assets")
	dir := filepath.Join(root, "custom-assets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "asset.md"), []byte("native asset"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ManagedFiles) != 1 || result.ManagedFiles[0].Path != "custom-assets/asset.md" {
		t.Fatalf("forward-compatible local source root lost: %+v", result.ManagedFiles)
	}
	if _, err := workspace.BuildSourceIndex(root, result); err != nil {
		t.Fatalf("local native artifact should be indexable: %v", err)
	}
}

func TestNestedPortableSourceRootIsValid(t *testing.T) {
	root := sourceRootFixture(t, "custom/assets")
	dir := filepath.Join(root, "custom", "assets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "native.txt"), []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ManagedFiles) != 1 || result.ManagedFiles[0].Path != "custom/assets/native.txt" {
		t.Fatalf("nested Workspace-relative source not managed: %+v", result.ManagedFiles)
	}
}

func TestMissingOptionalSourceRootDoesNotFail(t *testing.T) {
	root := sourceRootFixture(t, "custom-not-materialized")
	result, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ManagedFiles) != 0 {
		t.Fatalf("missing source root yielded files: %+v", result.ManagedFiles)
	}
}

func TestSymlinkedSourceCannotEscapeWorkspace(t *testing.T) {
	root := sourceRootFixture(t, "custom-assets")
	dir := filepath.Join(root, "custom-assets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("not managed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "alias.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := workspace.Load(root); err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("escaped source target was not rejected: %v", err)
	}
}

func TestSymlinkedSourceRootOutsideWorkspaceFailsClosed(t *testing.T) {
	root := sourceRootFixture(t, "custom-assets")
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "custom-assets")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := workspace.Load(root); err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("symlinked source root escaped Workspace: %v", err)
	}
}

func TestWorkspaceContainmentRejectsSiblingPathPrefix(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.MkdirAll(filepath.Join(root, "custom-assets"), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: demo\nsources:\n  - custom-assets\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "workspace-other")
	if err := os.MkdirAll(outside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "private.txt"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "private.txt"), filepath.Join(root, "custom-assets", "alias.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := workspace.Load(root); err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("similar Workspace path prefix bypassed containment: %v", err)
	}
}

func TestKnownSemanticRootRejectsEscapedFileBeforeParsing(t *testing.T) {
	root := sourceRootFixture(t, "semantic")
	dir := filepath.Join(root, "semantic")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "foreign.tei")
	if err := os.WriteFile(outside, []byte("invalid semantic source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "foreign.tei")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := workspace.Load(root); err == nil || !strings.Contains(err.Error(), "outside Workspace") {
		t.Fatalf("semantic parser read outside source before containment preflight: %v", err)
	}
}

func TestBrokenManagedSymlinkFailsClosed(t *testing.T) {
	root := sourceRootFixture(t, "custom-assets")
	dir := filepath.Join(root, "custom-assets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-target.txt", filepath.Join(dir, "alias.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := workspace.Load(root); err == nil {
		t.Fatal("broken source symlink was silently omitted")
	}
}
