package workspace_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func TestReadSourceEnforcesPositiveLimitAndRejectsOversizedInput(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "model.xead")
	if err := os.WriteFile(path, []byte("1234"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := workspace.ReadSource(root, "model.xead", 4)
	if err != nil || string(got) != "1234" {
		t.Fatalf("ReadSource() = %q, %v; want exact-limit content", got, err)
	}

	if err := os.WriteFile(path, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = workspace.ReadSource(root, "model.xead", 4)
	if !errors.Is(err, workspace.ErrSourceTooLarge) || got != nil {
		t.Fatalf("oversized source = %q, %v; want no content and ErrSourceTooLarge", got, err)
	}
	for _, limit := range []int64{0, -1} {
		if _, err := workspace.ReadSource(root, "model.xead", limit); !errors.Is(err, workspace.ErrInvalidSourceReadLimit) {
			t.Fatalf("limit %d error = %v; want ErrInvalidSourceReadLimit", limit, err)
		}
	}
}

func TestReadSourceRejectsAbsoluteTraversalAndNonregularPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "model.xead"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "directory"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"../outside.xead",
		"directory/../model.xead",
		filepath.Join(root, "model.xead"),
		"C:/private/model.xead",
		"directory",
	} {
		if got, err := workspace.ReadSource(root, path, 64); err == nil {
			t.Fatalf("unsafe source path %q returned content %q", path, got)
		}
	}
}

func TestReadSourceRejectsEscapingSymlinksAndAllowsInternalRelativeSymlinks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "model.xead"), []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.xead")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.xead")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := os.Symlink("model.xead", filepath.Join(root, "inside-link.xead")); err != nil {
		t.Skipf("relative symlink creation unavailable: %v", err)
	}
	if got, err := workspace.ReadSource(root, "escape.xead", 64); err == nil {
		t.Fatalf("escaping symlink returned content %q", got)
	}
	got, err := workspace.ReadSource(root, "inside-link.xead", 64)
	if err != nil || string(got) != "inside" {
		t.Fatalf("in-root relative symlink = %q, %v", got, err)
	}
}
