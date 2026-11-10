package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A command that advertises a SchemaVersion promises the caller a response
// shape. A command whose schema no test checks advertises something no test has
// seen (#4032).
//
// TestDiscoveryAdvertisesEveryModeAResponseCanCarry compares the schemaVersion of
// every command in its case table with the catalog, so those commands count as
// asserted here. Anything else has to be listed below with a reason, and a
// listed command that gains an assertion has to be removed, so the gap can only
// shrink.
var unassertedSchemaVersionCommands = map[string]string{}

func TestAdvertisedSchemaVersionsAreAsserted(t *testing.T) {
	asserted := assertedCommandNames(t)
	var missing []string
	for _, record := range commandCatalog().Commands {
		if record.SchemaVersion == "" {
			continue
		}
		_, listed := unassertedSchemaVersionCommands[record.Name]
		switch {
		case asserted[record.Name] && listed:
			t.Errorf("%s now has an assertion; remove it from unassertedSchemaVersionCommands", record.Name)
		case !asserted[record.Name] && !listed:
			missing = append(missing, record.Name+" ("+record.SchemaVersion+")")
		}
	}
	for _, entry := range missing {
		t.Errorf("%s has no test that asserts its schemaVersion; add one or list the command with a reason", entry)
	}
	if len(missing) != 0 {
		t.Fatalf("%d advertised commands have neither an assertion nor a listed reason", len(missing))
	}
}

// assertedCommandNames returns the commands whose schema a test checks: either
// the command's schemaVersion literal appears in a test, or the command is a
// case in the discovery modes test, which compares responses with the catalog.
func assertedCommandNames(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	literals := map[string]bool{}
	caseCommands := map[string]bool{}
	argsPattern := regexp.MustCompile(`\[\]string\{"([a-z][a-z0-9-]*)"`)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, readErr := os.ReadFile(filepath.Join(".", name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, field := range strings.Fields(string(source)) {
			field = strings.Trim(field, "\"'`(),;{}")
			if strings.HasPrefix(field, "tei.cli.") {
				literals[field] = true
			}
		}
		for _, match := range argsPattern.FindAllStringSubmatch(string(source), -1) {
			caseCommands[match[1]] = true
		}
	}
	if len(literals) == 0 {
		t.Fatal("no schema version literal was found in the tests; the shape changed")
	}
	if len(caseCommands) == 0 {
		t.Fatal("no command was found in the discovery modes case table; the shape changed")
	}
	asserted := map[string]bool{}
	for _, record := range commandCatalog().Commands {
		if record.SchemaVersion == "" {
			continue
		}
		if literals[record.SchemaVersion] || caseCommands[record.Name] {
			asserted[record.Name] = true
		}
	}
	return asserted
}

func TestWorkspaceFilesSchemaVersionIsAsserted(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	if code, err := workspaceFilesJSON(&out, root, "", 0); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
	}
	assertSchemaVersionLiteral(t, out.Bytes(), "tei.cli.workspace-files/v0")
}

func TestViewSchemaVersionIsAsserted(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	if code, err := runViewCommand([]string{"-workspace", root, "-application", "order_entry", "-format", "json"}, &out); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
	}
	assertSchemaVersionLiteral(t, out.Bytes(), "tei.cli.view/v0")
}

func assertSchemaVersionLiteral(t *testing.T, body []byte, want string) {
	t.Helper()
	var envelope struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, body)
	}
	if envelope.SchemaVersion != want {
		t.Fatalf("schemaVersion=%q, want %q", envelope.SchemaVersion, want)
	}
}
