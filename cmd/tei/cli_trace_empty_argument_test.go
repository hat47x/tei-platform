package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runTraceCLI builds the CLI and runs one trace invocation, returning the exit
// code and the decoded failure code, if any.
func runTraceCLI(t *testing.T, args ...string) (int, string) {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, isExit := err.(*exec.ExitError); isExit {
			code = exit.ExitCode()
		} else {
			t.Fatalf("CLI execution: %v", err)
		}
	}
	var doc struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("argv=%q did not return one JSON object: %v (%q)", args, err, stdout.String())
	}
	if doc.Error != nil {
		return code, doc.Error.Code
	}
	if !doc.OK {
		t.Fatalf("argv=%q: neither ok nor a failure code (%q)", args, stdout.String())
	}
	return code, ""
}

// An explicitly empty -direction/-mode/-expected-fingerprint is not the same
// request as omitting the flag. The CLI advertises mode=summary, but the typed
// normalizer turns "" into mode=full, which returns rows the caller did not ask
// for; the MCP adapter already rejects the empty string with -32602 (#3605).
func TestTraceRejectsExplicitlyEmptyArguments(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	base := []string{"trace", "-workspace", reference, "-scope", "r8",
		"-subject-kind", "semantic-data", "-subject-id", "sales.order"}

	for _, flag := range []string{"-direction", "-mode", "-expected-fingerprint"} {
		args := append(append([]string{}, base...), flag, "")
		code, failure := runTraceCLI(t, args...)
		if code != 2 {
			t.Fatalf("%s empty: exit=%d, want 2", flag, code)
		}
		if failure != "INVALID_ARGUMENT" {
			t.Fatalf("%s empty: failure=%q, want INVALID_ARGUMENT", flag, failure)
		}
	}
}

// The omitted flag keeps the advertised default. This test guards against
// "fixing" the empty case by removing the default altogether.
func TestTraceOmittedArgumentsUseTheAdvertisedDefaults(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	code, failure := runTraceCLI(t, "trace", "-workspace", reference, "-scope", "r8",
		"-subject-kind", "semantic-data", "-subject-id", "sales.order")
	if code != 0 || failure != "" {
		t.Fatalf("omitted flags: exit=%d failure=%q", code, failure)
	}
}
