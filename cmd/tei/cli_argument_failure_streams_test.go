package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// cliProcessStreams runs the built CLI and keeps stdout and stderr apart. The
// in-process tests call runVersionCommand with a bytes.Buffer, so they cannot
// show where main puts the error; a caller that piped the two streams
// together would still pass those tests.
type cliProcessStreams struct {
	stdout string
	stderr string
	code   int
}

func runCLIProcessWithStreams(t *testing.T, binary string, args ...string) cliProcessStreams {
	t.Helper()
	command := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("argv=%q: CLI execution: %v", args, err)
		}
		code = exit.ExitCode()
	}
	return cliProcessStreams{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// TestArgumentFailuresKeepStandardOutputMachineReadable pins the decision
// recorded for #2993: a command whose argument handling fails before it can
// answer writes the reason to standard error and leaves standard output empty,
// so a caller that asked for -json never parses a free-text line as JSON.
// `validate -json` is the contrast: it answers an argument failure with the
// envelope on standard output, which is why the rule has to be stated per
// command rather than inferred from one of them.
func TestArgumentFailuresKeepStandardOutputMachineReadable(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := buildCLIForLifecycle(t)

	cases := []struct {
		args    []string
		message string
	}{
		{args: []string{"version", "-x"}, message: `version accepts only -json, got "-x"`},
		{args: []string{"version", "-json", "-x"}, message: `version accepts only -json, got "-x"`},
		{args: []string{"commands", "extra"}, message: "commands does not accept arguments"},
	}
	for _, tc := range cases {
		got := runCLIProcessWithStreams(t, binary, tc.args...)
		if got.code != 2 {
			t.Fatalf("argv=%q: exit=%d (want 2)", tc.args, got.code)
		}
		if got.stdout != "" {
			t.Fatalf("argv=%q: standard output must stay empty, got %q", tc.args, got.stdout)
		}
		if !strings.Contains(got.stderr, tc.message) {
			t.Fatalf("argv=%q: standard error must explain the failure, got %q", tc.args, got.stderr)
		}
	}

	// The enveloped read commands answer argument failures on standard
	// output instead. Keeping this in the same test makes the difference
	// visible: neither stream rule is the CLI-wide default.
	got := runCLIProcessWithStreams(t, binary, "validate", "-json", "-nope")
	if got.code != 2 {
		t.Fatalf("validate -json unexpected argument: exit=%d", got.code)
	}
	if got.stdout == "" || got.stderr != "" {
		t.Fatalf("validate -json must answer on standard output only: stdout=%q stderr=%q", got.stdout, got.stderr)
	}
	var envelope struct {
		SchemaVersion string      `json:"schemaVersion"`
		OK            bool        `json:"ok"`
		Error         *cliFailure `json:"error"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &envelope); err != nil {
		t.Fatalf("validate -json argument failure is not one envelope: %v (%q)", err, got.stdout)
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("validate -json argument failure lost its code: %+v", envelope)
	}
}
