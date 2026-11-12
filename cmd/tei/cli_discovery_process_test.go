package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"testing"
)

// TestDiscoverySchemaVersionMatchesTheActualResponse pins the discovery
// contract (#2824): the schemaVersion that `tei commands` advertises for an
// enveloped command must be the schemaVersion the command actually returns.
// The catalogue test only compares constants; this one runs each command in a
// real process, so a rename in the response path cannot silently disagree with
// what a caller was told before it ran anything.
//
// The invocation uses the cheapest path for each command (no workspace, or a
// deliberate argument failure), because the schemaVersion belongs to the
// envelope regardless of success or failure. `version` is the exception: with
// no arguments it prints text, so it is asked for -json.
//
// `version -json` also does not carry ok/error: it returns the build identity
// object with schemaVersion at the top level. The catalogue labels it
// successShape "identity", and cli_discovery_shape_test.go checks that body.
// This test still requires the schemaVersion to match for that command.
//
// Negative control, run by hand before the PR: changing the advertised
// `tei.cli.model/v0` to `tei.cli.model/v9` in cli_contract.go makes this test
// fail with `model catalog: discovery advertises "tei.cli.model/v9" but the
// response has "tei.cli.model/v0"`.
func TestDiscoverySchemaVersionMatchesTheActualResponse(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	for _, cmd := range commandCatalog().Commands {
		if cmd.JSONContract != "enveloped" {
			continue
		}
		args := discoveryProbeArgs(cmd.Name)
		command := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("%s: CLI execution: %v", cmd.Name, err)
			}
			code = exit.ExitCode()
		}
		if code != 0 && code != 1 && code != 2 {
			t.Fatalf("%s: unexpected exit code %d (stdout=%q)", cmd.Name, code, stdout.String())
		}
		if stderr.Len() > 0 {
			t.Fatalf("%s: enveloped command wrote stderr: %q", cmd.Name, stderr.String())
		}
		var response struct {
			SchemaVersion string      `json:"schemaVersion"`
			OK            bool        `json:"ok"`
			Error         *cliFailure `json:"error"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("%s: stdout is not one JSON envelope: %v (%q)", cmd.Name, err, stdout.String())
		}
		if response.SchemaVersion != cmd.SchemaVersion {
			t.Fatalf("%s: discovery advertises %q but the response has %q",
				cmd.Name, cmd.SchemaVersion, response.SchemaVersion)
		}
		if cmd.Name == "version" {
			continue
		}
		if code == 0 && !response.OK {
			t.Fatalf("%s: exit 0 without ok:true (%q)", cmd.Name, stdout.String())
		}
		if code != 0 && (response.OK || response.Error == nil) {
			t.Fatalf("%s: exit %d without a machine-readable error (%q)", cmd.Name, code, stdout.String())
		}
	}
}

// discoveryProbeArgs returns the shortest argv that reaches the command's own
// argument handling. "model catalog" is a subcommand, so the catalogue's single
// display name is split into the two argv words the CLI dispatches on.
func discoveryProbeArgs(name string) []string {
	switch name {
	case "model catalog":
		return []string{"model", "catalog"}
	case "model locate":
		return []string{"model", "locate"}
	case "model overview":
		return []string{"model", "overview"}
	case "model document-draft":
		return []string{"model", "document-draft"}
	case "model document-reconcile":
		return []string{"model", "document-reconcile"}
	case "version":
		return []string{"version", "-json"}
	case "validate":
		// Without -json, validate reports missing -workspace as text on stderr.
		// -json keeps the probe on the envelope path this contract covers.
		return []string{"validate", "-json"}
	default:
		return []string{name}
	}
}
