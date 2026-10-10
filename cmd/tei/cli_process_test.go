package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIProcessJSONContract(t *testing.T) {
	// Exercise the actual argv/exit/stdout boundary, not just helper functions.
	binary := filepath.Join(t.TempDir(), "tei")
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
