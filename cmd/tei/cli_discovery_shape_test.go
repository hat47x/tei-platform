package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"testing"
)

// TestDiscoverySuccessShapeMatchesTheResponseBody closes the open question on
// #2824: `version -json` is advertised as jsonContract "enveloped", but its
// successful body carries schemaVersion at the top level without ok or result.
// A caller that trusts "enveloped" would read `ok` as absent-in-error rather
// than absent-by-design. The catalogue now names the body it returns.
func TestDiscoverySuccessShapeMatchesTheResponseBody(t *testing.T) {
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
		if stderr.Len() > 0 {
			t.Fatalf("%s: enveloped command wrote stderr: %q", cmd.Name, stderr.String())
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("%s: stdout is not one JSON object: %v (%q)", cmd.Name, err, stdout.String())
		}
		_, hasOK := body["ok"]
		_, hasResult := body["result"]
		want := cmd.SuccessShape
		if want == "" {
			// The default: a body with ok, and a result when the call succeeds.
			if !hasOK {
				t.Fatalf("%s advertises no successShape but the body has no ok", cmd.Name)
			}
			if code == 0 && !hasResult {
				t.Fatalf("%s exit 0 without result", cmd.Name)
			}
			continue
		}
		switch want {
		case "identity":
			if hasOK || hasResult {
				t.Fatalf("%s advertises identity but the body has ok=%t result=%t", cmd.Name, hasOK, hasResult)
			}
			if _, ok := body["schemaVersion"]; !ok {
				t.Fatalf("%s identity has no schemaVersion", cmd.Name)
			}
		case "envelope":
			if !hasOK {
				t.Fatalf("%s advertises envelope but the body has no ok", cmd.Name)
			}
		default:
			t.Fatalf("%s has an unnamed successShape %q", cmd.Name, want)
		}
	}
}
