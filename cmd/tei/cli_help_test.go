package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// TestHelpIsASuccessOnStandardOutput pins #3185: `-h` and `-help` ask the CLI to
// explain itself, so they must not be reported as argument failures. Before
// this, inspect/view/knowledge-qualify answered `-h` with INVALID_ARGUMENT and
// exit 2, which made the most ordinary exploration indistinguishable from a
// typo.
func TestHelpIsASuccessOnStandardOutput(t *testing.T) {
	cases := []struct {
		name string
		run  func(out *bytes.Buffer) (int, error)
		want string
	}{
		{
			name: "version",
			run: func(out *bytes.Buffer) (int, error) {
				return runVersionCommand([]string{"-h"}, out)
			},
			want: "-json",
		},
		{
			name: "model catalog",
			run: func(out *bytes.Buffer) (int, error) {
				return runModelCommand([]string{"catalog", "-h"}, out)
			},
			want: "-namespace",
		},
		{
			name: "model locate",
			run: func(out *bytes.Buffer) (int, error) {
				return runModelCommand([]string{"locate", "-h"}, out)
			},
			want: "-subject-kind",
		},
		{
			name: "model overview",
			run: func(out *bytes.Buffer) (int, error) {
				return runModelCommand([]string{"overview", "-h"}, out)
			},
			want: "-limit",
		},
		{
			name: "view",
			run: func(out *bytes.Buffer) (int, error) {
				return runViewCommand([]string{"-h"}, out)
			},
			want: "-allow-capability",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			code, err := tc.run(&out)
			if err != nil || code != 0 {
				t.Fatalf("exit=%d err=%v out=%q", code, err, out.String())
			}
			if !strings.HasPrefix(out.String(), "usage: tei ") {
				t.Fatalf("help does not start with a usage line: %q", out.String())
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("help omits %q: %q", tc.want, out.String())
			}
		})
	}
}

// TestHelpAtTheProcessBoundary keeps the two promises that only argv can show:
// the listing is a success with empty standard error, and a genuinely bad flag
// is still a failure. A test that only called the helper could pass while main
// still exited 2.
func TestHelpAtTheProcessBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := buildCLIForLifecycle(t)
	run := func(args ...string) (string, string, int) {
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
		return stdout.String(), stderr.String(), code
	}

	for _, args := range [][]string{
		{"inspect", "-h"},
		{"inspect", "-help"},
		{"view", "-h"},
		{"trace", "-h"},
		{"workspace-files", "-h"},
		{"knowledge-qualify", "-h"},
		{"commands", "-h"},
		{"version", "-h"},
		{"model", "catalog", "-h"},
	} {
		stdout, stderr, code := run(args...)
		if code != 0 {
			t.Fatalf("argv=%q exit=%d (want 0) stdout=%q stderr=%q", args, code, stdout, stderr)
		}
		if stderr != "" {
			t.Fatalf("argv=%q wrote stderr=%q", args, stderr)
		}
		if !strings.HasPrefix(stdout, "usage: tei ") {
			t.Fatalf("argv=%q help=%q", args, stdout)
		}
	}

	// The contrast: an unknown flag is still an argument failure. `inspect`
	// answers it with the envelope on standard output, `commands` with a
	// free-text standard error, and neither may turn into exit 0.
	for _, args := range [][]string{
		{"inspect", "-not-a-flag"},
		{"view", "-not-a-flag"},
		{"version", "-not-a-flag"},
		{"commands", "extra"},
	} {
		stdout, stderr, code := run(args...)
		if code != 2 {
			t.Fatalf("argv=%q exit=%d (want 2)", args, code)
		}
		if stdout == "" && stderr == "" {
			t.Fatalf("argv=%q failed without a diagnostic", args)
		}
	}
}
