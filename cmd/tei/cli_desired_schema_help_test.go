package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// `desired-schema` is dispatched but deliberately absent from the read-only
// discovery catalog, so it has no allowlist entry. Its `-h` used to say only
// "run `tei commands`", which points at a list that does not contain it: an
// operator asking how to pass a format was told to read a catalog without the
// command (#5485).
//
// The expected flag names are written from the flags the command actually
// parses, not from the help text.
func TestDesiredSchemaHelpNamesItsOwnFlags(t *testing.T) {
	binary := buildCLIForHelpTest(t)
	out, err := exec.Command(binary, "desired-schema", "-h").CombinedOutput()
	// -h exits non-zero through flag.ExitOnError, which is fine: the text is
	// what this test reads.
	help := string(out)
	if err != nil && strings.TrimSpace(help) == "" {
		t.Fatalf("-h printed nothing: %v", err)
	}

	for _, want := range []string{"-workspace", "-database", "-provider", "-format"} {
		if !strings.Contains(help, want) {
			t.Errorf("desired-schema help does not name %q:\n%s", want, help)
		}
	}
	// Both accepted format values are stated, so a caller learns them without
	// reading the source.
	for _, want := range []string{"json", "sql"} {
		if !strings.Contains(help, want) {
			t.Errorf("the help does not state the %q format:\n%s", want, help)
		}
	}
	// The catalog pointer must not be the whole answer: this command is not in
	// that catalog.
	if strings.TrimSpace(help) == "usage: tei desired-schema [options]\nrun `tei commands` for the advertised commands" {
		t.Fatalf("desired-schema help still defers to the catalog alone:\n%s", help)
	}
}

// The command still works after stating its flags, so the help change did not
// alter what it accepts.
func TestDesiredSchemaStillWorksAfterStatingItsFlags(t *testing.T) {
	binary := buildCLIForHelpTest(t)
	root := filepath.Join("..", "..", "reference", "workspace")
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, "desired-schema", "-workspace", root,
		"-database", "sales_core", "-provider", "postgres", "-format", "sql")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("desired-schema -format sql failed: %v (%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CREATE TABLE") {
		t.Fatalf("-format sql did not print statements: %s", stdout.String())
	}
}

func buildCLIForHelpTest(t *testing.T) string {
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
