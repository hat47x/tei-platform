package languagereference

import (
	"tei/internal/core"
	"tei/internal/semantic"
	"testing"
)

func TestExpandRelationReference(t *testing.T) {
	src := semantic.Data{ID: core.CanonicalID("sales.order"), Attributes: []semantic.Attribute{{ID: core.CanonicalID("sales.order.id"), TypeRef: core.CanonicalID("std.int")}}, Relations: []semantic.Relation{{ID: core.CanonicalID("sales.order.lines"), Target: core.CanonicalID("sales.line"), Cardinality: semantic.CardinalityMany, Keys: []semantic.RelationKey{{SourceAttribute: core.CanonicalID("sales.order.id"), TargetAttribute: core.CanonicalID("sales.line.order_id")}}}}}
	target := semantic.Data{ID: core.CanonicalID("sales.line"), Attributes: []semantic.Attribute{{ID: core.CanonicalID("sales.line.order_id"), TypeRef: core.CanonicalID("std.int")}}}
	ns := semantic.Namespace{ID: core.CanonicalID("sales"), Data: []semantic.Data{src, target}}
	pin := OperatorContract{ID: "std.op.expand", Version: "1", Fingerprint: "expand-v1", Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/sales.line"}
	call := PinnedCall{Contract: pin, InputType: pin.InputType}
	out, err := ExpandRelationReference(ns, src.ID, src.Relations[0].ID, call, []OperatorContract{pin})
	if err != nil || out != pin.OutputType {
		t.Fatalf("got %q: %v", out, err)
	}
	tests := []struct {
		name    string
		mutate  func(*semantic.Namespace, *PinnedCall)
		catalog []OperatorContract
	}{
		{"conceptual relation", func(n *semantic.Namespace, c *PinnedCall) { n.Data[0].Relations[0].Keys = nil }, []OperatorContract{pin}},
		{"wrong target", func(n *semantic.Namespace, c *PinnedCall) {
			n.Data[0].Relations[0].Target = core.CanonicalID("sales.missing")
		}, []OperatorContract{pin}},
		{"wrong operator", func(n *semantic.Namespace, c *PinnedCall) { c.Contract.ID = "std.op.where" }, []OperatorContract{pin}},
		{"wrong output", func(n *semantic.Namespace, c *PinnedCall) { c.Contract.OutputType = "rows/other" }, []OperatorContract{pin}},
		{"unavailable pin", func(n *semantic.Namespace, c *PinnedCall) {}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clone := semantic.Namespace{ID: ns.ID, Data: []semantic.Data{ns.Data[0], ns.Data[1]}}
			clone.Data[0].Relations = append([]semantic.Relation(nil), src.Relations...)
			clone.Data[0].Relations[0].Keys = append([]semantic.RelationKey(nil), src.Relations[0].Keys...)
			c := call
			tc.mutate(&clone, &c)
			if _, err := ExpandRelationReference(clone, src.ID, src.Relations[0].ID, c, tc.catalog); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}
