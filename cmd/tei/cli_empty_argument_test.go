package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// emptyStringFlagSet builds a FlagSet holding the command's advertised flags
// with the same defaults the CLI registers, so the check runs against the real
// grammar rather than a toy set.
func emptyStringFlagSet(command string, defaults map[string]string) *flag.FlagSet {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	for _, name := range selectedReadOnlyFlagAllowlist[command] {
		value, ok := defaults[name]
		if !ok {
			value = ""
		}
		fs.String(name, value, "")
	}
	return fs
}

// TestEmptyStringArgumentsAreRejected pins #3605: an explicit "" is a different
// claim from an omitted flag. Before this, `trace -mode ""` fell through to
// "full" inside the operation while the advertised default is "summary",
// so a caller that thought it was using the default observed a different
// result shape.
func TestEmptyStringArgumentsAreRejected(t *testing.T) {
	guarded := map[string][]string{
		"trace":             {"mode", "direction", "expected-fingerprint", "scope", "subject-kind", "subject-id"},
		"inspect":           {"subject-kind", "subject-id"},
		"source-index":      {"subject-kind"},
		"knowledge-context": {"disclose"},
		"plugin-lock":       {"environment"},
	}
	defaults := map[string]string{"direction": "both", "mode": "summary"}
	for command, names := range guarded {
		for _, name := range names {
			t.Run(command+" -"+name, func(t *testing.T) {
				fs := emptyStringFlagSet(command, defaults)
				if err := fs.Parse([]string{"-" + name, ""}); err != nil {
					t.Fatalf("parse: %v", err)
				}
				err := validateSelectedReadOnlyFlags(fs, command)
				if err == nil {
					t.Fatal("an explicit empty string was accepted")
				}
				if !strings.Contains(err.Error(), "-"+name) ||
					!strings.Contains(err.Error(), "must not be an empty string") {
					t.Fatalf("message does not name the flag and the rule: %v", err)
				}
			})
		}
	}
}

// Omitting the flag must keep its documented default, so the rule only rejects
// a value the caller actually supplied.
func TestOmittedArgumentsKeepTheirDefault(t *testing.T) {
	for _, command := range []string{"trace", "inspect", "source-index"} {
		fs := emptyStringFlagSet(command, map[string]string{"direction": "both", "mode": "summary"})
		// Only the required selectors are supplied; the optional ones stay omitted.
		args := []string{"-workspace", filepath.Join(testRepositoryRoot(t), "reference", "workspace")}
		if command == "trace" {
			args = append(args, "-scope", "s", "-subject-kind", "semantic-data", "-subject-id", "sales.order")
		} else {
			args = append(args, "-subject-kind", "semantic-data", "-subject-id", "sales.order")
		}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("%s parse: %v", command, err)
		}
		if err := validateSelectedReadOnlyFlags(fs, command); err != nil {
			t.Fatalf("%s: an omitted flag was rejected: %v", command, err)
		}
	}
}

// The model catalogue parses its own FlagSet, so the shared check cannot see
// -namespace. The CLI applies the same rule there; this fixes the message and
// the exit code of that path.
func TestModelCatalogRejectsAnEmptyNamespace(t *testing.T) {
	var out bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", filepath.Join(testRepositoryRoot(t), "reference", "workspace"), "-namespace", ""}, &out)
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if code != 2 {
		t.Fatalf("code=%d, want 2 (invalid argument)", code)
	}
	var response struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("catalog answer is not JSON: %v (%q)", err, out.String())
	}
	if response.OK || response.Error == nil || response.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("catalog failure = %+v", response)
	}
	if !strings.Contains(response.Error.Message, "-namespace must not be an empty string") {
		t.Fatalf("message does not state the rule: %q", response.Error.Message)
	}
}

// Omitting -namespace still means "no filter", so the check must not turn the
// documented default into a required flag.
func TestModelCatalogAcceptsAnOmittedNamespace(t *testing.T) {
	var out bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", filepath.Join(testRepositoryRoot(t), "reference", "workspace")}, &out)
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if code != 0 {
		t.Fatalf("code=%d want 0: %q", code, out.String())
	}
	var response struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil || !response.OK {
		t.Fatalf("catalog did not succeed: %v (%q)", err, out.String())
	}
}

// The built CLI must apply the rule end to end, and the corrected value must
// not be reachable through any other spelling.
func TestEmptyStringArgumentsInBuiltProcess(t *testing.T) {
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
	run := func(args ...string) (code int, stdout string) {
		t.Helper()
		command := exec.Command(binary, args...)
		var out bytes.Buffer
		command.Stdout = &out
		err := command.Run()
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), out.String()
		} else if err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
		return 0, out.String()
	}
	base := []string{"trace", "-workspace", filepath.Join(testRepositoryRoot(t), "reference", "workspace"), "-scope", "r8",
		"-subject-kind", "semantic-data", "-subject-id", "sales.order"}

	// The omitted flag keeps the advertised default.
	code, stdout := run(base...)
	if code != 0 || !strings.Contains(stdout, `"mode":"summary"`) {
		t.Fatalf("omitted -mode no longer uses the default: exit=%d %q", code, stdout)
	}
	// The explicit empty string is refused with exit 2 and INVALID_ARGUMENT.
	for _, flag := range []string{"-mode", "-direction", "-expected-fingerprint"} {
		code, stdout := run(append(base, flag, "")...)
		if code != 2 || !strings.Contains(stdout, "INVALID_ARGUMENT") ||
			!strings.Contains(stdout, flag+" must not be an empty string") {
			t.Fatalf("%s empty string: exit=%d %q", flag, code, stdout)
		}
	}
	// model catalog uses its own FlagSet; the rule still applies.
	if code, stdout := run("model", "catalog", "-workspace", filepath.Join(testRepositoryRoot(t), "reference", "workspace"), "-namespace", ""); code != 2 ||
		!strings.Contains(stdout, "INVALID_ARGUMENT") {
		t.Fatalf("model catalog -namespace empty: exit=%d %q", code, stdout)
	}
}
