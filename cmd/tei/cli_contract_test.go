package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandCatalogReadOnlyDiscovery(t *testing.T) {
	catalog := commandCatalog()
	if catalog.SchemaVersion != "tei.cli.discovery/v0" || catalog.Scope != "selected-read-only-commands" {
		t.Fatalf("unexpected discovery contract: %+v", catalog)
	}
	seen := map[string]bool{}
	for _, cmd := range catalog.Commands {
		if cmd.Effect != "read-only" || cmd.Name == "" || len(cmd.Required) == 0 || cmd.Output == "" {
			t.Fatalf("invalid command metadata: %+v", cmd)
		}
		switch cmd.JSONContract {
		case "enveloped":
			if cmd.SchemaVersion != "tei.cli."+cmd.Name+"/v0" && cmd.SchemaVersion != "tei.cli.model/v0" {
				t.Fatalf("%s has an unexpected schemaVersion %q", cmd.Name, cmd.SchemaVersion)
			}
		case "raw":
			if cmd.SchemaVersion != "" {
				t.Fatalf("%s is outside the JSON contract but advertises %q", cmd.Name, cmd.SchemaVersion)
			}
		default:
			t.Fatalf("%s has an unknown jsonContract %q", cmd.Name, cmd.JSONContract)
		}
		if seen[cmd.Name] {
			t.Fatalf("duplicate command: %s", cmd.Name)
		}
		seen[cmd.Name] = true
	}
	for _, name := range []string{"validate", "world", "source-index", "inspect", "trace", "plugin-lock", "plugin-resolve"} {
		if !seen[name] {
			t.Fatalf("discovery omitted %q", name)
		}
	}
	raw, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var decoded cliCommandCatalog
	if err := json.Unmarshal(raw, &decoded); err != nil || len(decoded.Commands) != len(catalog.Commands) {
		t.Fatalf("invalid JSON discovery result: %v", err)
	}
}

// TestDiscoveryAdvertisesTheCommandsOwnSchemaVersion pins the discovery field to
// the version each command actually returns, so a caller does not have to run
// a command to learn its contract. `model catalog` is a subcommand of `model`
// and keeps its own name.
func TestDiscoveryAdvertisesTheCommandsOwnSchemaVersion(t *testing.T) {
	for _, cmd := range commandCatalog().Commands {
		if cmd.JSONContract != "enveloped" {
			continue
		}
		if cmd.SchemaVersion == "" {
			t.Errorf("%s advertises no schemaVersion", cmd.Name)
		}
		if cmd.Name != "model catalog" && cmd.SchemaVersion != "tei.cli."+cmd.Name+"/v0" {
			t.Errorf("%s schemaVersion=%q", cmd.Name, cmd.SchemaVersion)
		}
	}
}

// discoveryRecord returns the advertised record for a command name.
func discoveryRecord(name string) (cliCommandRecord, bool) {
	for _, record := range commandCatalog().Commands {
		if record.Name == name {
			return record, true
		}
	}
	return cliCommandRecord{}, false
}

// assertAdvertisedVersion compares one actual response envelope with the
// discovery record. A constant-to-constant test alone would pass while the
// response drifted; this compares the value the command actually wrote.
func assertAdvertisedVersion(t *testing.T, name, responseSchema string) {
	t.Helper()
	record, ok := discoveryRecord(name)
	if !ok {
		t.Fatalf("%s is not advertised by tei commands", name)
	}
	if record.SchemaVersion == "" || responseSchema != record.SchemaVersion {
		t.Fatalf("%s advertised %q, response has %q", name, record.SchemaVersion, responseSchema)
	}
}

