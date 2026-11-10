package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/modelref"
)

// TestTraceDefaultsComeFromOneSource pins #3605's third expectation: the CLI
// flag default, `tei commands` argumentDefaults and the MCP adapter must read
// the same value. Before this, three copies existed and the internal
// normalization disagreed with all of them ("full" instead of "summary").
func TestTraceDefaultsComeFromOneSource(t *testing.T) {
	// The catalogue advertises the modelref constants, not a copy.
	catalog := commandCatalog()
	var record cliCommandRecord
	for _, candidate := range catalog.Commands {
		if candidate.Name == "trace" {
			record = candidate
			break
		}
	}
	if record.Name == "" {
		t.Fatal("trace is not in the catalogue")
	}
	if record.ArgumentDefaults["direction"] != modelref.TraceDefaultDirection ||
		record.ArgumentDefaults["mode"] != modelref.TraceDefaultMode {
		t.Fatalf("catalogue defaults = %+v, modelref = %q/%q",
			record.ArgumentDefaults, modelref.TraceDefaultDirection, modelref.TraceDefaultMode)
	}
	// The advertised default is a member of the advertised closed set.
	if !valueInSet(record.ArgumentValues["mode"], record.ArgumentDefaults["mode"]) {
		t.Fatalf("advertised default %q is outside %v", record.ArgumentDefaults["mode"], record.ArgumentValues["mode"])
	}
}

func valueInSet(set []string, want string) bool {
	for _, value := range set {
		if value == want {
			return true
		}
	}
	return false
}

// The built CLI must apply the same value the catalogue advertises when -mode is
// omitted, so an advertised default and the applied value cannot drift.
func TestOmittedModeAppliesTheAdvertisedDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	root := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	command := exec.Command(binary, "trace", "-workspace", root, "-scope", "s",
		"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer")
	var out bytes.Buffer
	command.Stdout = &out
	if err := command.Run(); err != nil {
		t.Fatalf("trace: %v (%q)", err, out.String())
	}
	var response struct {
		Result struct {
			Trace struct {
				Mode string `json:"mode"`
			} `json:"trace"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("trace answer is not JSON: %v", err)
	}
	if response.Result.Trace.Mode != modelref.TraceDefaultMode {
		t.Fatalf("omitted -mode applied %q, advertised %q",
			response.Result.Trace.Mode, modelref.TraceDefaultMode)
	}
	// The text catalogue entry and the applied value agree as strings.
	if !strings.EqualFold(response.Result.Trace.Mode, modelref.TraceDefaultMode) {
		t.Fatalf("mode %q does not match %q", response.Result.Trace.Mode, modelref.TraceDefaultMode)
	}
}
