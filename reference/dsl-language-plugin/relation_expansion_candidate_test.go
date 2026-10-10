package languagereference

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func TestPinnedExpansionKeepsDistinctRelationsWithSameEndpoints(t *testing.T) {
	w := crossNamespaceRelationFixture()
	w.Semantics[0].Data[0].Attributes = append(
		w.Semantics[0].Data[0].Attributes,
		semantic.Attribute{ID: "sales.order.alternate_customer_id", TypeRef: "customer.customer_id"},
	)
	w.Semantics[0].Data[0].Relations = append(
		w.Semantics[0].Data[0].Relations,
		semantic.Relation{
			ID:          "sales.order.alternate_customer",
			Target:      "customer.customer",
			Cardinality: semantic.CardinalityOne,
			Keys: []semantic.RelationKey{{
				SourceAttribute: "sales.order.alternate_customer_id",
				TargetAttribute: "customer.customer.id",
			}},
		},
	)
	if ds := validation.Validate(w); validation.HasErrors(ds) {
		t.Fatalf("fixture must satisfy the canonical world rules: %#v", ds)
	}
	if ds := validation.ValidateSemanticRelationKeys(w); len(ds) != 0 {
		t.Fatalf("fixture keys must satisfy canonical validation: %#v", ds)
	}

	pin := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "expand-v1",
		Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/customer.customer",
	}
	call := PinnedCall{Contract: pin, InputType: pin.InputType}
	first, err := InspectPinnedRelationExpansion(w, "sales.order", "sales.order.customer", call, []OperatorContract{pin})
	if err != nil {
		t.Fatal(err)
	}
	second, err := InspectPinnedRelationExpansion(w, "sales.order", "sales.order.alternate_customer", call, []OperatorContract{pin})
	if err != nil {
		t.Fatal(err)
	}
	if first.OutputType != second.OutputType || first.RelationID == second.RelationID {
		t.Fatalf("relation identity lost: first=%#v second=%#v", first, second)
	}
	if first.Keys[0].SourceAttribute == second.Keys[0].SourceAttribute {
		t.Fatal("different key semantics collapsed into one binding")
	}
	if first.TargetID != "customer.customer" || second.TargetID != first.TargetID ||
		first.Cardinality != semantic.CardinalityOne || second.Operator != pin {
		t.Fatalf("unexpected candidate: first=%#v second=%#v", first, second)
	}
	w.Semantics[0].Data[0].Relations[1].Keys[0].SourceAttribute = core.CanonicalID("sales.order.changed")
	if second.Keys[0].SourceAttribute != "sales.order.alternate_customer_id" {
		t.Fatal("candidate keys alias mutable semantic World")
	}
}

func TestPinnedExpansionRequiresExecutableAndAcceptedContracts(t *testing.T) {
	w := crossNamespaceRelationFixture()
	pin := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "expand-v1",
		Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/customer.customer",
	}
	call := PinnedCall{Contract: pin, InputType: pin.InputType}
	for _, tc := range []struct {
		name    string
		prepare func()
		catalog []OperatorContract
	}{
		{"unavailable pin", func() {}, nil},
		{"conceptual relation", func() { w.Semantics[0].Data[0].Relations[0].Keys = nil }, []OperatorContract{pin}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := crossNamespaceRelationFixture()
			if tc.name == "conceptual relation" {
				world.Semantics[0].Data[0].Relations[0].Keys = nil
			}
			if _, err := InspectPinnedRelationExpansion(world, "sales.order", "sales.order.customer", call, tc.catalog); err == nil {
				t.Fatal("expected a fail-closed relation candidate")
			}
		})
	}
}
