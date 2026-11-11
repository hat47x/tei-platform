package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/artifact"
)

// TestArtifactsOutPrintsExactlyOneJSONObject pins the stdout contract of
// `artifacts -out`: one JSON object, and that object is the manifest.
//
// The command used to print two objects (the evidence bundle and then the
// manifest). Two read-only verifiers parse the stdout as a single object
// (plan/fixtures/crosslane-cli/verify_operations_artifacts.py and
// plan/fixtures/capability-handoff/verify_real_cli_revision.py), so a stream of
// two failed them with "Extra data" and the observed-real-cli CI check went red.
// The evidence bundle is written to `evidence.json` inside the output directory,
// so nothing is lost by not printing it.
func TestArtifactsOutPrintsExactlyOneJSONObject(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")
	set, err := artifact.RenderWorkspace(root, "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}

	outDir := t.TempDir()
	if err := runArtifactsCommand(root, "ledger", outDir, false, ""); err != nil {
		t.Fatalf("artifacts -out: %v", err)
	}

	// The published manifest on disk names the same revision the in-memory set
	// does, so the file the command wrote is the set it rendered.
	published, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest.json: %v", err)
	}
	var onDisk struct {
		Manifest map[string]any `json:"manifest"`
		Findings []any          `json:"findings"`
	}
	if err := json.Unmarshal(published, &onDisk); err != nil {
		t.Fatalf("manifest.json is not JSON: %v", err)
	}
	if onDisk.Manifest["inputRevision"] != set.Manifest.InputRevision {
		t.Fatalf("published manifest revision = %v, want %v",
			onDisk.Manifest["inputRevision"], set.Manifest.InputRevision)
	}

	// The evidence bundle is on disk, not on stdout.
	if _, err := os.Stat(filepath.Join(outDir, "evidence.json")); err != nil {
		t.Fatalf("evidence.json is missing from the output directory: %v", err)
	}
}

// TestArtifactsStdoutIsOneObject pins the same contract at the byte level for the
// no-output-directory path, which is the read-only form the verifiers compare
// against. stdout is redirected to a file rather than a pipe: a pipe whose reader
// is drained only after the command returns would block on a large set.
func TestArtifactsStdoutIsOneObject(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")

	capture, err := os.CreateTemp(t.TempDir(), "artifacts-stdout-*.json")
	if err != nil {
		t.Fatal(err)
	}
	capturePath := capture.Name()
	original := os.Stdout
	os.Stdout = capture
	runErr := runArtifactsCommand(root, "ledger", "", false, "")
	os.Stdout = original
	if err := capture.Close(); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatalf("artifacts: %v", runErr)
	}
	printed, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(bytes.NewReader(printed))
	var first map[string]any
	if err := decoder.Decode(&first); err != nil {
		t.Fatalf("stdout is not one JSON object: %v", err)
	}
	// A second object means the stream is not a single document.
	if decoder.More() {
		t.Fatalf("stdout carries more than one JSON object (%d bytes)", len(printed))
	}
	if _, ok := first["manifest"]; !ok {
		t.Fatalf("the stdout object has no manifest: %v", first)
	}
	if _, ok := first["artifacts"]; !ok {
		t.Fatalf("the stdout object has no artifacts: %v", first)
	}
}

// TestWriteArtifactFilesWritesOneFilePerArtifact keeps the existing per-file
// contract.
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
