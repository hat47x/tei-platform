package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// cliCommandNameForTool maps each advertised tool to the CLI command that
// performs the same operation. The mapping is declared here, not inferred from
// a string transform, because the two surfaces name operations differently
// (`model catalog` vs `tei_model_catalog`).
var cliCommandNameForTool = map[string]string{
	"tei_model_catalog":  "model catalog",
	"tei_model_locate":   "model locate",
	"tei_model_overview": "model overview",
	"tei_inspect":        "inspect",
	"tei_trace":          "trace",
	"tei_validate":       "validate",
}

// cliResponseSchemaForTool pins the response document each tool answers. A
// caller that reads the tool list learns this name from the CLI catalog, so the
// two must agree; otherwise a client has to know which surface to trust
// (ADR-0048 §2.6).
var cliResponseSchemaForTool = map[string]string{
	"tei_model_catalog":  "tei.cli.model/v0",
	"tei_model_locate":   "tei.cli.model-locate/v0",
	"tei_model_overview": "tei.cli.model-overview/v0",
	"tei_inspect":        "tei.cli.inspect/v0",
	"tei_trace":          "tei.cli.trace/v0",
	"tei_validate":       "tei.cli.validate/v0",
}

// TestToolListAgreesWithTheCLICatalog runs the real CLI discovery document and
// compares it with the tools the adapter advertises. It fails if a tool names a
// command the CLI does not advertise, if the CLI records that operation as
// anything but read-only, or if the response schema the adapter writes differs
// from the one the CLI catalog declares.
func TestToolListAgreesWithTheCLICatalog(t *testing.T) {
	cliPath := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		cliPath += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", cliPath, "../../cmd/tei").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	var stdout, stderr bytes.Buffer
	cli := exec.Command(cliPath, "commands")
	cli.Stdout, cli.Stderr = &stdout, &stderr
	if err := cli.Run(); err != nil {
		t.Fatalf("CLI commands: %v (%s)", err, stderr.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("CLI wrote stderr: %q", stderr.String())
	}

	var catalog struct {
		SchemaVersion string `json:"schemaVersion"`
		Scope         string `json:"scope"`
		Commands      []struct {
			Name          string   `json:"name"`
			Effect        string   `json:"effect"`
			SchemaVersion string   `json:"schemaVersion"`
			JSONContract  string   `json:"jsonContract"`
			WriteFlags    []string `json:"writeFlags"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &catalog); err != nil {
		t.Fatalf("CLI discovery is not JSON: %v (%q)", err, stdout.String())
	}
	if catalog.SchemaVersion != "tei.cli.discovery/v0" {
		t.Fatalf("discovery schemaVersion = %q", catalog.SchemaVersion)
	}
	type record struct {
		Effect        string
		SchemaVersion string
		JSONContract  string
		WriteFlags    []string
	}
	records := map[string]record{}
	for _, c := range catalog.Commands {
		records[c.Name] = record{c.Effect, c.SchemaVersion, c.JSONContract, c.WriteFlags}
	}

	tools := toolList()
	if len(tools) != len(cliCommandNameForTool) {
		t.Fatalf("adapter advertises %d tools, the mapping names %d", len(tools), len(cliCommandNameForTool))
	}
	for _, tool := range tools {
		command, ok := cliCommandNameForTool[tool.Name]
		if !ok {
			t.Fatalf("tool %q has no CLI command mapping", tool.Name)
		}
		record, ok := records[command]
		if !ok {
			t.Fatalf("tool %q maps to %q, which the CLI catalog does not advertise", tool.Name, command)
		}
		if record.Effect != "read-only" {
			t.Fatalf("%q is advertised as effect=%q; the adapter must expose read-only operations only", command, record.Effect)
		}
		if len(record.WriteFlags) > 0 {
			t.Fatalf("%q writes files via %v; the adapter must not expose a writing operation", command, record.WriteFlags)
		}
		if want := cliResponseSchemaForTool[tool.Name]; record.SchemaVersion != want {
			t.Fatalf("%q: CLI schemaVersion=%q, adapter pins %q", command, record.SchemaVersion, want)
		}
		if record.JSONContract != "enveloped" {
			t.Fatalf("%q: jsonContract=%q, the adapter answers an envelope", command, record.JSONContract)
		}
	}
}

// TestToolDescriptionsNameTheCLICommand stops a client from having to guess
// which CLI invocation the tool stands for. The CLI is named in backticks, the
// same way the adapter README does.
func TestToolDescriptionsNameTheCLICommand(t *testing.T) {
	for _, tool := range toolList() {
		command, ok := cliCommandNameForTool[tool.Name]
		if !ok {
			t.Fatalf("tool %q has no CLI command mapping", tool.Name)
		}
		if want := "tei " + command; !bytes.Contains([]byte(tool.Description), []byte(want)) {
			t.Fatalf("tool %q description does not name %q: %q", tool.Name, want, tool.Description)
		}
	}
}
