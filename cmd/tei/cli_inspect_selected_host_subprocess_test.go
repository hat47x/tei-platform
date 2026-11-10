package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

// The in-process Helper tests do not execute cmd/tei/main.go.
// Run an actual Go CLI program so its normal dispatch, selected Host
// registration and typed Operation must all be present in one executable.
func TestNormalInspectExecutableDispatchSelectsReferenceProvider(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "inspect",
		"-workspace", root,
		"-subject-kind", "semantic-data",
		"-subject-id", "sales.order")
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("normal tei inspect executable failed: %v output=%s", err, raw)
	}
	var got cliInspectionReport
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("normal CLI did not return a JSON envelope: %v output=%s", err, raw)
	}
	if !got.OK || got.Result == nil || got.Result.Subject.ID != "sales.order" ||
		got.Result.WorldFingerprint == "" || got.Result.SourceIndexFingerprint == "" {
		t.Fatalf("normal executable Inspector response incomplete: %+v", got)
	}
}
