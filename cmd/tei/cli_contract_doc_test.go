package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// contractDocTableRows extracts `| \`name\` | \`schemaVersion\` | ...` rows from
// the CLI JSON output contract. The table is the document's prose, not a
// generated artifact, so this check reads it as text.
func contractDocTableRows(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "plan", "design", "CLI_JSON_OUTPUT_CONTRACT_2026-10.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the CLI JSON contract: %v", err)
	}
	row := regexp.MustCompile("(?m)^\\| `([^|]+?)` \\| `(tei\\.cli\\.[^`]+)` \\|")
	rows := map[string]string{}
	for _, match := range row.FindAllStringSubmatch(string(body), -1) {
		rows[strings.TrimSpace(match[1])] = match[2]
	}
	if len(rows) == 0 {
		t.Fatalf("%s has no schemaVersion table rows", path)
	}
	return rows
}

// TestContractDocumentCoversEveryAdvertisedCommand closes the open item in
// CLI_JSON_OUTPUT_CONTRACT_2026-10.md §8: whether the table and the code agree.
// Every enveloped command that `tei commands` advertises must appear in the
// table with the same schemaVersion, and each table row must name a command
// that exists, so a rename in one place fails here instead of being found by
// a reader (#2824).
func TestContractDocumentCoversEveryAdvertisedCommand(t *testing.T) {
	rows := contractDocTableRows(t)
	docName := func(command string) string {
		switch command {
		case "version", "validate":
			// The table documents the JSON invocation, not the text default.
			if command == "version" {
				return "version -json"
			}
			return "validate -workspace -json"
		}
		return command
	}
	catalog := commandCatalog()
	advertised := map[string]bool{}
	for _, cmd := range catalog.Commands {
		name := docName(cmd.Name)
		advertised[name] = true
		if cmd.JSONContract != "enveloped" {
			continue
		}
		version, ok := rows[name]
		if !ok {
			t.Errorf("%s is advertised but has no row in the CLI JSON contract table", name)
			continue
		}
		if version != cmd.SchemaVersion {
			t.Errorf("%s: table says %q, discovery advertises %q", name, version, cmd.SchemaVersion)
		}
	}
	// `commands` documents the discovery response itself, which is not one of
	// the commands it lists. Everything else must be traceable to an entry.
	if _, ok := rows["commands"]; !ok {
		t.Errorf("the contract table no longer documents the commands response")
	}
	for name := range rows {
		if name == "commands" || advertised[name] {
			continue
		}
		t.Errorf("the contract table documents %q, which tei commands does not advertise", name)
	}
}
