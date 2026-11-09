package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/validation"
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
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: valid\nformatVersion: 1\nsources:\n  - semantic\n")
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
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: bad\nformatVersion: 1\nsources:\n  - semantic\n")
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
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: bad\nformatVersion: 1\nsources:\n  - applications\n")
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

// TestValidateWorkspaceJSONProposesWithoutAccepting checks that diagnostics are
// paired with proposals bound to the read revision, and that the report never
// marks one accepted: acceptance is a separate human decision.
func TestValidateWorkspaceJSONProposesWithoutAccepting(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: bad\nformatVersion: 1\nsources:\n  - applications\n")
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
	if len(report.Proposals) != len(report.Diagnostics) {
		t.Fatalf("diagnostics=%d proposals=%d", len(report.Diagnostics), len(report.Proposals))
	}
	for _, proposal := range report.Proposals {
		if proposal.Accepted {
			t.Fatalf("proposal was accepted without a decision: %+v", proposal)
		}
		if proposal.Revision != report.WorldFingerprint {
			t.Fatalf("proposal revision=%q want=%q", proposal.Revision, report.WorldFingerprint)
		}
	}
	if err := validation.Accept(&report.Proposals[0], "rev-changed"); err == nil {
		t.Fatal("a changed revision accepted the proposal")
	}
	report.Proposals = nil
	out.Reset()
	if _, err := validateWorkspaceJSON(&out, root); err != nil {
		t.Fatal(err)
	}
}

func TestValidateWorkspaceJSONDuplicateHasDeclarationCandidates(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: duplicate\nformatVersion: 1\nsources:\n  - semantic\n")
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
		if item.Diagnostic.Subject == nil || item.Diagnostic.Subject.ID != "intake.request.status" || len(item.Sources) != 2 ||
			item.Sources[0].Range.Start.Line != 3 || item.Sources[1].Range.Start.Line != 4 ||
			item.Sources[0].Subject == nil || item.Sources[0].Subject.Kind != "semantic-attribute" ||
			item.Sources[1].Subject == nil || item.Sources[1].Subject.Kind != "semantic-attribute" {
			t.Fatalf("duplicate diagnostic: %+v", item)
		}
		return
	}
	t.Fatalf("duplicate diagnostic missing: %+v", report)
}

func TestValidateWorkspaceJSONCrossKindDuplicateLabelsCandidates(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: duplicate\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/a.tei", "namespace intake\ntype Request from std.code\n")
	writeWorkspaceFile(t, root, "semantic/b.tei", "namespace intake\ndata Request {}\n")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 1 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	for _, item := range readValidationReport(t, &out).Diagnostics {
		if item.Diagnostic.Code == "DUPLICATE_SEMANTIC_ELEMENT" {
			if len(item.Sources) != 2 || item.Sources[0].Subject == nil || item.Sources[0].Subject.Kind != "semantic-type" || item.Sources[1].Subject == nil || item.Sources[1].Subject.Kind != "semantic-data" {
				t.Fatalf("cross-kind candidates: %+v", item)
			}
			return
		}
	}
	t.Fatal("cross-kind diagnostic missing")
}

