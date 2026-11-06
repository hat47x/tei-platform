package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteLineageWritesInsideTheRequestedPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.json")
	if err := writeLineage(path, []byte("{}"+"\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "{}\n" {
		t.Fatalf("read: %q err=%v", content, err)
	}
	if runtime.GOOS != "windows" {
		// Windows reports a synthesised mode, so the check is meaningful only
		// where the permission bits are real.
		if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("the observation must stay private: %v %v", info, err)
		}
	}
}

func TestWriteLineageRejectsAnEmptyPath(t *testing.T) {
	if err := writeLineage("", []byte("{}")); err == nil {
		t.Fatal("an empty path must be refused before anything is written")
	}
}
