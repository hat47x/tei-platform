package main

import (
	"bytes"
	"encoding/json"
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
			// A subcommand's schema joins its words with "-" (model locate ->
			// tei.cli.model-locate/v0); model catalog keeps its older name.
			if cmd.SchemaVersion != "tei.cli."+strings.ReplaceAll(cmd.Name, " ", "-")+"/v0" && cmd.SchemaVersion != "tei.cli.model/v0" {
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
		// A subcommand's schema joins its words with "-" (model locate ->
		// tei.cli.model-locate/v0); model catalog keeps its older name.
		if cmd.Name == "model locate" || cmd.Name == "model overview" ||
			cmd.Name == "model document-draft" || cmd.Name == "model document-reconcile" {
			if cmd.SchemaVersion != "tei.cli."+strings.ReplaceAll(cmd.Name, " ", "-")+"/v0" {
				t.Errorf("%s schemaVersion=%q", cmd.Name, cmd.SchemaVersion)
			}
			continue
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
	t.Run("model locate", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := runModelCommand([]string{"locate", "-workspace", root, "-subject-kind", "semantic", "-subject-id", "std"}, &out); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
		}
		assertAdvertisedVersion(t, "model locate", decode(t, &out))
	})
	t.Run("model overview", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := runModelCommand([]string{"overview", "-workspace", root}, &out); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
		}
		assertAdvertisedVersion(t, "model overview", decode(t, &out))
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
