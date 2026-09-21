package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/semantic"
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

	app := referenceApplicationByID(t, result.World.Applications, "order_entry")
	if len(app.Patterns) != 5 {
		t.Fatalf("browse、record、reference、command、header-detailが共存していません: %#v", app.Patterns)
	}
	pattern := applicationPatternByID(t, app.Patterns, patternID)
	if pattern.Kind != application.PatternBrowse || pattern.Data != "sales.order" || pattern.Query != patternID {
		t.Fatalf("browse Patternの意味が期待値と異なります: %#v", pattern)
	}
	if pattern.Relation != "" || pattern.Capability != "" || pattern.Datasource != "" {
		t.Fatalf("browse Patternへ不要な参照が重複記述されています: %#v", pattern)
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
	if pattern.Relation != "" || pattern.Query != "" || pattern.Capability != "" || pattern.Datasource != "" {
		t.Fatalf("record Patternへ取得経路やRelation、Capabilityが重複して入り込んでいます: %#v", pattern)
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

func TestReferenceReferencePatternReusesRelationAndReadOnlyDatasourceContracts(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	const patternID = "order_entry.customer_reference"
	const patternPath = "applications/order-entry/patterns/customer-reference.tei"

	app := result.World.Applications[0]
	pattern := applicationPatternByID(t, app.Patterns, patternID)
	if pattern.Kind != application.PatternReference || pattern.Relation != "sales.order.customer" || pattern.Datasource != "master" {
		t.Fatalf("reference Patternの意味が期待値と異なります: %#v", pattern)
	}
	if pattern.Data != "" || pattern.Query != "" || pattern.Capability != "" {
		t.Fatalf("Relationが持つsource / targetの意味やCapabilityがPatternへ重複しています: %#v", pattern)
	}

	sourceData, relation := semanticRelationByID(t, result.World.Semantics, pattern.Relation)
	if sourceData != "sales.order" || relation.Target != "customer.customer" || relation.Cardinality != semantic.CardinalityOne {
		t.Fatalf("Relationが保持する参照元・参照先・cardinalityが期待値と異なります: source=%s relation=%#v", sourceData, relation)
	}

	master, ok := app.DatabaseByAlias(pattern.Datasource)
	if !ok {
		t.Fatalf("reference Patternが参照するDatasource %qがApplicationにありません", pattern.Datasource)
	}
	if master.Access != database.AccessRead {
		t.Fatalf("master Datasourceのread-only境界が失われました: access=%q", master.Access)
	}
	if master.TargetDatabase != "master_data" {
		t.Fatalf("master Datasourceのtarget=%q want=master_data", master.TargetDatabase)
	}
	if len(master.RequiredContracts) != 1 {
		t.Fatalf("master DatasourceのRequired Contract=%#v", master.RequiredContracts)
	}

	requirement := master.RequiredContracts[0]
	if requirement.ContractID != "master_data.customer" || requirement.CompatibleMajor != 2 {
		t.Fatalf("master DatasourceのContract Requirementが期待値と異なります: %#v", requirement)
	}
	contract := databaseContractByID(t, result.World.Databases, requirement.ContractID)
	if contract.Version.Major != requirement.CompatibleMajor {
		t.Fatalf("Contract major=%d requirement=%d", contract.Version.Major, requirement.CompatibleMajor)
	}
	if !containsCanonicalID(contract.Reads, relation.Target) {
		t.Fatalf("Relation target %sをContract %sからreadできません: reads=%#v", relation.Target, contract.ID, contract.Reads)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "application-pattern", ID: patternID})
	if len(refs) != 1 {
		t.Fatalf("reference Patternのsource references=%#v", refs)
	}
	if refs[0].ArtifactPath != patternPath {
		t.Fatalf("reference Patternのsource path=%q want=%q", refs[0].ArtifactPath, patternPath)
	}
}

// referenceApplicationByIDは、Reference WorkspaceのApplicationをIDで取り出す。
// Reference上のApplication数は第二Application受理後に増えるため、順序や件数に依存しない。
func referenceApplicationByID(t *testing.T, applications []application.Application, id string) application.Application {
	t.Helper()
	for _, app := range applications {
		if string(app.ID) == id {
			return app
		}
	}
	t.Fatalf("Application %q がありません: %#v", id, applications)
	return application.Application{}
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

func semanticRelationByID(t *testing.T, namespaces []semantic.Namespace, id core.CanonicalID) (core.CanonicalID, semantic.Relation) {
	t.Helper()
	for _, namespace := range namespaces {
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				if relation.ID == id {
					return data.ID, relation
				}
			}
		}
	}
	t.Fatalf("Semantic Relation %sが見つかりません", id)
	return "", semantic.Relation{}
}

func databaseContractByID(t *testing.T, databases []database.Database, id core.CanonicalID) database.Contract {
	t.Helper()
	for _, db := range databases {
		for _, contract := range db.Contracts {
			if contract.ID == id {
				return contract
			}
		}
	}
	t.Fatalf("Database Contract %sが見つかりません", id)
	return database.Contract{}
}

func containsCanonicalID(ids []core.CanonicalID, target core.CanonicalID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
