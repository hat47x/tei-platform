package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestReferenceOrderLineExistsAsSemanticDetailWithoutPersistenceAssumption(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	orderLine := semanticDataByID(t, result.World.Semantics, "sales.order_line")
	if len(orderLine.Attributes) != 2 {
		t.Fatalf("OrderLineはheader-detail実証に必要な最小属性だけを持つ想定です: %#v", orderLine.Attributes)
	}
	orderLineID := semanticAttributeByID(t, orderLine.Attributes, "sales.order_line.order_line_id")
	if orderLineID.TypeRef != "sales.order_line_id" {
		t.Fatalf("OrderLine識別子の型が期待値と異なります: %#v", orderLineID)
	}
	amount := semanticAttributeByID(t, orderLine.Attributes, "sales.order_line.amount")
	if amount.TypeRef != "std.money" {
		t.Fatalf("OrderLine金額の型が期待値と異なります: %#v", amount)
	}

	source, lines := semanticRelationByID(t, result.World.Semantics, "sales.order.lines")
	if source != "sales.order" || lines.Target != "sales.order_line" || lines.Cardinality != semantic.CardinalityMany {
		t.Fatalf("OrderからOrderLineへの1:N Relationが期待値と異なります: source=%s relation=%#v", source, lines)
	}

	for _, mapping := range result.World.Persistence {
		if mapping.SemanticData == "sales.order_line" {
			t.Fatalf("Semantic Relationの追加だけからOrderLineのPersistence Mappingが暗黙生成されています: %#v", mapping)
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

func semanticAttributeByID(t *testing.T, attributes []semantic.Attribute, id core.CanonicalID) semantic.Attribute {
	t.Helper()
	for _, attribute := range attributes {
		if attribute.ID == id {
			return attribute
		}
	}
	t.Fatalf("Semantic Attribute %sが見つかりません: %#v", id, attributes)
	return semantic.Attribute{}
}
