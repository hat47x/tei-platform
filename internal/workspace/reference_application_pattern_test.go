package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceBrowsePatternReusesExistingQueryDatasource(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	const patternID = "order_entry.recent_orders"
	const patternPath = "applications/order-entry/patterns/recent-orders.tei"

	if len(result.World.Applications) != 1 {
		t.Fatalf("applications=%#v", result.World.Applications)
	}
	app := result.World.Applications[0]
	if app.ID != "order_entry" {
		t.Fatalf("application id=%q", app.ID)
	}
	if len(app.Patterns) != 2 {
		t.Fatalf("browseとrecordが共存していません: %#v", app.Patterns)
	}
	pattern := applicationPatternByID(t, app.Patterns, patternID)
	if pattern.Kind != application.PatternBrowse || pattern.Data != "sales.order" || pattern.Query != patternID {
		t.Fatalf("browse Patternの意味が期待値と異なります: %#v", pattern)
	}
	if pattern.Datasource != "" {
		t.Fatalf("Pattern側にQueryと同じDatasourceが重複記述されています: %q", pattern.Datasource)
	}

	if len(result.World.Queries) != 1 {
		t.Fatalf("queries=%#v", result.World.Queries)
	}
	query := result.World.Queries[0]
	if query.ID != pattern.Query || query.Datasource != "sales" {
		t.Fatalf("Patternが参照するQueryのDatasourceが期待値と異なります: %#v", query)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "application-pattern", ID: patternID})
	if len(refs) != 1 {
		t.Fatalf("実Reference Patternのsource references=%#v", refs)
	}
	if refs[0].ArtifactPath != patternPath {
		t.Fatalf("実Reference Patternのsource path=%q want=%q", refs[0].ArtifactPath, patternPath)
	}
}

func TestReferenceRecordPatternUsesSemanticDataWithoutQueryOrDatasource(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	const patternID = "order_entry.order_record"
	const patternPath = "applications/order-entry/patterns/order-record.tei"

	app := result.World.Applications[0]
	pattern := applicationPatternByID(t, app.Patterns, patternID)
	if pattern.Kind != application.PatternRecord || pattern.Data != "sales.order" {
		t.Fatalf("record Patternの意味が期待値と異なります: %#v", pattern)
	}
	if pattern.Query != "" || pattern.Datasource != "" {
		t.Fatalf("record Patternへ取得経路が重複して入り込んでいます: %#v", pattern)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "application-pattern", ID: patternID})
	if len(refs) != 1 {
		t.Fatalf("record Patternのsource references=%#v", refs)
	}
	if refs[0].ArtifactPath != patternPath {
		t.Fatalf("record Patternのsource path=%q want=%q", refs[0].ArtifactPath, patternPath)
	}
}

func applicationPatternByID(t *testing.T, patterns []application.Pattern, id string) application.Pattern {
	t.Helper()
	for _, pattern := range patterns {
		if string(pattern.ID) == id {
			return pattern
		}
	}
	t.Fatalf("Application Pattern %sが見つかりません: %#v", id, patterns)
	return application.Pattern{}
}
