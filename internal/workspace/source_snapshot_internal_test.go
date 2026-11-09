package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedReaderRejectsExternalSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "private.tei")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "alias.tei")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	physicalRoot, err := canonicalWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readWorkspaceFile(physicalRoot, "alias.tei"); err == nil {
		t.Fatalf("external source was accepted: %v", err)
	}
}

func TestCheckedReaderAllowsLocalSymlinkToRegularFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.tei"), []byte("trusted bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.tei", filepath.Join(root, "alias.tei")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	physicalRoot, err := canonicalWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readWorkspaceFile(physicalRoot, "alias.tei")
	if err != nil || string(got) != "trusted bytes" {
		t.Fatalf("in-root symlink was not resolved safely: %q, %v", got, err)
	}
}

func TestCheckedReaderPreservesMissingPathError(t *testing.T) {
	root, err := canonicalWorkspaceRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = readWorkspaceFile(root, "migration-provider.tei")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing optional source lost os.ErrNotExist: %v", err)
	}
}
