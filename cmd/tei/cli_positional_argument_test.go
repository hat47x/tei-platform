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

// referenceWorkspacePath is the workspace every case here reads. The tests need
// a path that parses; the subject of the test is the extra bare word, not the
// workspace contents.
func referenceWorkspacePath() string {
	return filepath.Join("..", "..", "reference", "workspace")
}

// The CLI is flag-only: every command reads its inputs from flags, so a bare
// word is a typo. `validate` and `world` used to ignore one and still exit 0,
// which makes a mistyped run look like a clean one (#4165). Every advertised
// command must reject it as an argument failure, and say which word it rejected.
func TestCommandsRejectPositionalArguments(t *testing.T) {
	binary := buildCLIForLifecycle(t)

	cases := [][]string{
		{"validate", "-workspace", referenceWorkspacePath(), "stray"},
		{"world", "-workspace", referenceWorkspacePath(), "stray"},
		{"inspect", "-workspace", referenceWorkspacePath(),
			"-subject-kind", "semantic-data", "-subject-id", "sales.order", "stray"},
		{"trace", "-workspace", referenceWorkspacePath(), "-scope", "s",
			"-subject-kind", "semantic-data", "-subject-id", "sales.order", "stray"},
	}
	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			command := exec.Command(binary, args...)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("%q: CLI execution: %v", args, err)
			}
			if code := exit.ExitCode(); code != 2 {
				t.Fatalf("%q: exit=%d, want 2 for an argument failure (stdout=%q stderr=%q)",
					args, code, stdout.String(), stderr.String())
			}
			// The message has to name the offending word, or the caller cannot
			// tell which argument was unexpected.
			combined := stdout.String() + stderr.String()
			if !strings.Contains(combined, "positional") || !strings.Contains(combined, "stray") {
				t.Fatalf("%q: message does not name the positional argument: %q", args, combined)
			}
			// An envelope command answers on stdout with a stable code.
			if stdout.Len() > 0 {
				var envelope struct {
					OK    bool `json:"ok"`
					Error *struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
					t.Fatalf("%q: stdout is not one envelope: %v (%q)", args, err, stdout.String())
				}
				if envelope.OK || envelope.Error == nil || envelope.Error.Code != "INVALID_ARGUMENT" {
					t.Fatalf("%q: envelope is not an INVALID_ARGUMENT failure: %q", args, stdout.String())
				}
			}
		})
	}
}

// A run without a positional argument must keep working: the guard must not
// reject anything that used to succeed.
func TestValidateStillAcceptsAFlagOnlyInvocation(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	command := exec.Command(binary, "validate", "-workspace", referenceWorkspacePath())
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("validate without a positional argument failed: %v (stderr=%q)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Fatalf("validate did not report ok: %q", stdout.String())
	}
}
