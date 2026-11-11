package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestThreeElementEnvelopesCarryTheirSchemaVersions asserts the schemaVersion of
// every three-element envelope.
//
// The catalogue advertises `tei.cli.three-element/v0`, and a test has to show
// what a caller actually receives: the outer envelope carries that identifier,
// and each subcommand's result carries its own. Naming them here also satisfies
// the advertised-schema coverage check, which otherwise lists the command with
// a reason.
func TestThreeElementEnvelopesCarryTheirSchemaVersions(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	// Each subcommand answers with its own schemaVersion. The catalogue
	// advertises the dispatcher's version (tei.cli.three-element/v0), which is
	// what an unknown or malformed subcommand returns; a caller that branches on
	// the version has to read the one the actual answer carries.
	for _, tc := range []struct {
		subcommand string
		envelope   string
		nested     string
		args       []string
	}{
		{"validate", "tei.cli.three-element.validate/v0", "tei.three-element.model/v0", []string{"-model", modelPath}},
		{"crud", "tei.cli.three-element.crud/v0", "tei.three-element.crud/v0", []string{"-model", modelPath}},
		{"cross-reference", "tei.cli.three-element.cross-reference/v0", "tei.three-element.cross-reference/v0", []string{"-model", modelPath}},
		// diff compares two models rather than reading one.
		{"diff", "tei.cli.three-element.diff/v0", "tei.three-element.diff/v0", []string{"-before", modelPath, "-after", modelPath}},
	} {
		t.Run(tc.subcommand, func(t *testing.T) {
			var out strings.Builder
			code, err := runThreeElementCommand(append([]string{tc.subcommand}, tc.args...), &out)
			if err != nil {
				t.Fatalf("command error: %v", err)
			}
			if code != 0 {
				t.Fatalf("exit=%d body=%s", code, out.String())
			}
			var envelope struct {
				SchemaVersion string `json:"schemaVersion"`
			}
			if err := json.Unmarshal([]byte(out.String()), &envelope); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if envelope.SchemaVersion != tc.envelope {
				t.Fatalf("envelope schemaVersion=%q, want %q", envelope.SchemaVersion, tc.envelope)
			}
			if !strings.Contains(out.String(), tc.nested) {
				t.Fatalf("body does not carry %q: %s", tc.nested, out.String())
			}
		})
	}

	// The dispatcher answers an unknown subcommand with its own envelope, so the
	// outer version is named without needing a model. An argument error is still
	// an answer: exit 2 with a JSON body.
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"unknown"}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 2 {
		t.Fatalf("exit=%d, want 2: %s", code, out.String())
	}
	if !strings.Contains(out.String(), "tei.cli.three-element/v0") {
		t.Fatalf("dispatcher envelope does not carry tei.cli.three-element/v0: %s", out.String())
	}
}
