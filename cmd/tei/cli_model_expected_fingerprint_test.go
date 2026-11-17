package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

// runModelSnapshotCLI runs one model subcommand invocation, returning the exit
// code and the decoded failure code, if any. It follows the same shape as
// runInspectCLI: the real argv, stdout and exit code boundary is what carries
// the contract, not the helper functions.
func runModelSnapshotCLI(t *testing.T, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(sharedCLIBinary(t), args...)
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

// modelSnapshotFingerprintFor reads one snapshot fingerprint from the real
// binary, so the value the precondition is judged against is the one the CLI
// reports rather than one recomputed here.
func modelSnapshotFingerprintFor(t *testing.T, args ...string) string {
	t.Helper()
	var stdout bytes.Buffer
	cmd := exec.Command(sharedCLIBinary(t), args...)
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

// `model locate` and `model overview` return the same two fingerprints inspect
// returns, so a caller that reads a Workspace twice through either of them needs
// the same precondition. A stale value must fail closed with the operation's own
// code and the exit code these commands use for argument and precondition
// failures, and the current value must be accepted.
func TestModelLocateAndModelOverviewExpectedFingerprintPreconditionAtProcessBoundary(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "workspace")
	cases := []struct {
		name string
		base []string
	}{
		{"model locate", []string{"model", "locate", "-workspace", reference,
			"-subject-kind", "semantic-data", "-subject-id", "sales.order"}},
		{"model overview", []string{"model", "overview", "-workspace", reference}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if code, failure := runModelSnapshotCLI(t, testCase.base...); code != 0 || failure != "" {
				t.Fatalf("%s without a precondition: exit=%d code=%q",
					testCase.name, code, failure)
			}
			current := modelSnapshotFingerprintFor(t, testCase.base...)

			ok, failure := runModelSnapshotCLI(t, append(append([]string{}, testCase.base...),
				"-expected-fingerprint", current)...)
			if ok != 0 || failure != "" {
				t.Fatalf("%s refused the current snapshot fingerprint: exit=%d code=%q",
					testCase.name, ok, failure)
			}

			stale, failure := runModelSnapshotCLI(t, append(append([]string{}, testCase.base...),
				"-expected-fingerprint", "sha256:stale")...)
			if stale != 2 || failure != "STALE_SNAPSHOT" {
				t.Fatalf("%s stale precondition: exit=%d code=%q, want exit=2 code=STALE_SNAPSHOT",
					testCase.name, stale, failure)
			}

			// A trace continuation token is a different domain: passing one here
			// must not be read as agreement with this Workspace.
			traceCode, traceFailure := runModelSnapshotCLI(t, append(append([]string{}, testCase.base...),
				"-expected-fingerprint", inspectTraceTokenFor(t))...)
			if traceCode != 2 || traceFailure != "STALE_SNAPSHOT" {
				t.Fatalf("%s accepted a trace token as a snapshot fingerprint: exit=%d code=%q",
					testCase.name, traceCode, traceFailure)
			}

			// An explicitly empty value is not "omitted": the flag switches on a
			// comparison, so "" is refused rather than silently skipping it.
			empty, failure := runModelSnapshotCLI(t, append(append([]string{}, testCase.base...),
				"-expected-fingerprint", "")...)
			if empty != 2 || failure != "INVALID_ARGUMENT" {
				t.Fatalf("%s accepted an empty -expected-fingerprint: exit=%d code=%q",
					testCase.name, empty, failure)
			}
		})
	}
}

// One token for one Workspace, whichever of the four commands produced it, is
// what lets a caller chain reads without tracking which command it last called.
// inspect, model locate and model overview agree because they share one
// composer. trace is deliberately different: its token also binds the typed
// graph fingerprint, so it proves the projected relations did not change as well.
func TestInspectAndModelLocateAndModelOverviewReportOneSnapshotFingerprintAtProcessBoundary(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "workspace")
	inspect := modelSnapshotFingerprintFor(t, "inspect", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order")
	locate := modelSnapshotFingerprintFor(t, "model", "locate", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order")
	overview := modelSnapshotFingerprintFor(t, "model", "overview", "-workspace", reference)
	if locate != inspect {
		t.Fatalf("model locate reports %s, inspect reports %s", locate, inspect)
	}
	if overview != inspect {
		t.Fatalf("model overview reports %s, inspect reports %s", overview, inspect)
	}
	trace := modelSnapshotFingerprintFor(t, "trace", "-workspace", reference,
		"-scope", "reference-observation",
		"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
		"-mode", "summary")
	if trace == inspect {
		t.Fatal("trace reports the inspect snapshot fingerprint; its continuation token is expected to bind the graph as well")
	}
}
