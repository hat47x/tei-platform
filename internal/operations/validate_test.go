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
	if report.OK || !report.InspectionIncomplete || len(report.Diagnostics) != 3 || report.WorldFingerprint != "" || report.SourceIndexFingerprint != "" || report.DeclarationCounts != nil || len(report.Proposals) != 0 {
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
