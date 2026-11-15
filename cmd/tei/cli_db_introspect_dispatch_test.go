package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// An accepted db-introspect command must reach the existing read-only
// database implementation without first demanding a TEI Workspace or World.
// These negative cases deliberately connect to no real database.
func TestDBIntrospectCLIDispatchAndArgumentRefusals(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	cases := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"missing provider and dsn", []string{"db-introspect"}, 2, "requires -provider and -dsn"},
		{"missing dsn", []string{"db-introspect", "-provider", "postgres"}, 2, "requires -provider and -dsn"},
		{"unexpected positional", []string{"db-introspect", "-provider", "sqlite", "-dsn", ":memory:", "extra"}, 2, "accepts no positional arguments"},
		{"unsupported provider", []string{"db-introspect", "-provider", "invalid", "-dsn", "not-a-database"}, 1, "unsupported database provider"},
		{"semantic flag is parsed", []string{"db-introspect", "-provider", "invalid", "-dsn", "not-a-database", "-semantic"}, 1, "unsupported database provider"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binary, tc.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != tc.code {
				t.Fatalf("argv=%q exit=%v, want %d; stderr=%q", tc.args, err, tc.code, stderr.String())
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("argv=%q stdout=%q stderr=%q, want %q", tc.args, stdout.String(), stderr.String(), tc.want)
			}
			if strings.Contains(stderr.String(), "exactly one of -workspace or -world is required") {
				t.Fatalf("db-introspect attempted to load a workspace: %q", stderr.String())
			}
		})
	}
}
