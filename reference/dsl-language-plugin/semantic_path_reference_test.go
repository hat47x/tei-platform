package languagereference

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
)

func TestSemanticPathValidation(t *testing.T) {
	ns := semantic.Namespace{ID: core.CanonicalID("sales"), Data: []semantic.Data{{
		ID: core.CanonicalID("sales.order"),
		Attributes: []semantic.Attribute{
			{ID: core.CanonicalID("sales.order.amount"), TypeRef: core.CanonicalID("std.decimal")},
		},
		Relations: []semantic.Relation{
			{ID: core.CanonicalID("sales.order.lines"), Target: core.CanonicalID("sales.line"), Cardinality: semantic.CardinalityMany},
		},
	}}}
	for _, tc := range []struct {
		name string
		path SemanticPath
		want semantic.Cardinality
	}{
		{"data", SemanticPath{DataID: core.CanonicalID("sales.order")}, semantic.CardinalityMany},
		{"attribute", SemanticPath{DataID: core.CanonicalID("sales.order"), AttributeID: core.CanonicalID("sales.order.amount")}, semantic.CardinalityOne},
		{"relation", SemanticPath{DataID: core.CanonicalID("sales.order"), RelationID: core.CanonicalID("sales.order.lines")}, semantic.CardinalityMany},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateSemanticPath(ns, tc.path)
			if err != nil || got != tc.want {
				t.Fatalf("got %q: %v", got, err)
			}
		})
	}
	for _, tc := range []struct {
		name string
		ns   semantic.Namespace
		path SemanticPath
	}{
		{"missing data", ns, SemanticPath{DataID: core.CanonicalID("sales.missing")}},
		{"missing attribute", ns, SemanticPath{DataID: core.CanonicalID("sales.order"), AttributeID: core.CanonicalID("sales.order.missing")}},
		{"missing relation", ns, SemanticPath{DataID: core.CanonicalID("sales.order"), RelationID: core.CanonicalID("sales.order.missing")}},
		{"ambiguous path", ns, SemanticPath{DataID: core.CanonicalID("sales.order"), AttributeID: core.CanonicalID("sales.order.amount"), RelationID: core.CanonicalID("sales.order.lines")}},
		{"invalid cardinality", semantic.Namespace{ID: ns.ID, Data: []semantic.Data{{ID: core.CanonicalID("sales.order"), Relations: []semantic.Relation{{ID: core.CanonicalID("sales.order.lines")}}}}}, SemanticPath{DataID: core.CanonicalID("sales.order"), RelationID: core.CanonicalID("sales.order.lines")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ValidateSemanticPath(tc.ns, tc.path); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}
