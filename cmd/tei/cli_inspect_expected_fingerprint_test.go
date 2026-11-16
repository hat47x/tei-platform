package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runInspectCLI builds the CLI and runs one inspect invocation, returning the
// exit code and the decoded failure code, if any. It follows the same shape as
// runTraceCLI: the real argv, stdout and exit code boundary is what carries the
// contract, not the helper functions.
func runInspectCLI(t *testing.T, args ...string) (int, string) {
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
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result *struct {
			SnapshotFingerprint string `json:"snapshotFingerprint"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("argv=%q did not return one JSON object: %v (%q)", args, err, stdout.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("argv=%q emitted stderr in the JSON path: %q", args, stderr.String())
	}
	if doc.Error != nil {
		return code, doc.Error.Code
	}
	if !doc.OK || doc.Result == nil || doc.Result.SnapshotFingerprint == "" {
		t.Fatalf("argv=%q: neither ok nor a failure code (%q)", args, stdout.String())
	}
	return code, ""
}

// `-expected-fingerprint` is the precondition that lets a caller prove two
// reads of one subject saw the same Workspace. A stale value must fail closed
// with the operation's own code and the exit code inspect uses for argument and
// precondition failures, and the current value must be accepted.
func TestInspectExpectedFingerprintPreconditionAtProcessBoundary(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "workspace")
	base := []string{"inspect", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order"}

	code, failure := runInspectCLI(t, base...)
	if code != 0 || failure != "" {
		t.Fatalf("inspect without a precondition: exit=%d code=%q", code, failure)
	}
	current := inspectSnapshotFingerprintFor(t, base...)

	ok, failure := runInspectCLI(t, append(append([]string{}, base...),
		"-expected-fingerprint", current)...)
	if ok != 0 || failure != "" {
		t.Fatalf("the current snapshot fingerprint was refused: exit=%d code=%q", ok, failure)
	}

	stale, failure := runInspectCLI(t, append(append([]string{}, base...),
		"-expected-fingerprint", "sha256:stale")...)
	if stale != 2 || failure != "STALE_SNAPSHOT" {
		t.Fatalf("stale precondition: exit=%d code=%q, want exit=2 code=STALE_SNAPSHOT", stale, failure)
	}

	// A trace continuation token is a different domain: passing one here must not
	// be read as agreement with an inspect snapshot.
	traceCode, traceFailure := runInspectCLI(t, append(append([]string{}, base...),
		"-expected-fingerprint", inspectTraceTokenFor(t))...)
	if traceCode != 2 || traceFailure != "STALE_SNAPSHOT" {
		t.Fatalf("a trace token was accepted as an inspect snapshot: exit=%d code=%q",
			traceCode, traceFailure)
	}

	// The flag is advertised, so it is accepted by the shared allowlist rather
	// than rejected as unrelated to inspect.
	advertised, failure := runInspectCLI(t, append(append([]string{}, base...),
		"-expected-fingerprint", "")...)
	if advertised != 2 || failure != "INVALID_ARGUMENT" {
		t.Fatalf("-expected-fingerprint is not an accepted inspect flag: exit=%d code=%q",
			advertised, failure)
	}
}

// inspectSnapshotFingerprintFor reads one snapshot fingerprint from the real
// binary, so the value the precondition is judged against is the one the CLI
// reports rather than one recomputed here.
func inspectSnapshotFingerprintFor(t *testing.T, args ...string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	var stdout bytes.Buffer
	cmd := exec.Command(binary, args...)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("argv=%q: %v", args, err)
	}
	var doc struct {
		Result *struct {
			SnapshotFingerprint string `json:"snapshotFingerprint"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("argv=%q is not one JSON object: %v (%q)", args, err, stdout.String())
	}
	if doc.Result == nil || doc.Result.SnapshotFingerprint == "" {
		t.Fatalf("argv=%q reported no snapshot fingerprint: %q", args, stdout.String())
	}
	return doc.Result.SnapshotFingerprint
}

func inspectTraceTokenFor(t *testing.T) string {
	t.Helper()
	return inspectSnapshotFingerprintFor(t, "trace",
		"-workspace", filepath.Join("..", "..", "reference", "workspace"),
		"-scope", "reference-observation",
		"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
		"-mode", "summary")
}
