package main

import (
	"os"
	"path/filepath"
	"testing"
)

// stateDir returns a temporary directory for an isolated Go module cache.
//
// t.TempDir() cannot be used here: the Go toolchain marks extracted module-cache
// files read-only (0444), so the automatic RemoveAll fails half-way and reports a
// spurious test failure. This helper registers its own cleanup that restores
// write permission before removing. A leftover directory is still a test failure,
// so the helper does not silently ignore errors.
func stateDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tei-supply-state-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeWritable(dir); err != nil {
			t.Errorf("remove isolated state %s: %v", dir, err)
		}
	})
	return dir
}

func removeWritable(root string) error {
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return os.Chmod(path, 0o600)
	}); err != nil {
		return err
	}
	return os.RemoveAll(root)
}
