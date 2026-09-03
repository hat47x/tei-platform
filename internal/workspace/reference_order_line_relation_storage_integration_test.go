package workspace_test

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestReferenceOrderLineRelationStorageAndQueryRealizationStayExplicit(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	result, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}

	var orderLineFound bool
	for _, mapping := range result.World.Persistence {
		if mapping.ID != "sales_core.order_line" {
			continue
		}
		orderLineFound = true
		if mapping.SemanticData != "sales.order_line" || mapping.Table != "public.order_lines" {
			t.Fatalf("OrderLine Persistence Mappingの正本が変わっています: %#v", mapping)
		}
		for _, column := range mapping.Columns {
			if column.Column == "order_id" {
				t.Fatalf("Relation support column order_idを通常のSemantic attribute列へ混入させています: %#v", column)
			}
		}
		storage, ok := mapping.RelationStorageByRelation("sales.order.lines")
		if !ok {
			t.Fatalf("sales.order.linesのRelation Storageがありません: %#v", mapping.RelationStorage)
		}
		if len(storage.Components) != 1 {
			t.Fatalf("Relation Storage component数が期待値と異なります: %#v", storage.Components)
		}
		component := storage.Components[0]
		if component.SourceAttribute != "sales.order.order_id" || component.Column != "order_id" || component.Storage != "int64" || component.Nullability != "required" {
			t.Fatalf("Order -> OrderLineの物理保存境界が変わっています: %#v", component)
		}
	}
	if !orderLineFound {
		t.Fatal("sales_core.order_line Persistence Mappingがありません")
	}

	if diagnostics := validation.ValidatePersistenceRelationStorageMappings(result.World); len(diagnostics) != 0 {
		t.Fatalf("Reference Relation StorageがSemantic Relationと整合しません: %#v", diagnostics)
	}

	var realizationFound bool
	for _, realizationDef := range result.World.ContractQuerySQLRealizations {
		if realizationDef.Target.Database != "sales_core" || realizationDef.Target.Contract != "sales_core.order_line" || realizationDef.Target.Query != "find_by_order" {
			continue
		}
		realizationFound = true
		if realizationDef.Provider != "postgres" {
			t.Fatalf("find_by_orderのproviderが変わっています: %s", realizationDef.Provider)
		}
		if len(realizationDef.Inputs) != 1 || realizationDef.Inputs[0].Input != "order_id" || realizationDef.Inputs[0].Parameter != "order_id" {
			t.Fatalf("find_by_orderのinput -> physical parameter mappingが不正です: %#v", realizationDef.Inputs)
		}
		if !strings.Contains(realizationDef.Source, "FROM public.order_lines") || !strings.Contains(realizationDef.Source, "WHERE order_id = :order_id") {
			t.Fatalf("find_by_order native SQLがRelation Storageの物理契約と一致しません: %q", realizationDef.Source)
		}
	}
	if !realizationFound {
		t.Fatal("sales_core.order_line/find_by_orderのSQL realizationがありません")
	}

	if diagnostics := validation.ValidateContractQuerySQLRealizations(result.World); len(diagnostics) != 0 {
		t.Fatalf("Reference find_by_order SQL realizationがContract Queryと整合しません: %#v", diagnostics)
	}

	storageSource := result.Sources["persistence-relation-storage:sales_core.order_line/sales.order.lines"]
	if storageSource.File != "databases/sales-core/relation-storage/order-lines.tei" {
		t.Fatalf("Relation Storage authoring sourceを追跡できません: %#v", storageSource)
	}
	querySource := result.Sources["contract-query-sql-realization:sales_core/sales_core.order_line/find_by_order/sql/postgres"]
	if querySource.File != "databases/sales-core/query-realizations/order-lines.tei" {
		t.Fatalf("Contract Query SQL realization authoring sourceを追跡できません: %#v", querySource)
	}
}
