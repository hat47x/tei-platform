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
		if seen[cmd.Name] {
			t.Fatalf("duplicate command: %s", cmd.Name)
		}
		seen[cmd.Name] = true
	}
	for _, name := range []string{"validate", "world", "source-index", "plugin-lock", "plugin-resolve"} {
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
