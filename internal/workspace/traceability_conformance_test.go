package workspace_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/query/analyzer"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestTraceabilityConformanceNativeSQLFlowsWithoutAuthoringPath(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}

	const queryID = "order_entry.recent_orders"
	const sqlPath = "applications/order-entry/queries/recent_orders.sql"
	var found bool
	var request analyzer.Request
	for _, query := range result.World.Queries {
		if query.ID != queryID {
			continue
		}
		found = true
		request = analyzer.Request{
			ProtocolVersion: 1,
			Operation:       "analyze",
			Query:           query.ID,
			Dialect:         query.Dialect,
			Source:          query.Source,
		}
		break
	}
	if !found {
		t.Fatalf("query %s not found", queryID)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "query", ID: queryID})
	authoring := referenceByRole(t, refs, sourceindex.RoleAuthoring)
	if authoring.ArtifactPath != sqlPath {
		t.Fatalf("authoring path=%q", authoring.ArtifactPath)
	}

	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), sqlPath) || strings.Contains(string(payload), "artifactPath") {
		t.Fatalf("authoring path leaked into analyzer request: %s", payload)
	}
	var shape map[string]any
	if err := json.Unmarshal(payload, &shape); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"path", "file", "artifact", "artifactPath"} {
		if _, exists := shape[forbidden]; exists {
			t.Fatalf("analyzer request contains source-side field %q: %#v", forbidden, shape)
		}
	}
	if shape["query"] != queryID || shape["source"] == "" {
		t.Fatalf("analyzer request lost canonical query meaning: %#v", shape)
	}
}

func TestTraceabilityConformanceMoveKeepsWorldAndRetargetsDiagnostic(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - applications\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTraceabilityApplication(t, root, "order-entry")

	before, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	beforeDiagnostic := diagnosticByCode(t, validation.Validate(before.World), "INVALID_DATABASE_ACCESS")
	if beforeDiagnostic.Subject == nil || beforeDiagnostic.Subject.Kind != "application" || beforeDiagnostic.Subject.ID != "order_entry" {
		t.Fatalf("diagnostic subject=%#v", beforeDiagnostic.Subject)
	}
	beforeIndex, err := workspace.BuildSourceIndex(root, before)
	if err != nil {
		t.Fatal(err)
	}
	beforeProjected := workspace.ProjectDiagnostics([]validation.Diagnostic{beforeDiagnostic}, beforeIndex)[0]
	if len(beforeProjected.Sources) != 1 {
		t.Fatalf("before sources=%#v", beforeProjected.Sources)
	}
	beforeSource := beforeProjected.Sources[0]
	if beforeSource.ArtifactPath != "applications/order-entry/application.tei" {
		t.Fatalf("before source=%#v", beforeSource)
	}

	if err := os.Rename(
		filepath.Join(root, "applications", "order-entry"),
		filepath.Join(root, "applications", "checkout"),
	); err != nil {
		t.Fatal(err)
	}

	after, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.World, after.World) {
		t.Fatalf("directory move changed normalized World\nbefore=%#v\nafter=%#v", before.World, after.World)
	}
	afterDiagnostic := diagnosticByCode(t, validation.Validate(after.World), "INVALID_DATABASE_ACCESS")
	if !reflect.DeepEqual(beforeDiagnostic, afterDiagnostic) {
		t.Fatalf("directory move changed semantic diagnostic\nbefore=%#v\nafter=%#v", beforeDiagnostic, afterDiagnostic)
	}
	afterIndex, err := workspace.BuildSourceIndex(root, after)
	if err != nil {
		t.Fatal(err)
	}
	afterProjected := workspace.ProjectDiagnostics([]validation.Diagnostic{afterDiagnostic}, afterIndex)[0]
	if len(afterProjected.Sources) != 1 {
		t.Fatalf("after sources=%#v", afterProjected.Sources)
	}
	afterSource := afterProjected.Sources[0]
	if afterSource.ArtifactPath != "applications/checkout/application.tei" {
		t.Fatalf("after source=%#v", afterSource)
	}
	if beforeSource.Range != afterSource.Range {
		t.Fatalf("unchanged source content changed range: before=%+v after=%+v", beforeSource.Range, afterSource.Range)
	}

	beforeArtifact := conformanceArtifactByPath(t, beforeIndex, beforeSource.ArtifactPath)
	afterArtifact := conformanceArtifactByPath(t, afterIndex, afterSource.ArtifactPath)
	if beforeArtifact.Fingerprint != afterArtifact.Fingerprint {
		t.Fatalf("directory move changed content fingerprint: %s != %s", beforeArtifact.Fingerprint, afterArtifact.Fingerprint)
	}
}

func TestTraceabilityConformanceParseErrorIsSourceDirect(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - applications\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "applications", "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "application.tei"), []byte("application order_entry { unexpected }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("broken source unexpectedly loaded")
	}
	diagnostic, ok := workspace.SourceDiagnosticFromError(root, err)
	if !ok {
		t.Fatalf("load error is not source-direct: %v", err)
	}
	if diagnostic.Code != "PARSE_ERROR" || diagnostic.Source.ArtifactPath != "applications/broken/application.tei" {
		t.Fatalf("source diagnostic=%#v", diagnostic)
	}
	if diagnostic.Source.Range.Start.Line != 1 || diagnostic.Source.Range.Start.Column <= 1 {
		t.Fatalf("source diagnostic range=%+v", diagnostic.Source.Range)
	}
}

func writeTraceabilityApplication(t *testing.T, root, unit string) {
	t.Helper()
	dir := filepath.Join(root, "applications", unit)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const source = `application order_entry {
    database sales {
        target sales_core
        access invalid
    }
}
`
	if err := os.WriteFile(filepath.Join(dir, "application.tei"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func diagnosticByCode(t *testing.T, diagnostics []validation.Diagnostic, code string) validation.Diagnostic {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return diagnostic
		}
	}
	t.Fatalf("diagnostic %s not found: %#v", code, diagnostics)
	return validation.Diagnostic{}
}

func conformanceArtifactByPath(t *testing.T, index sourceindex.Index, path string) sourceindex.Artifact {
	t.Helper()
	for _, artifact := range index.Artifacts {
		if artifact.Path == path {
			return artifact
		}
	}
	t.Fatalf("artifact %s not found", path)
	return sourceindex.Artifact{}
}
