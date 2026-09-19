package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceHeaderDetailPatternReusesManyRelationWithoutPersistenceAssumption(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	const patternID = "order_entry.order_lines"
	const patternPath = "applications/order-entry/patterns/order-lines.tei"

	if len(result.World.Applications) != 1 {
		t.Fatalf("applications=%#v", result.World.Applications)
	}
	app := result.World.Applications[0]
	pattern := applicationPatternByID(t, app.Patterns, patternID)
	if pattern.Kind != application.PatternHeaderDetail || pattern.Relation != "sales.order.lines" {
		t.Fatalf("header-detail Patternの意味が期待値と異なります: %#v", pattern)
	}
	if pattern.Data != "" || pattern.Query != "" || pattern.Capability != "" || pattern.Datasource != "" {
		t.Fatalf("Relationが持つHeader / Detailの意味やアクセス経路がPatternへ重複しています: %#v", pattern)
	}

	sourceData, relation := semanticRelationByID(t, result.World.Semantics, pattern.Relation)
	if sourceData != "sales.order" || relation.Target != "sales.order_line" || relation.Cardinality != semantic.CardinalityMany {
		t.Fatalf("header-detail Relationが期待値と異なります: source=%s relation=%#v", sourceData, relation)
	}

	for _, mapping := range result.World.Persistence {
		if mapping.SemanticData == "sales.order_line" {
			requireExplicitDatabaseAuthoring(t, result, "persistence:"+string(mapping.ID), "header-detail Patternとは別のOrderLine Persistence Mapping")
		}
	}
	for _, database := range result.World.Databases {
		for _, contract := range database.Contracts {
			if containsCanonicalID(contract.Reads, "sales.order_line") || containsCanonicalID(contract.Writes, "sales.order_line") {
				requireExplicitDatabaseAuthoring(t, result, "database-contract:"+string(contract.ID), "OrderLineを公開するDatabase Contract")
			}
		}
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "application-pattern", ID: patternID})
	if len(refs) != 1 {
		t.Fatalf("header-detail Patternのsource references=%#v", refs)
	}
	if refs[0].ArtifactPath != patternPath {
		t.Fatalf("header-detail Patternのsource path=%q want=%q", refs[0].ArtifactPath, patternPath)
	}
}
