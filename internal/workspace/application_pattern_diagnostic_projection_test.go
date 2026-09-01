package workspace_test

import (
	"testing"

	"tei/internal/subject"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestApplicationPatternValidationDiagnosticProjectsToCanonicalIdentifierToken(t *testing.T) {
	root := t.TempDir()
	writePatternWorkspaceFile(t, root, "tei.workspace.yaml", `workspace: pattern_diagnostic_projection_test
sources:
  - applications
`)
	writePatternWorkspaceFile(t, root, "applications/sample/application.tei", `application sample_app {
}`)

	patternPath := "applications/sample/patterns/recent-orders.tei"
	patternSource := `pattern sample_app.recent_orders browse {
    data sales.missing
    query sample_app.missing_query
}`
	writePatternWorkspaceFile(t, root, patternPath, patternSource)

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Workspaceの読み込みに失敗しました: %v", err)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}

	diagnostics := validation.ValidateApplicationPatterns(loaded.World)
	diagnostic, ok := diagnosticByCode(diagnostics, "UNKNOWN_APPLICATION_PATTERN_DATA")
	if !ok {
		t.Fatalf("Pattern Validationに期待したDiagnosticがありません: %#v", diagnostics)
	}
	wantSubject := subject.Subject{Kind: "application-pattern", ID: "sample_app.recent_orders"}
	if diagnostic.Subject == nil || *diagnostic.Subject != wantSubject {
		t.Fatalf("DiagnosticのSubject=%#v want=%+v", diagnostic.Subject, wantSubject)
	}

	projected := workspace.ProjectDiagnostics([]validation.Diagnostic{diagnostic}, idx)
	if len(projected) != 1 {
		t.Fatalf("Diagnostic projectionの件数=%d want=1: %#v", len(projected), projected)
	}
	if len(projected[0].Sources) != 1 {
		t.Fatalf("投影されたsourceの件数=%d want=1: %#v", len(projected[0].Sources), projected[0].Sources)
	}

	source := projected[0].Sources[0]
	if source.ArtifactPath != patternPath {
		t.Fatalf("投影先=%q want=%q", source.ArtifactPath, patternPath)
	}

	content := []byte(patternSource)
	rng := source.Range
	if got := string(content[rng.Start.Offset:rng.End.Offset]); got != "sample_app.recent_orders" {
		t.Fatalf("Diagnosticの投影範囲=%q want=%q range=%+v", got, "sample_app.recent_orders", rng)
	}
}
