package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// TestAdvertisedOutputAgreesWithJSONContract pins the `output` field of
// `tei commands` against the two other fields that describe the same thing
// (#2824). Before this test the field was only checked for being non-empty,
// so a command could be advertised as "json" while its jsonContract said
// "none", or as text while every invocation returned an envelope.
func TestAdvertisedOutputAgreesWithJSONContract(t *testing.T) {
	for _, cmd := range commandCatalog().Commands {
		output := strings.TrimSpace(cmd.Output)
		if output == "" {
			t.Errorf("%s advertises an empty output", cmd.Name)
			continue
		}
		saysJSON := strings.HasPrefix(output, "json") || strings.Contains(output, "json when")
		switch cmd.JSONContract {
		case "enveloped":
			// The envelope is JSON even on the failure path, so an enveloped
			// command must say json somewhere in its output column.
			if !saysJSON {
				t.Errorf("%s is enveloped but advertises output %q", cmd.Name, output)
			}
			if cmd.SuccessShape == "identity" && output == "json" {
				t.Errorf("%s returns a build identity, not a bare json result, but advertises %q", cmd.Name, output)
			}
		case "raw":
			// `world` writes the raw World; it is json and it is not text.
			if output != "json" {
				t.Errorf("%s is raw but advertises output %q", cmd.Name, output)
			}
		default:
			t.Errorf("%s has an unknown jsonContract %q", cmd.Name, cmd.JSONContract)
		}
	}
}

// TestTextDefaultsDoNotEmitTheJSONEnvelope is the process-level half: the
// commands that advertise a text default must not answer a bare invocation
// with a JSON envelope, and the ones that advertise json must. Reading only
// the catalogue would keep the two fields consistent while both drifted away
// from the binary.
func TestTextDefaultsDoNotEmitTheJSONEnvelope(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := buildCLIForLifecycle(t)
	run := func(args ...string) (string, int) {
		t.Helper()
		cmd := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		code := 0
		if err := cmd.Run(); err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("argv=%q: %v", args, err)
			}
			code = exit.ExitCode()
		}
		return stdout.String(), code
	}

	// Text defaults: no JSON envelope on standard output.
	for _, args := range [][]string{
		{"version"},
		{"validate", "-workspace", "../../reference/workspace"},
		{"view", "-workspace", "../../reference/workspace", "-application", "order_entry"},
	} {
		stdout, code := run(args...)
		if code != 0 {
			t.Fatalf("argv=%q toString exit=%d stdout=%q", args, code, stdout)
		}
		if strings.HasPrefix(strings.TrimSpace(stdout), "{") {
			var body map[string]json.RawMessage
			if json.Unmarshal([]byte(stdout), &body) == nil && body["schemaVersion"] != nil {
				t.Fatalf("argv=%q advertises text but wrote a JSON envelope: %q", args, stdout)
			}
		}
	}

	// JSON defaults: a bare invocation still answers on standard output with
	// one JSON document, even when it is an argument failure.
	for _, args := range [][]string{
		{"workspace-files"},
		{"inspect"},
		{"trace"},
		{"source-index"},
		{"plugin-lock"},
	} {
		stdout, code := run(args...)
		if code != 1 && code != 2 {
			t.Fatalf("argv=%q exit=%d stdout=%q", args, code, stdout)
		}
		var envelope struct {
			SchemaVersion string      `json:"schemaVersion"`
			OK            bool        `json:"ok"`
			Error         *cliFailure `json:"error"`
		}
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatalf("argv=%q advertises json but stdout is not one envelope: %v (%q)", args, err, stdout)
		}
		if envelope.SchemaVersion == "" || envelope.OK || envelope.Error == nil {
			t.Fatalf("argv=%q did not fail closed in the envelope: %q", args, stdout)
		}
	}
}
