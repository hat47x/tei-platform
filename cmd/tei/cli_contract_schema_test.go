package main

import (
	"os"
	"strings"
	"testing"
)

// Every advertised command states which schemaVersion its JSON carries, or
// that its JSON is outside the contract (#2824). Commands wrapped in the
// #2354 result envelope must advertise the version the envelope really uses.
func TestCommandCatalogAdvertisesOutputSchema(t *testing.T) {
	outside := map[string]bool{"world": true}
	for _, cmd := range commandCatalog().Commands {
		if outside[cmd.Name] {
			if !cmd.OutsideContract || cmd.OutputSchemaVersion != "" {
				t.Fatalf("%s must be marked outside the contract without a schemaVersion: %+v", cmd.Name, cmd)
			}
			continue
		}
		if cmd.OutsideContract || cmd.OutputSchemaVersion == "" {
			t.Fatalf("%s must advertise a schemaVersion: %+v", cmd.Name, cmd)
		}
		if isResultEnvelopeCommand(cmd.Name) && cmd.OutputSchemaVersion != resultSchemaVersion(cmd.Name) {
			t.Fatalf("%s advertises %q, envelope uses %q", cmd.Name, cmd.OutputSchemaVersion, resultSchemaVersion(cmd.Name))
		}
	}
}

// The command table in the JSON output contract document must list every
// command that `tei commands` advertises, so the document and the catalogue
// cannot drift apart silently (#2824).
func TestCommandCatalogListedInContractDocument(t *testing.T) {
	raw, err := os.ReadFile("../../plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := strings.ReplaceAll(string(raw), "\r\n", "\n")
	for _, cmd := range commandCatalog().Commands {
		row := "| `" + cmd.Name
		if !strings.Contains(doc, "\n"+row+"`") && !strings.Contains(doc, "\n"+row+" ") {
			t.Errorf("contract document table lacks a row for %q", cmd.Name)
		}
		if cmd.OutputSchemaVersion != "" && !strings.Contains(doc, "`"+cmd.OutputSchemaVersion+"`") {
			t.Errorf("contract document does not mention %q", cmd.OutputSchemaVersion)
		}
	}
}
