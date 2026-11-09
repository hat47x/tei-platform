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
	// Test-only shape oracle. It does not add a new runtime response validator.
	// The first-byte check rejects null, which json.Unmarshal into a Go slice
	// otherwise accepts as a valid nil slice.
	arrayResult := func(raw json.RawMessage) bool {
		var items []json.RawMessage
		return len(raw) > 0 && raw[0] == '[' && json.Unmarshal(raw, &items) == nil
	}
	if !arrayResult(reply.Result) {
		t.Fatalf("advertised array, actual result has different shape: %q", reply.Result)
	}
	for name, fixture := range map[string]json.RawMessage{
		"object":  json.RawMessage(`{"notAnArray":true}`),
		"null":    json.RawMessage(`null`),
		"invalid": json.RawMessage(`[{"missing":`),
	} {
		if arrayResult(fixture) {
			t.Errorf("%s fixture incorrectly satisfies resultShape=array", name)
		}
	}
}
