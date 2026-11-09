package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTextInvocationsDoNotEmitTheJSONEnvelope pins the "text; json when ..."
// half of the discovery catalogue. A caller that probes for an envelope by
// reading stdout must not receive one from `view -format text|html|print` or
// from text `validate`; otherwise it would parse human text as JSON, or treat
// a document as an error response.
//
// The JSON envelope is the contract for the JSON invocation only. This test
// says so at the real argv/stdout boundary.
func TestTextInvocationsDoNotEmitTheJSONEnvelope(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	reference := filepath.Join("..", "..", "reference", "workspace")

	run := func(args ...string) (string, int, string) {
		t.Helper()
		command := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("%q: CLI execution: %v", args, err)
			}
			code = exit.ExitCode()
		}
		return stdout.String(), code, stderr.String()
	}

	textCases := []struct {
		args []string
		want string
	}{
		{[]string{"view", "-workspace", reference, "-application", "order_entry", "-format", "text"}, "application order_entry"},
		{[]string{"validate", "-workspace", reference}, "ok\n"},
	}
	for _, tc := range textCases {
		stdout, code, stderr := run(tc.args...)
		if code != 0 || stderr != "" {
			t.Fatalf("%q: exit=%d stderr=%q", tc.args, code, stderr)
		}
		if !strings.Contains(stdout, tc.want) {
			t.Fatalf("%q: stdout=%q does not contain %q", tc.args, stdout, tc.want)
		}
		if strings.HasPrefix(strings.TrimSpace(stdout), "{") {
			t.Fatalf("%q: text invocation emitted JSON: %q", tc.args, stdout)
		}
	}

	// html and print are documents, not envelopes, even though the command
	// advertises jsonContract "enveloped" for its JSON form.
	for _, format := range []string{"html", "print"} {
		args := []string{"view", "-workspace", reference, "-application", "order_entry",
			"-pattern", "order_entry.order_lines", "-format", format}
		stdout, code, stderr := run(args...)
		if code != 0 || stderr != "" {
			t.Fatalf("%q: exit=%d stderr=%q", args, code, stderr)
		}
		if !strings.HasPrefix(stdout, "<!doctype html>") {
			t.Fatalf("%q: stdout=%q is not an HTML document", args, stdout)
		}
		if strings.Contains(stdout, `"schemaVersion"`) {
			t.Fatalf("%q: HTML document carries an envelope field", args)
		}
	}

	// The JSON invocations still produce exactly one envelope.
	jsonCases := [][]string{
		{"view", "-workspace", reference, "-application", "order_entry", "-format", "json"},
		{"validate", "-workspace", reference, "-json"},
	}
	for _, args := range jsonCases {
		stdout, code, stderr := run(args...)
		if code != 0 || stderr != "" {
			t.Fatalf("%q: exit=%d stderr=%q", args, code, stderr)
		}
		var envelope struct {
			SchemaVersion string `json:"schemaVersion"`
		}
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatalf("%q: JSON invocation is not one envelope: %v (%q)", args, err, stdout)
		}
		if envelope.SchemaVersion != "tei.cli."+args[0]+"/v0" {
			t.Fatalf("%q: schemaVersion=%q", args, envelope.SchemaVersion)
		}
	}

	// The catalogue must not describe these invocations as plain json.
	for _, record := range commandCatalog().Commands {
		switch record.Name {
		case "view", "validate":
			if !strings.HasPrefix(record.Output, "text;") || !strings.Contains(record.Output, "json") {
				t.Fatalf("%s output=%q must describe both the text default and the JSON form", record.Name, record.Output)
			}
		}
	}
}
