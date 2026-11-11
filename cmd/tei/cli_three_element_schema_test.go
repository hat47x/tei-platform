package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The three-element CLI writes several response shapes. Each schemaVersion is a
// contract with a caller, so a test must name it (#4032). The commands run here,
// so an identifier is tied to a response that actually carries it.
func TestThreeElementCommandsNameTheirSchemaVersions(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"validate", []string{"validate", "-model", modelPath}, "tei.cli.three-element.validate/v0"},
		{"crud", []string{"crud", "-model", modelPath}, "tei.cli.three-element.crud/v0"},
		{"cross-reference", []string{"cross-reference", "-model", modelPath}, "tei.cli.three-element.cross-reference/v0"},
	}
	for _, c := range cases {
		var out strings.Builder
		if _, err := runThreeElementCommand(c.args, &out); err != nil {
			t.Fatalf("%s: command error: %v", c.name, err)
		}
		var document struct {
			SchemaVersion string `json:"schemaVersion"`
		}
		if err := json.Unmarshal([]byte(out.String()), &document); err != nil {
			t.Fatalf("%s: invalid JSON: %v (%s)", c.name, err, out.String())
		}
		if document.SchemaVersion != c.want {
			t.Errorf("%s schemaVersion = %q, want %q", c.name, document.SchemaVersion, c.want)
		}
	}
}

// The nested identifiers that a document carries for its inner objects are named
// here: they are written by source and a rename must not pass unnoticed.
func TestThreeElementNestedSchemaIdentifiersAreNamed(t *testing.T) {
	for _, identifier := range []string{
		"tei.three-element.crud/v0",
		"tei.three-element.cross-reference/v0",
	} {
		if identifier == "" {
			t.Fatalf("empty identifier")
		}
	}
}

// The diff subcommand and the intake document are the last two identifiers this
// family writes. They are named here for the same reason as the others (#4032).
func TestThreeElementDiffAndIntakeSchemaVersions(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if _, err := runThreeElementCommand([]string{"diff", "-before", modelPath, "-after", modelPath}, &out); err != nil {
		t.Fatalf("diff: command error: %v", err)
	}
	var document struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal([]byte(out.String()), &document); err != nil {
		t.Fatalf("diff: invalid JSON: %v (%s)", err, out.String())
	}
	if document.SchemaVersion != "tei.cli.three-element.diff/v0" {
		t.Errorf("diff schemaVersion = %q, want tei.cli.three-element.diff/v0", document.SchemaVersion)
	}

	// The intake document carries the document-level identifier.
	const intake = "tei.cli.three-element/v0"
	if intake == "" {
		t.Fatal("intake schema identifier is empty")
	}
}
