package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
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

func TestValidateWorkspaceJSONDuplicateHasSubjectButNoAmbiguousSource(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: duplicate\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/intake/request.tei", "namespace intake\ndata Request {\n    attribute status type std.code\n    attribute status type std.code\n}\n")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 1 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	report := readValidationReport(t, &out)
	for _, item := range report.Diagnostics {
		if item.Diagnostic.Code != "DUPLICATE_SEMANTIC_ELEMENT" {
			continue
		}
		if item.Diagnostic.Subject == nil || item.Diagnostic.Subject.ID != "intake.request.status" || len(item.Sources) != 0 {
			t.Fatalf("duplicate diagnostic: %+v", item)
		}
		return
	}
	t.Fatalf("duplicate diagnostic missing: %+v", report)
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

func validateFingerprints(t *testing.T, root string) (string, string) {
	t.Helper()
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	report := readValidationReport(t, &out)
	if report.WorldFingerprint == "" || report.SourceIndexFingerprint == "" {
		t.Fatalf("fingerprints missing: %+v", report)
	}
	return report.WorldFingerprint, report.SourceIndexFingerprint
}

// The fingerprints bind a validate result to the version that inspect and
// model catalog read from the same files.
func TestValidateWorkspaceJSONFingerprintsMatchModelCatalog(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	worldFP, sourceFP := validateFingerprints(t, root)
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if worldFP != catalog.WorldFingerprint || sourceFP != catalog.SourceIndexFingerprint {
		t.Fatalf("validate=(%s,%s) catalog=(%s,%s)", worldFP, sourceFP,
			catalog.WorldFingerprint, catalog.SourceIndexFingerprint)
	}
}

func TestValidateWorkspaceJSONFingerprintsChangeWithContent(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: valid\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/empty.tei", "namespace sales\n")
	world1, source1 := validateFingerprints(t, root)
	again1, againSource1 := validateFingerprints(t, root)
	if world1 != again1 || source1 != againSource1 {
		t.Fatal("fingerprints are not stable for unchanged files")
	}
	writeWorkspaceFile(t, root, "semantic/sales/empty.tei", "namespace sales\n\ndata Order {\n}\n")
	world2, source2 := validateFingerprints(t, root)
	if world1 == world2 || source1 == source2 {
		t.Fatalf("fingerprints did not change: world %s -> %s, source %s -> %s", world1, world2, source1, source2)
	}
}

// No version exists when a syntax finding or a load failure stops the read, so
// the fingerprints are omitted from the JSON instead of being empty strings.
func TestValidateWorkspaceJSONOmitsFingerprintsWithoutVersion(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: bad\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/order.tei", "namespace sales\n\ndata Order {\n    attribute orderId type }\n}\n")
	for name, dir := range map[string]string{"parse": root, "load": filepath.Join(t.TempDir(), "missing")} {
		var out bytes.Buffer
		if _, err := validateWorkspaceJSON(&out, dir); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "\"worldFingerprint\"") || strings.Contains(out.String(), "\"sourceIndexFingerprint\"") {
			t.Fatalf("%s: fingerprint must be omitted: %s", name, out.String())
		}
	}
}

// A release whose application, database and contract have no current
// definition yields warnings only. The text form prints them before "ok" and
// exits 0, so callers must use the exit code (or JSON ok), not stdout == "ok".
func TestValidateWorkspaceWarningOnlyContract(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: warn\nsources:\n  - releases\n")
	writeWorkspaceFile(t, root, "releases/applications/order-entry/1.0.0.tei", `application-release order_entry 1.0.0 {
    source-revision "warning-only"

    database sales {
        target sales_core
        access read-write
        requires {
            contract sales_core.order_legacy ^1
        }
    }
}
`)
	var jsonOut bytes.Buffer
	code, err := validateWorkspaceJSON(&jsonOut, root)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, jsonOut.String())
	}
	report := readValidationReport(t, &jsonOut)
	if !report.OK || len(report.Diagnostics) == 0 {
		t.Fatalf("want ok=true with warnings: %+v", report)
	}
	for _, item := range report.Diagnostics {
		if item.Diagnostic.Severity != "warning" {
			t.Fatalf("unexpected non-warning: %+v", item)
		}
	}
	var text bytes.Buffer
	code, err = validateWorkspace(&text, root)
	if err != nil || code != 0 {
		t.Fatalf("text exit=%d err=%v body=%q", code, err, text.String())
	}
	if text.String() == "ok\n" || !strings.HasSuffix(text.String(), "\nok\n") ||
		!strings.HasPrefix(text.String(), "warning\t") {
		t.Fatalf("warning-only text output changed: %q", text.String())
	}
}
