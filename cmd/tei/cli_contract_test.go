package main

import (
	"encoding/json"
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
