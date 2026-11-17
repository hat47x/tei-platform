package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// `desired-schema` renders two things: the JSON snapshot and the SQL that
// creates it. An unknown `-format` used to fall through to the JSON branch, so
// `-format bogus` printed a snapshot with exit 0 — the caller asked for
// something the command does not have and was told it succeeded (#5408).
//
// The check has to run before the Workspace is loaded, so this test uses the
// real binary: an unknown value must be refused even when nothing else about
// the invocation is wrong.
func TestDesiredSchemaRefusesAnUnknownFormat(t *testing.T) {
	binary := buildCLIForFormatTest(t)
	for _, format := range []string{"bogus", "SQL", "text", "html"} {
		t.Run(format, func(t *testing.T) {
			cmd := exec.Command(binary,
				"desired-schema",
				"-workspace", filepath.Join("..", "..", "reference", "workspace"),
				"-database", "sales_core",
				"-provider", "postgres",
				"-format", format,
			)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err == nil {
				t.Fatalf("-format %q was accepted; stdout=%s", format, stdout.String())
			}
			// Nothing was printed on stdout: a refused format must not also
			// emit a snapshot the caller did not ask for.
			if strings.TrimSpace(stdout.String()) != "" {
				t.Fatalf("-format %q printed output before refusing: %s", format, stdout.String())
			}
			// The message names the value the caller used, so they know what
			// to change.
			if !strings.Contains(stderr.String(), format) {
				t.Fatalf("the refusal does not name %q: %s", format, stderr.String())
			}
			if !strings.Contains(stderr.String(), "json or sql") {
				t.Fatalf("the refusal does not state the accepted values: %s", stderr.String())
			}
		})
	}
}

// The two accepted values keep working, so the guard is a narrowing and not a
// break.
func TestDesiredSchemaStillAcceptsJSONAndSQL(t *testing.T) {
	binary := buildCLIForFormatTest(t)
	root := filepath.Join("..", "..", "reference", "workspace")

	jsonCmd := exec.Command(binary, "desired-schema", "-workspace", root,
		"-database", "sales_core", "-provider", "postgres", "-format", "json")
	jsonOut, err := jsonCmd.Output()
	if err != nil {
		t.Fatalf("-format json failed: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(jsonOut)), "{") {
		t.Fatalf("-format json did not print a snapshot: %s", jsonOut)
	}

	sqlCmd := exec.Command(binary, "desired-schema", "-workspace", root,
		"-database", "sales_core", "-provider", "postgres", "-format", "sql")
	sqlOut, err := sqlCmd.Output()
	if err != nil {
		t.Fatalf("-format sql failed: %v", err)
	}
	if !strings.Contains(string(sqlOut), "CREATE TABLE") {
		t.Fatalf("-format sql did not print statements: %s", sqlOut)
	}
}

func buildCLIForFormatTest(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}
