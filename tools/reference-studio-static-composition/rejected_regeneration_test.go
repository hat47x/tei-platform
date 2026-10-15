package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestPoCBRejectedRegenerationKeepsPublishedAssets checks the real two CLI
// entrypoints. Invalid selections must never replace a previously generated
// static source, its declaration, or the associated provenance files.
func TestPoCBRejectedRegenerationKeepsPublishedAssets(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("Node.js is required for the CLI integrity test: %v", err)
	}
	out := t.TempDir()
	run := func(command string, args ...string) ([]byte, error) {
		t.Helper()
		return exec.Command(command, args...).CombinedOutput()
	}
	for _, cmd := range []*exec.Cmd{
		exec.Command("go", "run", ".", "-fixture", "fixture.json", "-out", out),
		exec.Command(node, "compose-ui.mjs", "fixture.json", out),
	} {
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("baseline CLI generation failed: %v\n%s", err, output)
		}
	}

	// Read the real profile and change only the selected service version:
	// both Go and Node generators must independently reject this exact input.
	raw, err := os.ReadFile("fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var profile map[string]any
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatal(err)
	}
	services, ok := profile["services"].([]any)
	if !ok || len(services) != 2 {
		t.Fatal("unexpected fixed service fixture")
	}
	services[0].(map[string]any)["version"] = "9.9.9"
	bad, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	badFixture := filepath.Join(t.TempDir(), "invalid-selection.json")
	if err := os.WriteFile(badFixture, bad, 0600); err != nil {
		t.Fatal(err)
	}

	snapshot := func() map[string]string {
		t.Helper()
		files, err := os.ReadDir(out)
		if err != nil {
			t.Fatal(err)
		}
		expected := []string{
			"services.generated.go",
			"services.provenance.json",
			"ui.generated.d.mts",
			"ui.generated.mjs",
			"ui.provenance.json",
		}
		if len(files) != len(expected) {
			t.Fatalf("unexpected generated file count %d", len(files))
		}
		digests := make(map[string]string, len(files))
		for i, file := range files {
			if file.Name() != expected[i] || !file.Type().IsRegular() {
				t.Fatalf("unexpected or non-regular generated asset %q", file.Name())
			}
			data, err := os.ReadFile(filepath.Join(out, file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(data)
			digests[file.Name()] = hex.EncodeToString(sum[:])
		}
		return digests
	}
	before := snapshot()
	failures := []struct {
		name, command, expectedMessage string
		args                           []string
	}{
		{"Go", "go", "incompatible version", []string{"run", ".", "-fixture", badFixture, "-out", out}},
		{"UI", node, "invalid_poc_fixture_services", []string{"compose-ui.mjs", badFixture, out}},
	}
	for _, tc := range failures {
		t.Run(tc.name, func(t *testing.T) {
			output, err := run(tc.command, tc.args...)
			if err == nil || !strings.Contains(string(output), tc.expectedMessage) {
				t.Fatalf("invalid fixture was not rejected as expected: err=%v output=%s", err, output)
			}
			after := snapshot()
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("%s modified previously generated asset digests: before=%v after=%v",
					tc.name, before, after)
			}
		})
	}
}
