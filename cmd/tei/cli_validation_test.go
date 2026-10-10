package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func readValidationReport(t *testing.T, b *bytes.Buffer) cliValidationReport {
	t.Helper()
	var report cliValidationReport
	if err := json.Unmarshal(b.Bytes(), &report); err != nil {
		t.Fatalf("not a single JSON document: %v: %q", err, b.String())
	}
	if report.SchemaVersion != "tei.cli.validate/v0" || report.Diagnostics == nil {
		t.Fatalf("invalid CLI result: %+v", report)
	}
	return report
}

func TestValidateWorkspaceJSONValid(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: valid\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/empty.tei", "namespace sales\n")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	report := readValidationReport(t, &out)
	if !report.OK || report.Error != nil || len(report.Diagnostics) != 0 {
		t.Fatalf("unexpected success report: %+v", report)
	}
}

func TestValidateWorkspaceJSONParseFailure(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: bad\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/order.tei", "namespace sales\n\ndata Order {\n    attribute orderId type }\n}\n")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 1 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	report := readValidationReport(t, &out)
	if report.OK || len(report.Diagnostics) != 1 || report.Diagnostics[0].Diagnostic.Code != "PARSE_ERROR" {
		t.Fatalf("unexpected parse report: %+v", report)
	}
	item := report.Diagnostics[0]
	if item.Diagnostic.Subject != nil || len(item.Sources) != 1 ||
		item.Sources[0].ArtifactPath != "semantic/sales/order.tei" ||
		item.Sources[0].Range.Start.Line != 4 {
		t.Fatalf("parse source lost: %+v", item)
	}
}

func TestValidateWorkspaceJSONSemanticFailure(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: bad\nsources:\n  - applications\n")
	writeWorkspaceFile(t, root, "applications/order-entry/application.tei", `application order_entry {
    database sales {
        target sales_core
        access invalid
    }
}
`)
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 1 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	report := readValidationReport(t, &out)
	found := false
	for _, item := range report.Diagnostics {
		if item.Diagnostic.Code == "INVALID_DATABASE_ACCESS" {
			found = true
			if len(item.Sources) == 0 || item.Sources[0].ArtifactPath != "applications/order-entry/application.tei" {
				t.Fatalf("source location missing: %+v", item)
			}
		}
	}
	if report.OK || !found {
		t.Fatalf("unexpected semantic report: %+v", report)
	}
}

func TestValidateWorkspaceJSONLoadFailure(t *testing.T) {
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, filepath.Join(t.TempDir(), "missing"))
	if err != nil || code != 2 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	report := readValidationReport(t, &out)
	if report.OK || report.Error == nil || report.Error.Code != "WORKSPACE_LOAD_FAILED" ||
		!strings.Contains(report.Error.Message, "missing") {
		t.Fatalf("unexpected load report: %+v", report)
	}
}
