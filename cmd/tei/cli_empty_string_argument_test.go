package main

import (
	"bytes"
	"errors"
	"flag"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// #3605: an explicitly empty string flag used to mean "the caller omitted it",
// so a value the CLI never advertised as a default was applied. The reference
// MCP adapter already rejects an empty member, so the two entry points split.
// These tests pin the CLI side of that rule for the flags whose empty value
// selects an outcome that matches neither the advertised default nor "absent".
var rejectEmptyStringCases = []struct {
	name string
	args []string
	code string
}{
	{name: "trace mode", args: []string{"trace", "-workspace", "w", "-scope", "s", "-subject-kind", "k", "-subject-id", "i", "-mode="}, code: "INVALID_ARGUMENT"},
	{name: "trace direction", args: []string{"trace", "-workspace", "w", "-scope", "s", "-subject-kind", "k", "-subject-id", "i", "-direction="}, code: "INVALID_ARGUMENT"},
	{name: "trace expected-fingerprint", args: []string{"trace", "-workspace", "w", "-scope", "s", "-subject-kind", "k", "-subject-id", "i", "-expected-fingerprint="}, code: "INVALID_ARGUMENT"},
	{name: "model catalog namespace", args: []string{"model", "catalog", "-workspace", "w", "-namespace="}, code: "INVALID_ARGUMENT"},
	{name: "inspect subject-kind", args: []string{"inspect", "-workspace", "w", "-subject-kind=", "-subject-id", "i"}, code: "INVALID_ARGUMENT"},
}

// The rejection happens in a real process: the rule is applied where the
// command's flags are parsed, not inside each command's own argument check.
func TestExplicitEmptyStringArgumentsAreRejectedInARealProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	for _, tc := range rejectEmptyStringCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binary, tc.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 2 {
				t.Fatalf("want exit 2 for %v, got %v (stderr=%s)", tc.args, err, stderr.String())
			}
			if !strings.Contains(stdout.String(), `"code":"`+tc.code+`"`) {
				t.Fatalf("rejection is not the documented code %s: %s", tc.code, stdout.String())
			}
			if !strings.Contains(stdout.String(), "must not be an empty string") {
				t.Fatalf("rejection does not name the rule: %s", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("rejection wrote to standard error: %q", stderr.String())
			}
		})
	}
}

// Omitting the flag must keep working: the rule rejects an explicit empty
// value, not the absence of the flag.
func TestOmittedTraceFlagsKeepTheirDefaults(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	code, err := traceWorkspaceJSON(&out, root, "", "r8", "semantic-data", "sales.order", "both", "summary", "", 0, 0, 0)
	if err != nil || code != 0 {
		t.Fatalf("trace with the defaults: code=%d err=%v out=%s", code, err, out.String())
	}
	if !strings.Contains(out.String(), `"mode":"summary"`) {
		t.Fatalf("the default mode was not applied: %s", out.String())
	}
}

// The helper must look at the flags the caller passed, not at every string
// flag that happens to hold "". A flag left out has its declared default.
func TestEmptyStringFlagOnlyReportsProvidedFlags(t *testing.T) {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	mode := fs.String("mode", "summary", "")
	fs.String("direction", "both", "")
	fs.String("expected-fingerprint", "", "")
	if err := fs.Parse(nil); err != nil {
		t.Fatal(err)
	}
	if name := emptyStringFlag(fs, emptyStringFlagsNotAllowed["trace"]); name != "" {
		t.Fatalf("omitted flags were reported as empty: %q", name)
	}
	// An empty default is only a problem when the caller passes it explicitly.
	fs = flag.NewFlagSet("trace", flag.ContinueOnError)
	fs.String("mode", *mode, "")
	fs.String("direction", "both", "")
	fs.String("expected-fingerprint", "", "")
	if err := fs.Parse([]string{"-expected-fingerprint="}); err != nil {
		t.Fatal(err)
	}
	if name := emptyStringFlag(fs, emptyStringFlagsNotAllowed["trace"]); name != "expected-fingerprint" {
		t.Fatalf("explicitly empty flag was not reported: %q", name)
	}
}
