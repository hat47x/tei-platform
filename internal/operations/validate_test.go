package operations

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func writeWorkspaceFixture(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateWorkspaceCollectsSyntaxWithoutPartialWorld(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFixture(t, root, "tei.workspace.yaml", "workspace: multi\nformatVersion: 1\nsources:\n  - semantic\n  - plugins\n")
	writeWorkspaceFixture(t, root, "semantic/a.tei", "namespace a\ndata Broken { attribute x type }\n")
	writeWorkspaceFixture(t, root, "semantic/b.tei", "namespace b\ntype Bad from\n")
	writeWorkspaceFixture(t, root, "plugins/broken/plugin.tei", "plugin broken { invalid }\n")
	got, err := ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Direct == nil || len(got.AdditionalDirect) != 2 || !got.InspectionIncomplete || !got.HasErrors() {
		t.Fatalf("expected three source findings: %+v", got)
	}
	report := WorkspaceValidationReportFrom(got)
	if report.OK || len(report.Diagnostics) != 3 || report.WorldFingerprint != "" || report.SourceIndexFingerprint != "" || report.DeclarationCounts != nil || len(report.Proposals) != 0 {
		t.Fatalf("unexpected partial-world report: %+v", report)
	}
	if report.Incomplete == nil || report.Incomplete.Reason != "syntax-coverage-unverified" || report.Incomplete.StoppedAt != "" {
		t.Fatalf("multiple findings must not claim fail-fast stoppedAt: %+v", report.Incomplete)
	}
	_, err = workspace.Load(root)
	var single *workspace.SourceError
	var multiple *workspace.SourceErrors
	if !errors.As(err, &single) || errors.As(err, &multiple) {
		t.Fatalf("normal Load must fail fast: %v", err)
	}
}

func TestValidateWorkspaceCollectsImportedSyntaxAndKeepsIOFailure(t *testing.T) {
	base := t.TempDir()
	consumer, provider := filepath.Join(base, "consumer"), filepath.Join(base, "provider")
	manifest := "workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: p\n    path: ../provider\n    namespace: p\n"
	writeWorkspaceFixture(t, consumer, "tei.workspace.yaml", manifest)
	writeWorkspaceFixture(t, consumer, "semantic/a.tei", "namespace a\ntype Broken from\n")
	writeWorkspaceFixture(t, provider, "tei.workspace.yaml", "workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFixture(t, provider, "semantic/b.tei", "namespace p\ntype Broken from\n")
	got, err := ValidateWorkspace(consumer)
	if err != nil || got.Direct == nil || len(got.AdditionalDirect) != 1 || !got.InspectionIncomplete {
		t.Fatalf("import findings: %+v, %v", got, err)
	}
	paths := map[string]bool{got.Direct.Source.ArtifactPath: true, got.AdditionalDirect[0].Source.ArtifactPath: true}
	if !paths["semantic/a.tei"] || !paths["../provider/semantic/b.tei"] {
		t.Fatalf("import source paths lost: %v", paths)
	}
	writeWorkspaceFixture(t, consumer, "tei.workspace.yaml", "workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: p\n    path: ../missing\n    namespace: p\n")
	_, err = ValidateWorkspace(consumer)
	var failure *ValidationFailure
	if !errors.As(err, &failure) || failure.Stage != StageLoad {
		t.Fatalf("missing import after syntax failure must remain load failure: %v", err)
	}
}

func TestValidateWorkspaceAsTypedReadOperation(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFixture(t, root, "tei.workspace.yaml", "workspace: typed_validation\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFixture(t, root, "semantic/sales/order.tei", "namespace sales\n")

	valid, err := ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if valid.Direct != nil || valid.HasErrors() {
		t.Fatalf("unexpected validation failure: %+v", valid)
	}

	writeWorkspaceFixture(t, root, "semantic/sales/order.tei", "namespace sales\n\ndata Order {\n    attribute orderId type }\n}\n")
	invalid, err := ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Direct == nil || invalid.Direct.Code != "PARSE_ERROR" || !invalid.HasErrors() {
		t.Fatalf("parser finding should remain direct: %+v", invalid)
	}
	if invalid.Direct.Source.ArtifactPath != "semantic/sales/order.tei" ||
		invalid.Direct.Source.Range.Start.Line != 4 {
		t.Fatalf("source coordinate lost: %+v", invalid.Direct)
	}
}

func TestValidateWorkspaceFailureIsNotBusinessFinding(t *testing.T) {
	_, err := ValidateWorkspace(filepath.Join(t.TempDir(), "missing"))
	var failure *ValidationFailure
	if !errors.As(err, &failure) || failure.Stage != StageLoad {
		t.Fatalf("expected typed load failure, got %v", err)
	}
}

// A declared but missing source root has no Canonical Subject, so it is not a
// semantic diagnostic. The typed operation, Catalog and shared CLI/MCP report
// must nevertheless agree that the same observed Workspace is invalid.
func TestValidateMissingDeclaredSourceRootAgreesAcrossReaders(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFixture(t, root, "tei.workspace.yaml",
		"workspace: missing_root\nformatVersion: 1\nsources:\n  - semantic\n  - nonexistent\n")
	writeWorkspaceFixture(t, root, "semantic/order.tei", "namespace sales\n")

	validated, err := ValidateWorkspace(root)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validated.Direct != nil || len(validated.Diagnostics) != 0 {
		t.Fatalf("expected source-root-only error, not canonical diagnostic: %+v", validated)
	}
	found := false
	for _, finding := range validated.SourceRootFindings {
		if finding.Root == "nonexistent" && finding.Severity == "error" {
			found = true
		}
	}
	if !found || !validated.HasErrors() {
		t.Fatalf("missing root must invalidate typed result: %+v", validated.SourceRootFindings)
	}

	catalog, err := CatalogWorkspace(root, "")
	if err != nil {
		t.Fatalf("catalog of invalid Workspace should remain navigable: %v", err)
	}
	report := WorkspaceValidationReportFrom(validated)
	if catalog.Validation.OK || catalog.Validation.ErrorCount != 1 || report.OK {
		t.Fatalf("verdict mismatch: catalog=%+v report=%+v", catalog.Validation, report)
	}
	mapped, err := MapWorkspace(root)
	if err != nil {
		t.Fatalf("map of invalid Workspace must remain read-only: %v", err)
	}
	if mapped.Catalog.Validation != catalog.Validation ||
		mapped.Catalog.WorldFingerprint != catalog.WorldFingerprint ||
		mapped.Catalog.SourceIndexFingerprint != catalog.SourceIndexFingerprint ||
		mapped.Coverage != "partial-projection" {
		t.Fatalf("map cannot invent validation or coverage: %+v", mapped)
	}
	if catalog.WorldFingerprint == "" ||
		catalog.WorldFingerprint != report.WorldFingerprint ||
		catalog.SourceIndexFingerprint != report.SourceIndexFingerprint ||
		catalog.Workspace == nil || report.Workspace == nil {
		t.Fatal("read-only catalog and report lack matching observed source context")
	}
	// Separate calls are not one atomic reader attempt or an acceptance grant.
}

// An existing but unknown root is reported as uninterpreted, not as an
// automatically invalid Workspace; warnings must not be promoted to errors.
func TestValidateUnknownDeclaredSourceRootWarnsWithoutFailing(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFixture(t, root, "tei.workspace.yaml",
		"workspace: unknown_root\nformatVersion: 1\nsources:\n  - semantic\n  - future\n")
	writeWorkspaceFixture(t, root, "semantic/order.tei", "namespace sales\n")
	writeWorkspaceFixture(t, root, "future/note.txt", "not interpreted by current loader\n")

	validated, err := ValidateWorkspace(root)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	found := false
	for _, finding := range validated.SourceRootFindings {
		if finding.Root == "future" && finding.Severity == "warning" {
			found = true
		}
	}
	if !found || validated.HasErrors() {
		t.Fatalf("unknown but existing source root must be a warning: %+v", validated.SourceRootFindings)
	}
	catalog, err := CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	report := WorkspaceValidationReportFrom(validated)
	if !catalog.Validation.OK || !report.OK || catalog.Validation.WarningCount == 0 {
		t.Fatalf("warning-only verdict mismatch: catalog=%+v report=%+v", catalog.Validation, report)
	}
	mapped, err := MapWorkspace(root)
	if err != nil {
		t.Fatalf("map of warning-only Workspace: %v", err)
	}
	if mapped.Catalog.Validation != catalog.Validation ||
		mapped.Coverage != "partial-projection" {
		t.Fatalf("map warning-only verdict mismatch: map=%+v catalog=%+v", mapped.Catalog.Validation, catalog.Validation)
	}
}

// MapWorkspace builds a CatalogWorld directly; its validation verdict must
// be explicitly supplied from the same Load rather than silently retaining
// CatalogWorld's zero-valued Validation.OK=false for a valid Workspace.
func TestMapWorkspaceCatalogValidationMatchesActualTypeChecking(t *testing.T) {
	for _, tc := range []struct {
		name     string
		extra    string
		wantOK   bool
		wantErrs int
	}{
		{"sound", "", true, 0},
		{"unknown type", "data X {\n    attribute a type sales.nothere\n}\n", false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeWorkspaceFixture(t, root, "tei.workspace.yaml",
				"workspace: map_validation\nformatVersion: 1\nsources:\n  - semantic\n")
			writeWorkspaceFixture(t, root, "semantic/model.tei",
				"namespace sales\n\n"+tc.extra)
			mapped, err := MapWorkspace(root)
			if err != nil {
				t.Fatalf("map: %v", err)
			}
			catalog, err := CatalogWorkspace(root, "")
			if err != nil {
				t.Fatalf("catalog: %v", err)
			}
			validated, err := ValidateWorkspace(root)
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
			report := WorkspaceValidationReportFrom(validated)
			if mapped.Catalog.Validation != catalog.Validation ||
				mapped.Catalog.Validation.OK != tc.wantOK ||
				mapped.Catalog.Validation.ErrorCount != tc.wantErrs ||
				report.OK != tc.wantOK ||
				mapped.Coverage != "partial-projection" {
				t.Fatalf("map/canonical validation disagree: map=%+v catalog=%+v report=%+v",
					mapped.Catalog.Validation, catalog.Validation, report)
			}
			// Repeated read of the same fixture does not prove one atomic
			// cross-Operation attempt or allow editing a Map link.
		})
	}
}
