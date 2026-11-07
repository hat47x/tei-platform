package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestRequirementsResultShapeDiscoveryChecksActualCLI exercises the built tei
// process instead of comparing two hand-written metadata constants (#3347).
// A successful JSON envelope does not imply its result is an object.
func TestRequirementsResultShapeDiscoveryChecksActualCLI(t *testing.T) {
	record, ok := discoveryRecord("requirements")
	if !ok || record.JSONContract != "enveloped" || record.ResultShape != "array" {
		t.Fatalf("requirements must advertise the exceptional result array: %+v", record)
	}
	for _, other := range commandCatalog().Commands {
		if other.Name != "requirements" && other.ResultShape != "" {
			t.Fatalf("%s advertised an unverified result shape %q", other.Name, other.ResultShape)
		}
	}

	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if raw, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build actual CLI: %v\n%s", err, raw)
	}

	workspaceRoot := filepath.Join("..", "..", "reference", "workspace")
	raw, err := exec.Command(binary, "requirements", "-workspace", workspaceRoot).CombinedOutput()
	if err != nil {
		t.Fatalf("actual requirements command failed: %v\n%s", err, raw)
	}
	var reply struct {
		SchemaVersion string          `json:"schemaVersion"`
		OK            bool            `json:"ok"`
		Result        json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		t.Fatalf("actual requirements response is not an envelope: %v\n%s", err, raw)
	}
	if !reply.OK || reply.SchemaVersion != record.SchemaVersion {
		t.Fatalf("actual response mismatch: ok=%v version=%q advertised=%q", reply.OK, reply.SchemaVersion, record.SchemaVersion)
	}
	if len(reply.Result) == 0 || reply.Result[0] != '[' {
		t.Fatalf("advertised array, actual result starts with %q", reply.Result)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(reply.Result, &entries); err != nil {
		t.Fatalf("actual result is not a JSON array: %v", err)
	}

	// A valid JSON object would have passed a parse-only gate, yet it must
	// not satisfy the array-shaped result guarantee in the discovery record.
	fake := json.RawMessage(`{"notAnArray":true}`)
	if len(fake) == 0 || fake[0] == '[' {
		t.Fatal("negative fixture accidentally matches array result")
	}
}
