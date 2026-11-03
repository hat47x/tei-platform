package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/artifact"
)

func TestWriteArtifactFilesWritesOneFilePerArtifact(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")
	set, err := artifact.RenderWorkspace(root, "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}

	outDir := t.TempDir()
	if err := writeArtifactFiles(outDir, set); err != nil {
		t.Fatalf("write artifacts: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(set.Artifacts)+1 {
		t.Fatalf("expected one file per artifact plus the manifest: got %d, want %d", len(entries), len(set.Artifacts)+1)
	}
	if _, err := os.Stat(filepath.Join(outDir, "manifest.json")); err != nil {
		t.Fatalf("manifest.json is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "operation-surface.json")); err != nil {
		t.Fatalf("the surface file is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "table-public-orders.md")); err != nil {
		t.Fatalf("the table file is missing: %v", err)
	}
}