func TestValidateWorkspaceJSONDuplicateWithUnrelatedImportKeepsLocalCandidates(t *testing.T) {
	root := t.TempDir()
	consumer := filepath.Join(root, "consumer")
	writeWorkspaceFile(t, consumer, "tei.workspace.yaml", "workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: external\n    path: ../provider\n    namespace: external\n")
	writeWorkspaceFile(t, consumer, "semantic/request.tei", "namespace intake\ndata Request {\n attribute status type std.code\n attribute status type std.code\n}\n")
	provider := filepath.Join(root, "provider")
	writeWorkspaceFile(t, provider, "tei.workspace.yaml", "workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, provider, "semantic/external.tei", "namespace external\ntype ExternalId from std.identifier\n")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, consumer)
	if err != nil || code != 1 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	for _, item := range readValidationReport(t, &out).Diagnostics {
		if item.Diagnostic.Code == "DUPLICATE_SEMANTIC_ELEMENT" {
			if item.Diagnostic.Subject == nil || item.Diagnostic.Subject.ID != "intake.request.status" ||
				len(item.Sources) != 2 || len(item.UnresolvedSourceOrigins) != 0 ||
				item.Sources[0].ArtifactPath != "semantic/request.tei" || item.Sources[0].Range.Start.Line != 3 ||
				item.Sources[1].Range.Start.Line != 4 {
				t.Fatalf("unrelated import erased local candidates: %+v", item)
			}
			return
		}
	}
	t.Fatal("duplicate diagnostic missing")
}

func TestValidateWorkspaceJSONDuplicateWithPreludeMarksStandardUnknown(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: prelude\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/std.tei", "namespace std\ntype code from std.text\n")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 1 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	count := 0
	for _, item := range readValidationReport(t, &out).Diagnostics {
		if item.Diagnostic.Code != "DUPLICATE_SEMANTIC_ELEMENT" || item.Diagnostic.Subject == nil || item.Diagnostic.Subject.ID != "std.code" {
			continue
		}
		count++
		if len(item.Sources) != 1 || item.Sources[0].ArtifactPath != "semantic/std.tei" || item.Sources[0].Range.Start.Line != 2 {
			t.Fatalf("local prelude candidate: %+v", item.Sources)
		}
		if len(item.UnresolvedSourceOrigins) != 1 || item.UnresolvedSourceOrigins[0] != "standard" {
			t.Fatalf("standard origin was not marked unknown: %+v", item)
		}
	}
	if count == 0 {
		t.Fatal("prelude duplicate diagnostic missing")
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
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: valid\nformatVersion: 1\nsources:\n  - semantic\n")
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
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: bad\nformatVersion: 1\nsources:\n  - semantic\n")
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
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: warn\nformatVersion: 1\nsources:\n  - releases\n")
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

// TestValidateJSONReportsZeroDeclarationsForAnEmptyWorkspace checks that
// ok=true is not mistaken for "model declarations were loaded" (#2822).
func TestValidateJSONReportsZeroDeclarationsForAnEmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: empty_counts\nformatVersion: 1\nsources:\n  - semantic\n")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	var report cliValidationReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.DeclarationCounts == nil || report.DeclarationCounts.Total != 0 {
		t.Fatalf("empty workspace report=%+v", report)
	}
}

// TestValidateJSONReportsDeclarationsForAReferenceWorkspace checks that a
// non-empty Workspace is distinguishable from the empty one.
func TestValidateJSONReportsDeclarationsForAReferenceWorkspace(t *testing.T) {
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, filepath.Join("..", "..", "reference", "workspace"))
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	var report cliValidationReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.DeclarationCounts == nil || report.DeclarationCounts.Total == 0 ||
		report.DeclarationCounts.Data == 0 || report.DeclarationCounts.Applications == 0 {
		t.Fatalf("reference workspace counts=%+v", report.DeclarationCounts)
	}
}

// TestValidateJSONDiagnosticsKeepNestedDiagnostic pins the v0 shape that the
// contract document describes (#3347): diagnostics[] elements carry the
// diagnostic under a "diagnostic" key, not flattened. A reader that assumes a
// flat element would silently read no code; this test makes that visible before
// any flattening (which would need v1).
func TestValidateJSONDiagnosticsKeepNestedDiagnostic(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: shape\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/order.tei",
		"namespace sales\n\ndata Order {\n    attribute orderId type }\n}\n")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
	}
	var document struct {
		Diagnostics []map[string]json.RawMessage `json:"diagnostics"`
	}
	if err := json.Unmarshal(out.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Diagnostics) == 0 {
		t.Fatal("diagnostics is empty")
	}
	first := document.Diagnostics[0]
	if _, ok := first["code"]; ok {
		t.Fatalf("diagnostics[0] has a top-level code; the v0 element is nested: %v", first)
	}
	if _, ok := first["diagnostic"]; !ok {
		t.Fatalf("diagnostics[0] lost its diagnostic object: %v", first)
	}
}