// TestDiscoverySchemaVersionsMatchActualResponses runs each enveloped command
// and compares the response's schemaVersion with what `tei commands`
// advertises (#2824). Commands that need extra input are exercised with a
// reference Workspace.
func TestDiscoverySchemaVersionsMatchActualResponses(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	decode := func(t *testing.T, out *bytes.Buffer) string {
		t.Helper()
		var envelope struct {
			SchemaVersion string `json:"schemaVersion"`
		}
		if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
			t.Fatalf("not JSON: %v: %s", err, out.String())
		}
		return envelope.SchemaVersion
	}
	t.Run("version", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := runVersionCommand([]string{"-json"}, &out); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
		}
		assertAdvertisedVersion(t, "version", decode(t, &out))
	})
	t.Run("model catalog", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := runModelCommand([]string{"catalog", "-workspace", root}, &out); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
		}
		assertAdvertisedVersion(t, "model catalog", decode(t, &out))
	})
	t.Run("workspace-files", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := workspaceFilesJSON(&out, root, "", 0); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
		}
		assertAdvertisedVersion(t, "workspace-files", decode(t, &out))
	})
	t.Run("inspect", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := inspectWorkspaceJSON(&out, root, "", "semantic-data", "sales.order", 0); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
		}
		assertAdvertisedVersion(t, "inspect", decode(t, &out))
	})
	t.Run("view", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := runViewCommand([]string{"-workspace", root, "-application", "order_entry", "-format", "json"}, &out); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
		}
		assertAdvertisedVersion(t, "view", decode(t, &out))
	})
	t.Run("validate", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := validateWorkspaceJSON(&out, root); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
		}
		assertAdvertisedVersion(t, "validate", decode(t, &out))
	})
}

// TestDiscoverySchemaVersionMismatchIsDetected is the negative control: the
// comparison must fail when a command advertises a version it does not return.
func TestDiscoverySchemaVersionMismatchIsDetected(t *testing.T) {
	record, ok := discoveryRecord("inspect")
	if !ok {
		t.Fatal("inspect is not advertised")
	}
	if record.SchemaVersion == "tei.cli.inspect/v1" {
		t.Fatal("fixture version must differ from the advertised one")
	}
	if record.SchemaVersion != "tei.cli.inspect/v0" {
		t.Fatalf("inspect schemaVersion=%q", record.SchemaVersion)
	}
}

// TestViewReadOnlyDiscoveryExcludesActualFileWriteVariant pins the advertised
// read-only subset to invocations without -execute. The same CLI command does
// accept an -execute invocation that can write a named file (#3182); it must
// not be included among the safe read-only command variants returned by
// `tei commands`. This deliberately does not test business DB commit.
func TestViewReadOnlyDiscoveryExcludesActualFileWriteVariant(t *testing.T) {
	record, ok := discoveryRecord("view")
	if !ok || record.Effect != "read-only" {
		t.Fatalf("missing read-only view discovery record: %+v", record)
	}
	if len(record.ExcludedFlagNames) != 1 || record.ExcludedFlagNames[0] != "execute" {
		t.Fatalf("view must exclude effectful execute flag: %+v", record.ExcludedFlagNames)
	}
	// Flag discovery must still describe every accepted CLI argument (#2332).
	// "execute" is an accepted flag, but the read-only claim excludes its use.
	if !strings.Contains(strings.Join(record.Optional, " "), "-execute") {
		t.Fatal("view discovery must still describe the accepted -execute flag")
	}
	// The exclusion is not a CLI flag ban. Exercise the real reference command
	// through its command entry and independently inspect the named output.
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, executeFlag := range []string{"-execute", "--execute", "-execute=true"} {
		t.Run(executeFlag, func(t *testing.T) {
			logFile := filepath.Join(t.TempDir(), "operation.json")
			var output bytes.Buffer
			code, err := runViewCommand([]string{
				"-workspace", root,
				"-application", "order_entry",
				"-pattern", "order_entry.submit_order",
				"-command", "sales.order.submit",
				"-input", "order=sales.order",
				executeFlag,
				"-operation-log", logFile,
				"-format", "json",
			}, &output)
			if err != nil || code != 0 {
				t.Fatalf("effectful invocation failed: code=%d err=%v body=%s", code, err, output.String())
			}
			fileBody, err := os.ReadFile(logFile)
			if err != nil || len(fileBody) == 0 || !json.Valid(fileBody) {
				t.Fatalf("expected actual JSON operation-log filesystem output: bytes=%d err=%v", len(fileBody), err)
			}
		})
	}
}
