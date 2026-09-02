package validation

import (
	"testing"

	"tei/internal/semantic"
)

func TestValidateSemanticRelationKeysRejectsMultipleSourcesForSameTargetAttribute(t *testing.T) {
	world := relationKeyWorld(
		semantic.RelationKey{
			SourceAttribute: "sales.order.customer_id",
			TargetAttribute: "customer.customer.customer_id",
		},
		semantic.RelationKey{
			SourceAttribute: "sales.order.alternate_customer_id",
			TargetAttribute: "customer.customer.customer_id",
		},
	)
	world.Semantics[0].Data[0].Attributes = append(
		world.Semantics[0].Data[0].Attributes,
		semantic.Attribute{
			ID:      "sales.order.alternate_customer_id",
			Symbol:  "alternateCustomerId",
			TypeRef: "customer.customer_id",
		},
	)

	diagnostics := ValidateSemanticRelationKeys(world)
	diagnostic := diagnosticByCode(diagnostics, "DUPLICATE_SEMANTIC_RELATION_KEY_TARGET_ATTRIBUTE")
	if diagnostic == nil {
		t.Fatalf("同じtarget attributeへ複数source attributeを対応付ける曖昧さを拒否していません: %#v", diagnostics)
	}
	if diagnostic.Subject == nil || diagnostic.Subject.Kind != "semantic-relation" || diagnostic.Subject.ID != "sales.order.customer" {
		t.Fatalf("DiagnosticがRelationのauthoring identityへ結び付いていません: %#v", diagnostic)
	}
}
