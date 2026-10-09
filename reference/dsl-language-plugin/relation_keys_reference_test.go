package languagereference

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
)

func TestRelationKeysReference(t *testing.T) {
	source := semantic.Data{ID: core.CanonicalID("sales.order"), Attributes: []semantic.Attribute{
		{ID: core.CanonicalID("sales.order.id"), TypeRef: core.CanonicalID("std.int")},
	}, Relations: []semantic.Relation{{
		ID: core.CanonicalID("sales.order.lines"), Target: core.CanonicalID("sales.line"), Cardinality: semantic.CardinalityMany,
		Keys: []semantic.RelationKey{{SourceAttribute: core.CanonicalID("sales.order.id"), TargetAttribute: core.CanonicalID("sales.line.order_id")}},
	}}}
	target := semantic.Data{ID: core.CanonicalID("sales.line"), Attributes: []semantic.Attribute{
		{ID: core.CanonicalID("sales.line.order_id"), TypeRef: core.CanonicalID("std.int")},
	}}
	ns := semantic.Namespace{ID: core.CanonicalID("sales"), Data: []semantic.Data{source, target}}
	if err := ValidateRelationKeys(ns, source.ID, source.Relations[0].ID); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*semantic.Namespace)
	}{
		{"missing target", func(n *semantic.Namespace) { n.Data = n.Data[:1] }},
		{"type mismatch", func(n *semantic.Namespace) { n.Data[1].Attributes[0].TypeRef = core.CanonicalID("std.text") }},
		{"missing source key", func(n *semantic.Namespace) { n.Data[0].Relations[0].Keys[0].SourceAttribute = core.CanonicalID("sales.order.unknown") }},
		{"duplicate target", func(n *semantic.Namespace) { n.Data = append(n.Data, n.Data[1]) }},
		{"invalid cardinality", func(n *semantic.Namespace) { n.Data[0].Relations[0].Cardinality = "unknown" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clone := semantic.Namespace{ID: ns.ID, Data: []semantic.Data{ns.Data[0], ns.Data[1]}}
			clone.Data[0].Relations = append([]semantic.Relation(nil), ns.Data[0].Relations...)
			clone.Data[0].Relations[0].Keys = append([]semantic.RelationKey(nil), ns.Data[0].Relations[0].Keys...)
			clone.Data[1].Attributes = append([]semantic.Attribute(nil), ns.Data[1].Attributes...)
			tc.change(&clone)
			if err := ValidateRelationKeys(clone, source.ID, source.Relations[0].ID); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}
