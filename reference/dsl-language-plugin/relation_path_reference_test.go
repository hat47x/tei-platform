package languagereference

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
)

func TestTraceRelationPathMultiplicity(t *testing.T) {
	ns := relationPathFixture()
	for _, tc := range []struct {
		name      string
		path      []core.CanonicalID
		target    core.CanonicalID
		wantCount semantic.Cardinality
	}{
		{"one hop", []core.CanonicalID{"order.customer"}, "customer", semantic.CardinalityOne},
		{"one then many", []core.CanonicalID{"order.customer", "customer.addresses"}, "address", semantic.CardinalityMany},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shape, err := TraceRelationPath(ns, "order", tc.path)
			if err != nil || shape.TargetID != tc.target || shape.SourceID != "order" || shape.Multiplicity != tc.wantCount {
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
		{"empty path", nil, "order", nil},
		{"missing source", nil, "missing", []core.CanonicalID{"order.customer"}},
		{"wrong second hop", nil, "order", []core.CanonicalID{"order.customer", "order.customer"}},
		{"conceptual relation", func(ns *semantic.Namespace) { ns.Data[0].Relations[0].Keys = nil }, "order", []core.CanonicalID{"order.customer"}},
		{"missing target", func(ns *semantic.Namespace) { ns.Data = ns.Data[:1] }, "order", []core.CanonicalID{"order.customer"}},
		{"key type mismatch", func(ns *semantic.Namespace) { ns.Data[1].Attributes[0].TypeRef = "std.text" }, "order", []core.CanonicalID{"order.customer"}},
		{"invalid cardinality", func(ns *semantic.Namespace) { ns.Data[0].Relations[0].Cardinality = "unknown" }, "order", []core.CanonicalID{"order.customer"}},
		{"duplicate target", func(ns *semantic.Namespace) { ns.Data = append(ns.Data, ns.Data[1]) }, "order", []core.CanonicalID{"order.customer"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ns := relationPathFixture()
			if tc.mutate != nil {
				tc.mutate(&ns)
			}
			if _, err := TraceRelationPath(ns, tc.start, tc.path); err == nil {
				t.Fatal("expected invalid relation path to be rejected")
			}
		})
	}
}

func relationPathFixture() semantic.Namespace {
	return semantic.Namespace{
		ID: "shop",
		Data: []semantic.Data{
			{
				ID:         "order",
				Attributes: []semantic.Attribute{{ID: "order.customer_id", TypeRef: "std.key"}},
				Relations: []semantic.Relation{{
					ID: "order.customer", Target: "customer", Cardinality: semantic.CardinalityOne,
					Keys: []semantic.RelationKey{{SourceAttribute: "order.customer_id", TargetAttribute: "customer.id"}},
				}},
			},
			{
				ID:         "customer",
				Attributes: []semantic.Attribute{{ID: "customer.id", TypeRef: "std.key"}},
				Relations: []semantic.Relation{{
					ID: "customer.addresses", Target: "address", Cardinality: semantic.CardinalityMany,
					Keys: []semantic.RelationKey{{SourceAttribute: "customer.id", TargetAttribute: "address.customer_id"}},
				}},
			},
			{
				ID:         "address",
				Attributes: []semantic.Attribute{{ID: "address.customer_id", TypeRef: "std.key"}},
			},
		},
	}
}
