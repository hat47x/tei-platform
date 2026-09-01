package workspace_test

import (
	"path/filepath"
	"testing"

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
	if len(app.Patterns) != 1 {
		t.Fatalf("patterns=%#v", app.Patterns)
	}
	pattern := app.Patterns[0]
	if pattern.ID != patternID || pattern.Kind != "browse" || pattern.Data != "sales.order" || pattern.Query != patternID {
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
