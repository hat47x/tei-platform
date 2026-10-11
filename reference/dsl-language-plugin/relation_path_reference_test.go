package languagereference

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func TestTraceRelationPathMultiplicity(t *testing.T) {
	ns := relationPathFixture()
	world := ir.World{Semantics: []semantic.Namespace{ns}}
	if ds := validation.Validate(world); validation.HasErrors(ds) {
		t.Fatalf("fixture is not a valid canonical world: %#v", ds)
	}
	if ds := validation.ValidateSemanticRelationKeys(world); len(ds) != 0 {
		t.Fatalf("fixture relation keys are invalid: %#v", ds)
	}
	for _, tc := range []struct {
		name      string
		path      []core.CanonicalID
		target    core.CanonicalID
		wantCount semantic.Cardinality
	}{
		{"one hop", []core.CanonicalID{"shop.order.customer"}, "shop.customer", semantic.CardinalityOne},
		{"one then many", []core.CanonicalID{"shop.order.customer", "shop.customer.addresses"}, "shop.address", semantic.CardinalityMany},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shape, err := TraceRelationPath(world, "shop.order", tc.path)
			if err != nil || shape.TargetID != tc.target || shape.SourceID != "shop.order" || shape.Multiplicity != tc.wantCount {
				t.Fatalf("unexpected shape %#v: %v", shape, err)
			}
		})
	}
}

func TestTraceRelationPathRejectsInvalidHops(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*semantic.Namespace)
		start  core.CanonicalID
		path   []core.CanonicalID
	}{
		{"empty path", nil, "shop.order", nil},
		{"missing source", nil, "shop.missing", []core.CanonicalID{"shop.order.customer"}},
		{"wrong second hop", nil, "shop.order", []core.CanonicalID{"shop.order.customer", "shop.order.customer"}},
		{"conceptual relation", func(ns *semantic.Namespace) { ns.Data[0].Relations[0].Keys = nil }, "shop.order", []core.CanonicalID{"shop.order.customer"}},
		{"missing target", func(ns *semantic.Namespace) { ns.Data = ns.Data[:1] }, "shop.order", []core.CanonicalID{"shop.order.customer"}},
		{"key type mismatch", func(ns *semantic.Namespace) { ns.Data[1].Attributes[0].TypeRef = "shop.other_key" }, "shop.order", []core.CanonicalID{"shop.order.customer"}},
		{"invalid cardinality", func(ns *semantic.Namespace) { ns.Data[0].Relations[0].Cardinality = "unknown" }, "shop.order", []core.CanonicalID{"shop.order.customer"}},
		{"duplicate target", func(ns *semantic.Namespace) { ns.Data = append(ns.Data, ns.Data[1]) }, "shop.order", []core.CanonicalID{"shop.order.customer"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ns := relationPathFixture()
			if tc.mutate != nil {
				tc.mutate(&ns)
			}
			if _, err := TraceRelationPath(ir.World{Semantics: []semantic.Namespace{ns}}, tc.start, tc.path); err == nil {
				t.Fatal("expected invalid relation path to be rejected")
			}
		})
	}
}

func relationPathFixture() semantic.Namespace {
	return semantic.Namespace{
		ID:    "shop",
		Types: []semantic.Type{{ID: "shop.key", Symbol: "Key", Kind: semantic.TypePrimitive}},
		Data: []semantic.Data{
			{
				ID:         "shop.order",
				Attributes: []semantic.Attribute{{ID: "shop.order.customer_id", TypeRef: "shop.key"}},
				Relations: []semantic.Relation{{
					ID: "shop.order.customer", Target: "shop.customer", Cardinality: semantic.CardinalityOne,
					Keys: []semantic.RelationKey{{SourceAttribute: "shop.order.customer_id", TargetAttribute: "shop.customer.id"}},
				}},
			},
			{
				ID:         "shop.customer",
				Attributes: []semantic.Attribute{{ID: "shop.customer.id", TypeRef: "shop.key"}},
				Relations: []semantic.Relation{{
					ID: "shop.customer.addresses", Target: "shop.address", Cardinality: semantic.CardinalityMany,
					Keys: []semantic.RelationKey{{SourceAttribute: "shop.customer.id", TargetAttribute: "shop.address.customer_id"}},
				}},
			},
			{
				ID:         "shop.address",
				Attributes: []semantic.Attribute{{ID: "shop.address.customer_id", TypeRef: "shop.key"}},
			},
		},
	}
}
