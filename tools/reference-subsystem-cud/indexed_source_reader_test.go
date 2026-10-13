package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexedSourceReaderRejectsSymlinkEscapeButPreservesInsideReferences(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	insidePath := filepath.Join(root, "source", "inside.json")
	if err := os.MkdirAll(filepath.Dir(insidePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(insidePath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	externalPath := filepath.Join(outside, "sensitive.json")
	if err := os.WriteFile(externalPath, []byte("{\"private\":true}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(insidePath, filepath.Join(root, "local-link.json")); err != nil {
		t.Skipf("filesystem does not support symlink tests: %v", err)
	}
	if err := os.Symlink(externalPath, filepath.Join(root, "external-link.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "external-dir")); err != nil {
		t.Fatal(err)
	}
	reader, err := indexedSourceReader(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"source/inside.json", "local-link.json"} {
		got, err := reader(path)
		if err != nil || !bytes.Equal(got, []byte("{}")) {
			t.Fatalf("read legitimate indexed source %q: %q %v", path, got, err)
		}
	}
	for _, path := range []string{"external-link.json", "external-dir/sensitive.json"} {
		if _, err := reader(path); err == nil || !strings.Contains(err.Error(), "outside the source root") {
			t.Fatalf("indexed path %q must reject symlink escape: %v", path, err)
		}
	}
	for _, path := range []string{"../sensitive.json", "/tmp/sensitive.json", "source/../source/inside.json", "external-dir/../source/inside.json"} {
		if _, err := reader(path); err == nil {
			t.Fatalf("indexed path %q must reject lexical ambiguity or escape", path)
		}
	}
}

func TestIndexedSourceReaderRejectsUnusableRoot(t *testing.T) {
	root := t.TempDir()
	if _, err := indexedSourceReader(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing source root accepted")
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := indexedSourceReader(file); err == nil {
		t.Fatal("regular file accepted as source root")
	}
}

func TestIndexedProcedureCannotLocateSourceThroughOutsideSymlink(t *testing.T) {
	world, procedure, facts := actualProcedureCandidate(t)
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "procedures"), 0o755); err != nil {
		t.Fatal(err)
	}
	externalPath := filepath.Join(outside, "order-handling.json")
	if err := os.WriteFile(externalPath, procedure, 0o600); err != nil {
		t.Fatal(err)
	}
	path := "procedures/order-handling.json"
	if err := os.Symlink(externalPath, filepath.Join(root, filepath.FromSlash(path))); err != nil {
		t.Skipf("filesystem does not support symlinks: %v", err)
	}
	idx := indexForActualProcedure(t, path, procedure)
	reader, err := indexedSourceReader(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectIndexedProcedureSnapshot(world, procedure, mustJSON(t, facts), idx, path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "unresolved" || report.IndexedProcedure != nil ||
		codeCount(report, "PROCEDURE_SOURCE_INDEX_INVALID") != 1 {
		t.Fatalf("matching external bytes must not validate indexed Procedure: %+v", report)
	}
}
