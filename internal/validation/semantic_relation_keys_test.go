package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateSemanticRelationKeysAcceptsExplicitCustomerKeyMapping(t *testing.T) {
	world := relationKeyWorld(semantic.RelationKey{
		SourceAttribute: "sales.order.customer_id",
		TargetAttribute: "customer.customer.customer_id",
	})

	if got := ValidateSemanticRelationKeys(world); len(got) != 0 {
		t.Fatalf("正しいRelation key mappingでDiagnosticが発生しました: %#v", got)
	}
}

func TestValidateSemanticRelationKeysKeepsKeylessRelationValid(t *testing.T) {
	world := relationKeyWorld()

	if got := ValidateSemanticRelationKeys(world); len(got) != 0 {
		t.Fatalf("key未定義Relationの既存契約を壊してはいけません: %#v", got)
	}
}

func TestValidateSemanticRelationKeysRejectsInvalidMappings(t *testing.T) {
	tests := []struct {
		name string
		key  semantic.RelationKey
		code string
	}{
		{
			name: "source attribute is missing",
			key: semantic.RelationKey{
				TargetAttribute: "customer.customer.customer_id",
			},
			code: "SEMANTIC_RELATION_KEY_SOURCE_REQUIRED",
		},
		{
			name: "target attribute is missing",
			key: semantic.RelationKey{
				SourceAttribute: "sales.order.customer_id",
			},
			code: "SEMANTIC_RELATION_KEY_TARGET_REQUIRED",
		},
		{
			name: "unknown source attribute",
			key: semantic.RelationKey{
				SourceAttribute: "sales.order.missing",
				TargetAttribute: "customer.customer.customer_id",
			},
			code: "UNKNOWN_SEMANTIC_RELATION_KEY_SOURCE_ATTRIBUTE",
		},
		{
			name: "source attribute belongs to another data",
			key: semantic.RelationKey{
				SourceAttribute: "customer.customer.customer_id",
				TargetAttribute: "customer.customer.customer_id",
			},
			code: "SEMANTIC_RELATION_KEY_SOURCE_PARENT_MISMATCH",
		},
		{
			name: "unknown target attribute",
			key: semantic.RelationKey{
				SourceAttribute: "sales.order.customer_id",
				TargetAttribute: "customer.customer.missing",
			},
			code: "UNKNOWN_SEMANTIC_RELATION_KEY_TARGET_ATTRIBUTE",
		},
		{
			name: "target attribute belongs to another data",
			key: semantic.RelationKey{
				SourceAttribute: "sales.order.customer_id",
				TargetAttribute: "sales.order.customer_id",
			},
			code: "SEMANTIC_RELATION_KEY_TARGET_PARENT_MISMATCH",
		},
		{
			name: "semantic type identity differs",
			key: semantic.RelationKey{
				SourceAttribute: "sales.order.order_id",
				TargetAttribute: "customer.customer.customer_id",
			},
			code: "SEMANTIC_RELATION_KEY_TYPE_MISMATCH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := ValidateSemanticRelationKeys(relationKeyWorld(tt.key))
			diagnostic := diagnosticByCode(diagnostics, tt.code)
			if diagnostic == nil {
				t.Fatalf("Diagnostic %sがありません: %#v", tt.code, diagnostics)
			}
			if diagnostic.Subject == nil || diagnostic.Subject.Kind != "semantic-relation" || diagnostic.Subject.ID != "sales.order.customer" {
				t.Fatalf("DiagnosticがRelationのauthoring identityへ結び付いていません: %#v", diagnostic)
			}
		})
	}
}

func TestValidateSemanticRelationKeysRejectsDuplicatePair(t *testing.T) {
	key := semantic.RelationKey{
		SourceAttribute: "sales.order.customer_id",
		TargetAttribute: "customer.customer.customer_id",
	}
	world := relationKeyWorld(key, key)

	diagnostics := ValidateSemanticRelationKeys(world)
	if diagnosticByCode(diagnostics, "DUPLICATE_SEMANTIC_RELATION_KEY") == nil {
		t.Fatalf("重複Relation key mappingを拒否していません: %#v", diagnostics)
	}
}

func relationKeyWorld(keys ...semantic.RelationKey) ir.World {
	return ir.World{Semantics: []semantic.Namespace{
		{
			ID: "sales",
			Types: []semantic.Type{
				{ID: "sales.order_id", Symbol: "OrderId", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
			},
			Data: []semantic.Data{
				{
					ID:     "sales.order",
					Symbol: "Order",
					Attributes: []semantic.Attribute{
						{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "sales.order_id"},
						{ID: "sales.order.customer_id", Symbol: "customerId", TypeRef: "customer.customer_id"},
					},
					Relations: []semantic.Relation{
						{
							ID:          "sales.order.customer",
							Symbol:      "customer",
							Target:      "customer.customer",
							Cardinality: semantic.CardinalityOne,
							Keys:        keys,
						},
					},
				},
			},
		},
		{
			ID: "customer",
			Types: []semantic.Type{
				{ID: "customer.customer_id", Symbol: "CustomerId", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
			},
			Data: []semantic.Data{
				{
					ID:     "customer.customer",
					Symbol: "Customer",
					Attributes: []semantic.Attribute{
						{ID: "customer.customer.customer_id", Symbol: "customerId", TypeRef: "customer.customer_id"},
					},
				},
			},
		},
	}}
}
