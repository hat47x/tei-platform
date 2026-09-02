package applicationdriver_test

import (
	"path/filepath"
	"testing"

	"tei/internal/applicationdriver"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/realization"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestReferenceOrderEntryScenarioResolvesCanonicalResourcesWithoutDuplication(t *testing.T) {
	root := applicationDriverRepoRoot(t)
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	patternIDs := []core.CanonicalID{
		"order_entry.recent_orders",
		"order_entry.order_record",
		"order_entry.customer_reference",
		"order_entry.order_lines",
		"order_entry.submit_order",
	}
	plan, err := applicationdriver.AssemblePatternScenario(loaded.World, "order_entry", patternIDs)
	if err != nil {
		t.Fatalf("order_entry Scenarioのassemblyに失敗しました: %v", err)
	}
	resolved, err := applicationdriver.ResolveScenarioResources(loaded.World, plan)
	if err != nil {
		t.Fatalf("order_entry Scenarioの参照先解決に失敗しました: %v", err)
	}
	if len(resolved) != len(patternIDs) {
		t.Fatalf("resolved=%#v", resolved)
	}

	browse := resolved[0]
	if browse.Query == nil || browse.Query.ID != "order_entry.recent_orders" {
		t.Fatalf("browse Queryを解決できていません: %#v", browse)
	}
	if browse.Query.Datasource != "sales" {
		t.Fatalf("browse Query datasource=%s", browse.Query.Datasource)
	}
	if browse.Datasource == nil || browse.Datasource.Alias != "sales" || browse.Datasource.TargetDatabase != "sales_core" || browse.Datasource.Access != database.AccessReadWrite {
		t.Fatalf("browse Datasourceが期待値と異なります: %#v", browse.Datasource)
	}

	record := resolved[1]
	if record.Query != nil || record.Datasource != nil || record.Relation != nil || record.Capability != nil || record.Binding != nil {
		t.Fatalf("record Patternへ未定義の依存が追加されています: %#v", record)
	}

	reference := resolved[2]
	if reference.RelationSource == nil || reference.RelationSource.ID != "sales.order" {
		t.Fatalf("reference Relation sourceを解決できていません: %#v", reference.RelationSource)
	}
	if reference.Relation == nil || reference.Relation.ID != "sales.order.customer" || reference.Relation.Target != "customer.customer" || reference.Relation.Cardinality != semantic.CardinalityOne {
		t.Fatalf("reference Relationが期待値と異なります: %#v", reference.Relation)
	}
	if reference.Datasource == nil || reference.Datasource.Alias != "master" || reference.Datasource.TargetDatabase != "master_data" || reference.Datasource.Access != database.AccessRead {
		t.Fatalf("reference Datasourceが期待値と異なります: %#v", reference.Datasource)
	}

	headerDetail := resolved[3]
	if headerDetail.RelationSource == nil || headerDetail.RelationSource.ID != "sales.order" {
		t.Fatalf("header-detail Relation sourceを解決できていません: %#v", headerDetail.RelationSource)
	}
	if headerDetail.Relation == nil || headerDetail.Relation.ID != "sales.order.lines" || headerDetail.Relation.Target != "sales.order_line" || headerDetail.Relation.Cardinality != semantic.CardinalityMany {
		t.Fatalf("header-detail Relationが期待値と異なります: %#v", headerDetail.Relation)
	}
	if headerDetail.Datasource != nil {
		t.Fatalf("header-detailへDatasourceを推測してはいけません: %#v", headerDetail.Datasource)
	}

	command := resolved[4]
	if command.Capability == nil || command.Capability.ID != "sales.order.submit" {
		t.Fatalf("command Capabilityを解決できていません: %#v", command.Capability)
	}
	if command.Binding == nil || command.Binding.Identity() != realization.BindingIdentity("order_entry", "sales.order.submit") {
		t.Fatalf("command Bindingが期待値と異なります: %#v", command.Binding)
	}
	if command.Query != nil || command.Datasource != nil || command.Relation != nil {
		t.Fatalf("command Patternへ未定義の参照先が追加されています: %#v", command)
	}
}
