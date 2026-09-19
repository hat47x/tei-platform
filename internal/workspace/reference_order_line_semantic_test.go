package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestReferenceOrderLineKeepsSemanticRelationSeparateFromPersistence(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	orderLine := semanticDataByID(t, result.World.Semantics, "sales.order_line")
	wantAttributes := map[core.CanonicalID]core.CanonicalID{
		"sales.order_line.order_line_id": "sales.order_line_id",
		"sales.order_line.line_number":   "std.integer",
		"sales.order_line.item_code":     "std.code",
		"sales.order_line.quantity":      "std.decimal",
		"sales.order_line.unit_price":    "std.money",
	}
	if len(orderLine.Attributes) != len(wantAttributes) {
		t.Fatalf("OrderLine attributes=%#v", orderLine.Attributes)
	}
	for _, attribute := range orderLine.Attributes {
		wantType, ok := wantAttributes[attribute.ID]
		if !ok {
			t.Fatalf("OrderLineに未想定の属性があります: %#v", attribute)
		}
		if attribute.TypeRef != wantType {
			t.Fatalf("OrderLine attribute %s type=%s want=%s", attribute.ID, attribute.TypeRef, wantType)
		}
	}

	source, lines := semanticRelationByID(t, result.World.Semantics, "sales.order.lines")
	if source != "sales.order" || lines.Target != "sales.order_line" || lines.Cardinality != semantic.CardinalityMany {
		t.Fatalf("Order→OrderLine Relationが期待値と異なります: source=%s relation=%#v", source, lines)
	}

	customerSource, customer := semanticRelationByID(t, result.World.Semantics, "sales.order.customer")
	if customerSource != "sales.order" || customer.Target != "customer.customer" || customer.Cardinality != semantic.CardinalityOne {
		t.Fatalf("既存customer Relationが変わっています: source=%s relation=%#v", customerSource, customer)
	}

	submit := semanticCapabilityByID(t, result.World.Semantics, "sales.order.submit")
	if len(submit.Inputs) != 1 || submit.Inputs[0].Name != "order" || submit.Inputs[0].Type != "sales.order" || submit.Output != "sales.order" {
		t.Fatalf("既存submit Capabilityの契約が変わっています: %#v", submit)
	}

	for _, mapping := range result.World.Persistence {
		if mapping.SemanticData == "sales.order_line" {
			requireExplicitDatabaseAuthoring(t, result, "persistence:"+string(mapping.ID), "Semanticとは別のOrderLine Persistence Mapping")
		}
	}
	for _, db := range result.World.Databases {
		for _, contract := range db.Contracts {
			if containsCanonicalID(contract.Reads, "sales.order_line") || containsCanonicalID(contract.Writes, "sales.order_line") {
				requireExplicitDatabaseAuthoring(t, result, "database-contract:"+string(contract.ID), "OrderLineを公開するDatabase Contract")
			}
		}
	}
}

func semanticDataByID(t *testing.T, namespaces []semantic.Namespace, id core.CanonicalID) semantic.Data {
	t.Helper()
	for _, namespace := range namespaces {
		for _, data := range namespace.Data {
			if data.ID == id {
				return data
			}
		}
	}
	t.Fatalf("Semantic Data %sが見つかりません", id)
	return semantic.Data{}
}
