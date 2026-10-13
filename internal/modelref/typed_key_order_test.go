package modelref_test

import (
	"reflect"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

// Distinct typed keys may have identical delimiter-joined representations.
// Diff and impact reviews must still have a total, predictable ordering.
func TestTypedKeyDiffAndImpactOrderingWithEmbeddedSeparator(t *testing.T) {
	a := modelref.Key{Scope: "orders", Subject: subject.Subject{Kind: "data", ID: "x\x00y"}}
	b := modelref.Key{Scope: "orders", Subject: subject.Subject{Kind: "data\x00x", ID: "y"}}
	consumer := modelref.Key{Scope: "orders", Subject: subject.Subject{Kind: "function", ID: "processor"}}
	before := modelref.Graph{
		Schema: "tei.modelref.v0",
		Nodes:  []modelref.Node{
			{Key: a, Version: "1"},
			{Key: b, Version: "1"},
			{Key: consumer},
		},
		Edges: []modelref.Edge{
			{From: consumer, To: modelref.Ref{Subject: a.Subject}, Relation: "reads"},
			{From: consumer, To: modelref.Ref{Subject: b.Subject}, Relation: "reads"},
		},
	}
	rules := []modelref.Rule{
		{Relation: "reads", FromKind: consumer.Subject.Kind, ToKind: a.Subject.Kind},
		{Relation: "reads", FromKind: consumer.Subject.Kind, ToKind: b.Subject.Kind},
	}
	if issues := modelref.Validate(before, rules); len(issues) != 0 {
		t.Fatalf("valid distinct subjects rejected: %+v", issues)
	}
	after := before
	after.Nodes = append([]modelref.Node(nil), before.Nodes...)
	after.Nodes[0].Version = "2"
	after.Nodes[1].Version = "2"

	for i := 0; i < 32; i++ {
		changes := modelref.Diff(before, after)
		if len(changes) != 2 || changes[0].Key != a || changes[1].Key != b ||
			changes[0].Kind != "version-changed" || changes[1].Kind != "version-changed" {
			t.Fatalf("iteration %d: distinct keys did not receive stable Diff order: %+v", i, changes)
		}
	}
	// The caller may supply change requests in any order.
	reversed := []modelref.Change{
		{Key: b, Kind: "version-changed"},
		{Key: a, Kind: "version-changed"},
	}
	affected := modelref.Impact(before, reversed, rules)
	if len(affected) != 2 || affected[0].Changed != a || affected[1].Changed != b ||
		affected[0].Consumer != consumer || affected[1].Consumer != consumer {
		t.Fatalf("impact order collided between distinct subject keys: %+v", affected)
	}
	review := modelref.ReviewImpactDiff(before, after, rules)
	if !review.GraphValid || len(review.Diagnostics) != 0 ||
		len(review.Changes) != 2 || len(review.Affected) != 2 ||
		review.Changes[0].Key != a || review.Changes[1].Key != b ||
		review.Affected[0].Changed != a || review.Affected[1].Changed != b {
		t.Fatalf("two-snapshot impact ordering is not stable: %+v", review)
	}
}

// Incoming TraceDirect rows use From as part of their stable sort key.
// Neither the edge input order nor paging may collapse distinct From subjects.
func TestTypedKeyTraceOrderingWithEmbeddedSeparator(t *testing.T) {
	a := modelref.Key{Scope: "orders", Subject: subject.Subject{Kind: "function", ID: "x\x00y"}}
	b := modelref.Key{Scope: "orders", Subject: subject.Subject{Kind: "function\x00x", ID: "y"}}
	target := modelref.Key{Scope: "orders", Subject: subject.Subject{Kind: "data", ID: "record"}}
	graph := modelref.Graph{
		Schema: "tei.modelref.v0",
		Nodes:  []modelref.Node{
			{Key: a}, {Key: b}, {Key: target},
		},
		Edges: []modelref.Edge{
			{From: b, To: modelref.Ref{Scope: target.Scope, Subject: target.Subject}, Relation: "reads"},
			{From: a, To: modelref.Ref{Scope: target.Scope, Subject: target.Subject}, Relation: "reads"},
		},
	}
	rules := []modelref.Rule{
		{Relation: "reads", FromKind: a.Subject.Kind, ToKind: target.Subject.Kind},
		{Relation: "reads", FromKind: b.Subject.Kind, ToKind: target.Subject.Kind},
	}
	if findings := modelref.Validate(graph, rules); len(findings) != 0 {
		t.Fatalf("valid graph unexpectedly failed validation: %+v", findings)
	}
	first, err := modelref.TraceDirect(graph, rules, target, modelref.TraceOptions{Direction: "incoming"})
	if err != nil || first.Total != 2 || first.Unresolved != 0 || len(first.Rows) != 2 ||
		(first.Rows[0].Edge.From != a && first.Rows[0].Edge.From != b) ||
		first.Rows[0].Edge.From == first.Rows[1].Edge.From {
		t.Fatalf("trace did not sort distinct From identities: %+v, err=%v", first, err)
	}
	page, err := modelref.TraceDirect(graph, rules, target, modelref.TraceOptions{
		Direction: "incoming", Limit: 1, ExpectedFingerprint: first.SnapshotFingerprint,
	})
	if err != nil || len(page.Rows) != 1 || page.Rows[0].Edge.From != first.Rows[0].Edge.From ||
		page.NextOffset == nil || *page.NextOffset != 1 {
		t.Fatalf("first trace page changed meaning: %+v, err=%v", page, err)
	}
	graph.Edges[0], graph.Edges[1] = graph.Edges[1], graph.Edges[0]
	second, err := modelref.TraceDirect(graph, rules, target, modelref.TraceOptions{
		Direction: "incoming", ExpectedFingerprint: first.SnapshotFingerprint,
	})
	if err != nil || first.SnapshotFingerprint != second.SnapshotFingerprint ||
		!reflect.DeepEqual(first.Rows, second.Rows) {
		t.Fatalf("permuting declarations changed trace order: first=%+v second=%+v err=%v", first.Rows, second.Rows, err)
	}
}
