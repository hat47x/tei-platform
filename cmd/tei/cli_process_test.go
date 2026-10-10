package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIProcessJSONContract(t *testing.T) {
	// Exercise the actual argv/exit/stdout boundary, not just helper functions.
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	run := func(expectedCode int, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		got := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("CLI execution: %v", err)
			}
			got = exit.ExitCode()
		}
		if got != expectedCode {
			t.Fatalf("argv=%q exit=%d want=%d stdout=%q stderr=%q", args, got, expectedCode, stdout.String(), stderr.String())
		}
		if stderr.Len() > 0 {
			t.Fatalf("argv=%q emitted stderr in the JSON/test path: %q", args, stderr.String())
		}
		return stdout.Bytes()
	}

	var catalog cliCommandCatalog
	if err := json.Unmarshal(run(0, "commands"), &catalog); err != nil {
		t.Fatalf("commands must return JSON: %v", err)
	}
	if catalog.Scope != "selected-read-only-commands" || len(catalog.Commands) == 0 {
		t.Fatalf("unexpected command catalogue: %+v", catalog)
	}

	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: cli_contract\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/empty.tei", "namespace sales\n")
	if got := string(run(0, "validate", "-workspace", root)); got != "ok\n" {
		t.Fatalf("text output compatibility: %q", got)
	}
	var report cliValidationReport
	if err := json.Unmarshal(run(0, "validate", "-workspace", root, "-json"), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.Error != nil {
		t.Fatalf("unexpected valid report: %+v", report)
	}
	for _, args := range [][]string{
		{"validate", "-json"},
		{"validate", "-workspace", root, "-world", "other.json", "-json"},
		{"validate", "-workspace", root, "-json", "unexpected"},
	} {
		var invalid cliValidationReport
		if err := json.Unmarshal(run(2, args...), &invalid); err != nil {
			t.Fatalf("invalid args %q did not produce JSON: %v", args, err)
		}
		if invalid.OK || invalid.Error == nil || invalid.Error.Code != "INVALID_ARGUMENT" {
			t.Fatalf("wrong invalid argument result: %+v", invalid)
		}
	}

	// Inspect operates over the same normalized Canonical World and Source
	// Index, but reports not-found and invalid requests as structured JSON.
	reference := filepath.Join("..", "..", "reference", "workspace")
	var inspected cliInspectionReport
	if err := json.Unmarshal(run(0, "inspect", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order"), &inspected); err != nil {
		t.Fatalf("inspect success must be JSON: %v", err)
	}
	if !inspected.OK || inspected.Result == nil || inspected.Result.SourceStatus != "indexed" ||
		inspected.Result.Subject.Kind != "semantic-data" || inspected.Result.Subject.ID != "sales.order" {
		t.Fatalf("unexpected inspect success: %+v", inspected)
	}
	var absent cliInspectionReport
	if err := json.Unmarshal(run(1, "inspect", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.not_found"), &absent); err != nil {
		t.Fatalf("inspect missing subject not JSON: %v", err)
	}
	if absent.OK || absent.Error == nil || absent.Error.Code != "SUBJECT_NOT_FOUND" {
		t.Fatalf("wrong missing subject result: %+v", absent)
	}
	var invalidInspect cliInspectionReport
	if err := json.Unmarshal(run(2, "inspect", "-workspace", reference, "-subject-kind", "semantic-data"), &invalidInspect); err != nil {
		t.Fatalf("inspect invalid argument not JSON: %v", err)
	}
	if invalidInspect.OK || invalidInspect.Error == nil || invalidInspect.Error.Code != "INVALID_SUBJECT" {
		t.Fatalf("wrong inspect invalid argument result: %+v", invalidInspect)
	}

	// Exercise argv, JSON, stdout/stderr and exit code for typed trace.
	var traceSummary cliTraceReport
	if err := json.Unmarshal(run(0,
		"trace", "-workspace", reference, "-scope", "reference-observation",
		"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
		"-mode", "summary"), &traceSummary); err != nil {
		t.Fatalf("trace summary must be JSON: %v", err)
	}
	if !traceSummary.OK || traceSummary.Result == nil ||
		traceSummary.Result.Trace.Total < 2 || len(traceSummary.Result.Trace.Rows) != 0 {
		t.Fatalf("unexpected typed trace summary: %+v", traceSummary)
	}
	var inspectRelation cliInspectionReport
	if err := json.Unmarshal(run(0,
		"inspect", "-workspace", reference,
		"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
	), &inspectRelation); err != nil {
		t.Fatalf("inspect relation must be JSON: %v", err)
	}
	if !inspectRelation.OK || inspectRelation.Result == nil ||
		inspectRelation.Result.Subject != traceSummary.Result.Trace.Subject.Subject {
		t.Fatalf("inspect and trace disagree on subject: inspect=%+v trace=%+v", inspectRelation, traceSummary)
	}
	if len(traceSummary.Result.Source.References) == 0 ||
		len(inspectRelation.Result.Sources.References) == 0 ||
		traceSummary.Result.Source.References[0].ArtifactPath !=
			inspectRelation.Result.Sources.References[0].ArtifactPath {
		t.Fatalf("inspect and trace disagree on provenance: inspect=%+v trace=%+v", inspectRelation.Result.Sources, traceSummary.Result.Source)
	}
	var traceFull cliTraceReport
	if err := json.Unmarshal(run(0,
		"trace", "-workspace", reference, "-scope", "reference-observation",
		"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
		"-mode", "full", "-limit", "1",
		"-expected-fingerprint", traceSummary.Result.SnapshotFingerprint), &traceFull); err != nil {
		t.Fatalf("trace full must be JSON: %v", err)
	}
	if !traceFull.OK || traceFull.Result == nil ||
		len(traceFull.Result.Trace.Rows) != 1 ||
		traceFull.Result.Trace.NextOffset == nil {
		t.Fatalf("wrong trace full result: %+v", traceFull)
	}
	var traceStale cliTraceReport
	if err := json.Unmarshal(run(1,
		"trace", "-workspace", reference, "-scope", "reference-observation",
		"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
		"-mode", "full", "-limit", "1", "-offset", "1",
		"-expected-fingerprint", "sha256:stale"), &traceStale); err != nil {
		t.Fatalf("trace stale must be JSON: %v", err)
	}
	if traceStale.OK || traceStale.Error == nil ||
		traceStale.Error.Code != "STALE_SNAPSHOT" {
		t.Fatalf("incorrect stale result: %+v", traceStale)
	}

	writeWorkspaceFile(t, root, "semantic/sales/empty.tei", "namespace sales\n\ndata Order {\n    attribute orderId type }\n}\n")
	var broken cliValidationReport
	if err := json.Unmarshal(run(1, "validate", "-workspace", root, "-json"), &broken); err != nil {
		t.Fatal(err)
	}
	if broken.OK || len(broken.Diagnostics) == 0 || broken.Diagnostics[0].Diagnostic.Code != "PARSE_ERROR" {
		t.Fatalf("parse diagnostics missing: %+v", broken)
	}
	// A parse failure has source coordinates, never a fabricated Subject.
	if broken.Diagnostics[0].Diagnostic.Subject != nil ||
		len(broken.Diagnostics[0].Sources) == 0 ||
		!strings.HasSuffix(broken.Diagnostics[0].Sources[0].ArtifactPath, "empty.tei") {
		t.Fatalf("invalid parse source projection: %+v", broken.Diagnostics[0])
	}
}

func TestValidateJSONDoesNotDependOnOutputFile(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: cli_contract\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/empty.tei", "namespace sales\n")
	var result bytes.Buffer
	code, err := validateWorkspaceJSON(&result, root)
	if err != nil || code != 0 || !json.Valid(result.Bytes()) {
		t.Fatalf("JSON validation failure: %d %v %q", code, err, result.String())
	}
	// JSON mode is a read-only view, not a hidden write of a derived artifact.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected write in workspace root: %v", entries)
	}
}
