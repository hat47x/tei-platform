package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A command that advertises a SchemaVersion promises the caller a response
// shape. Two of them are asserted here because their response bodies are cheap
// to produce; the rest are listed with the reason they are not, so the set of
// advertised-but-unasserted schemas can only shrink (#4032).
var unassertedSchemaVersionCommands = map[string]string{
	"knowledge-context":                      "asserted through TestDiscoveryAdvertisesEveryModeAResponseCanCarry, which reads modes rather than the schema literal",
	"knowledge-data-design-candidate-review": "asserted through the discovery modes test, which reads result shapes rather than the schema literal",
	"knowledge-qualify":                      "asserted through the discovery modes test, which reads result shapes rather than the schema literal",
	"knowledge-review-disclose":              "asserted through the discovery modes test, which reads result shapes rather than the schema literal",
	"knowledge-review-handoff":               "asserted through the discovery modes test, which reads result shapes rather than the schema literal",
	"knowledge-review-plan":                  "asserted through the discovery modes test, which reads result shapes rather than the schema literal",
	"plugin-lock":                            "asserted through the discovery modes test, which reads modes rather than the schema literal",
	"plugin-resolve":                         "asserted through the discovery modes test, which reads modes rather than the schema literal",
	"plugin-runtime-plan":                    "asserted through the discovery modes test, which reads result shapes rather than the schema literal",
	"requirements":                           "asserted through the discovery modes test, which reads result shapes rather than the schema literal",
	"self-build-entry":                       "spawns a child executable; its schema is asserted by the self-build test",
	"source-index":                           "asserted through TestDiscoveryAdvertisesEveryModeAResponseCanCarry, which reads modes rather than the schema literal",
}

// TestAdvertisedSchemaVersionsAreAsserted keeps the gap visible: a command added
// without a schema assertion fails here until it is either asserted or listed
// with a reason.
func TestAdvertisedSchemaVersionsAreAsserted(t *testing.T) {
	asserted := schemaVersionsAssertedByTests(t)
	catalog := commandCatalog()
	var missing []string
	for _, record := range catalog.Commands {
		if record.SchemaVersion == "" {
			continue
		}
		_, listed := unassertedSchemaVersionCommands[record.Name]
		switch {
		case asserted[record.SchemaVersion] && listed:
			t.Errorf("%s now asserts %s; remove it from unassertedSchemaVersionCommands", record.Name, record.SchemaVersion)
		case !asserted[record.SchemaVersion] && !listed:
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

// schemaVersionsAssertedByTests returns every `tei.cli.*/v0` literal that
// appears in this package's tests.
func schemaVersionsAssertedByTests(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	asserted := map[string]bool{}
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
				asserted[field] = true
			}
		}
	}
	if len(asserted) == 0 {
		t.Fatal("no schema version was found in the tests; the literal shape changed")
	}
	return asserted
}

// TestWorkspaceFilesSchemaVersionIsAsserted pins the two cheap response bodies.
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
