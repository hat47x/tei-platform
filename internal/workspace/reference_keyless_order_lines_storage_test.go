package workspace

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/database/providers/postgres"
	"tei/internal/validation"
)

func TestReferenceKeylessOrderLinesStorageReachesPhysicalQueryWithoutSyntheticSemanticAttribute(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}

	var orderLine persistence.Mapping
	found := false
	for _, mapping := range loaded.World.Persistence {
		if mapping.ID == "sales_core.order_line" {
			orderLine = mapping
			found = true
			break
		}
	}
	if !found {
		t.Fatal("sales_core.order_line Persistence Mappingがありません")
	}
	if len(orderLine.Columns) != 5 {
		t.Fatalf("OrderLineのSemantic attribute列数が想定外です: %#v", orderLine.Columns)
	}
	for _, column := range orderLine.Columns {
		if column.Column == "order_id" {
			t.Fatalf("Relation-owned order_idをOrderLineのSemantic attribute列へ偽装しています: %#v", column)
		}
	}
	if len(orderLine.RelationStorage) != 1 {
		t.Fatalf("sales.order.linesのRelation Storageがありません: %#v", orderLine.RelationStorage)
	}
	storage := orderLine.RelationStorage[0]
	if storage.Relation != "sales.order.lines" || len(storage.Components) != 1 || storage.Components[0].SourceAttribute != "sales.order.order_id" || storage.Components[0].Column != "order_id" {
		t.Fatalf("Relation Storageの意味または物理保存契約が変わっています: %#v", storage)
	}

	if diagnostics := validation.ValidatePersistenceRelationStorageMappings(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("Reference Relation StorageのWorld-level Validationが失敗しました: %#v", diagnostics)
	}
	if diagnostics := validation.ValidateContractQuerySQLRealizations(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("find_by_order SQL realizationのValidationが失敗しました: %#v", diagnostics)
	}

	snapshot, err := persistence.RealizeDesired(loaded.World.Persistence, "sales_core", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatalf("sales_core Desired Physical Schemaを生成できません: %v", err)
	}
	physicalFound := false
	for _, table := range snapshot.Tables {
		if table.Schema != "public" || table.Name != "order_lines" {
			continue
		}
		physicalFound = true
		if len(table.Columns) != 6 || table.Columns[5].Name != "order_id" {
			t.Fatalf("Relation-owned support columnがDesired Schemaへ正しく投影されていません: %#v", table.Columns)
		}
		if len(table.ForeignKeys) != 0 {
			t.Fatalf("Semantic Relationからphysical FKを暗黙生成しています: %#v", table.ForeignKeys)
		}
	}
	if !physicalFound {
		t.Fatal("public.order_linesがDesired Physical Schemaにありません")
	}

	realizationFound := false
	for _, realization := range loaded.World.ContractQuerySQLRealizations {
		if realization.Target.Database != "sales_core" || realization.Target.Contract != "sales_core.order_line" || realization.Target.Query != "find_by_order" {
			continue
		}
		realizationFound = true
		if realization.Provider != "postgres" {
			t.Fatalf("find_by_orderのproviderが変わっています: %s", realization.Provider)
		}
		if len(realization.Inputs) != 1 || realization.Inputs[0].Input != "order_id" || realization.Inputs[0].Parameter != "order_id" {
			t.Fatalf("find_by_orderのinput -> physical parameter mappingが不正です: %#v", realization.Inputs)
		}
		if !strings.Contains(realization.Source, "FROM public.order_lines") || !strings.Contains(realization.Source, "WHERE order_id = :order_id") {
			t.Fatalf("find_by_orderがRelation-owned physical columnを明示的に参照していません: %s", realization.Source)
		}
		if !strings.Contains(realization.Source, "ORDER BY line_number, order_line_id") {
			t.Fatalf("find_by_orderの並び順が同順位で不安定です: %s", realization.Source)
		}
		if strings.Contains(realization.Source, "sales.order_line.order_id") {
			t.Fatalf("native SQLへ存在しないSemantic attribute identityが漏れています: %s", realization.Source)
		}
	}
	if !realizationFound {
		t.Fatal("sales_core.order_line/find_by_order SQL realizationがありません")
	}

	storageSource := loaded.Sources["persistence-relation-storage:sales_core.order_line/sales.order.lines"]
	if storageSource.File != "databases/sales-core/relation-storage/order-line.tei" {
		t.Fatalf("Relation Storageの正本authoring sourceを追跡できません: %#v", storageSource)
	}
	querySource := loaded.Sources["contract-query-sql-realization:sales_core/sales_core.order_line/find_by_order/sql/postgres"]
	if querySource.File != "databases/sales-core/query-realizations/order-line-find-by-order.tei" {
		t.Fatalf("Contract Query SQL realizationの正本authoring sourceを追跡できません: %#v", querySource)
	}
}
