package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDomainSourceListingUsesPinnedRootAfterWorkspaceAliasReplacement(t *testing.T) {
	parent := t.TempDir()
	workspacePath := filepath.Join(parent, "workspace")
	if err := os.MkdirAll(filepath.Join(workspacePath, "semantic"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspacePath, "semantic", "inside.tei"), []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	pinned, err := os.OpenRoot(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()
	loaded := Result{sourceReadRoot: pinned}
	moved := filepath.Join(parent, "original")
	if err := os.Rename(workspacePath, moved); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(parent, "foreign")
	if err := os.MkdirAll(filepath.Join(foreign, "semantic"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "semantic", "outside.tei"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, workspacePath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	files, err := listPinnedDomainSources(workspacePath, &loaded, "semantic", func(name string) bool {
		return strings.HasSuffix(name, ".tei")
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(workspacePath, "semantic", "inside.tei")}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("enumeration escaped pinned root: got %v, want %v", files, want)
	}
	if _, err := readPinnedWorkspaceFile(pinned, "semantic/inside.tei"); err != nil {
		t.Fatalf("pinned source was lost after alias changed: %v", err)
	}
}

func TestDomainSourceListingRejectsExternalSymlinkAndSupportsAbsentRoots(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "plugins"), 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.tei")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "plugins", "plugin.tei")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	pinned, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()
	loaded := Result{sourceReadRoot: pinned}
	if _, err := listPinnedDomainSources(root, &loaded, "plugins", func(name string) bool {
		return name == "plugin.tei"
	}); err == nil {
		t.Fatal("external plugin symlink escaped the pinned root")
	}
	files, err := listPinnedDomainSources(root, &loaded, "bundles", func(name string) bool {
		return name == "bundle.tei"
	})
	if err != nil || len(files) != 0 {
		t.Fatalf("missing optional bundle root: %v, %v", files, err)
	}
}
