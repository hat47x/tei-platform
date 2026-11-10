package derivedbridge

import (
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/subject"
)

// A `realize X with custom go` reaches the graph twice: index.go records it as a
// DeclaredDependency, and projection.go also walks CustomGoBindings. Appending
// both reported one incoming edge as two rows, which read as "two applications
// implement this capability" (#4024).
func TestFromIndexCountsACustomGoBindingOnce(t *testing.T) {
	d := index.Derived{
		SemanticElements: []index.SemanticElementEntry{
			{Kind: "capability", ID: "security.authenticate"},
		},
		// The application end must be a node, or the edge is reported as
		// unmaterialized instead of being counted.
		ApplicationPatterns: []index.ApplicationPatternIndexEntry{
			{Application: "order_entry", Pattern: "order_entry.pattern"},
		},
		// index.go sets both of these for one binding.
		DeclaredDependencies: []index.DeclaredDependency{
			{FromKind: "application", From: "order_entry", Relation: "realizes",
				ToKind: "semantic-capability", To: "security.authenticate"},
		},
		CustomGoBindings: []index.BindingIndexEntry{
			{Application: "order_entry", Capability: "security.authenticate"},
		},
	}
	p := FromIndex(d, "c")
	if len(p.Graph.Edges) != 1 {
		t.Fatalf("one binding produced %d edges, want 1: %+v", len(p.Graph.Edges), p.Graph.Edges)
	}
	edge := p.Graph.Edges[0]
	if edge.Relation != "realizes" ||
		edge.From.Subject.Kind != "application" || edge.From.Subject.ID != "order_entry" ||
		edge.To.Subject.Kind != "semantic-capability" || edge.To.Subject.ID != "security.authenticate" {
		t.Fatalf("the kept edge is not the declared binding: %+v", edge)
	}
}

// The same statement written twice is still one statement. This is not specific
// to realizes: any dependency repeated in the index must collapse, otherwise
// every count a caller takes from the graph is too large.
func TestFromIndexCollapsesRepeatedDependencies(t *testing.T) {
	dependency := index.SemanticDependency{
		FromKind: "semantic-data", From: "sales.order",
		Relation: "targets", ToKind: "semantic-data", To: "sales.order_line",
	}
	d := index.Derived{
		SemanticElements: []index.SemanticElementEntry{
			{Kind: "data", ID: "sales.order"}, {Kind: "data", ID: "sales.order_line"},
		},
		SemanticDependencies: []index.SemanticDependency{dependency, dependency, dependency},
		DeclaredDependencies: []index.DeclaredDependency{{
			FromKind: dependency.FromKind, From: dependency.From, Relation: dependency.Relation,
			ToKind: dependency.ToKind, To: dependency.To,
		}},
	}
	p := FromIndex(d, "c")
	if len(p.Graph.Edges) != 1 {
		t.Fatalf("four statements of one edge produced %d edges: %+v", len(p.Graph.Edges), p.Graph.Edges)
	}
	if len(p.Rules) != 1 {
		t.Fatalf("the rule was recorded %d times, want 1: %+v", len(p.Rules), p.Rules)
	}
}

// Dependencies that differ only in a field count separately: collapsing must not
// merge distinct statements that happen to share endpoints.
func TestFromIndexKeepsDependenciesThatDifferInRelation(t *testing.T) {
	d := index.Derived{
		SemanticElements: []index.SemanticElementEntry{
			{Kind: "data", ID: "sales.order"}, {Kind: "data", ID: "sales.order_line"},
		},
		SemanticDependencies: []index.SemanticDependency{
			{FromKind: "semantic-data", From: "sales.order", Relation: "targets",
				ToKind: "semantic-data", To: "sales.order_line"},
			{FromKind: "semantic-data", From: "sales.order", Relation: "measures",
				ToKind: "semantic-data", To: "sales.order_line"},
		},
	}
	p := FromIndex(d, "c")
	if len(p.Graph.Edges) != 2 {
		t.Fatalf("two relations between the same pair produced %d edges, want 2: %+v", len(p.Graph.Edges), p.Graph.Edges)
	}
}

// An unresolved reference repeated by several declarations is one finding. The
// message names the reference and nothing else, so extra copies only inflate the
// count a reader takes from projectionUnresolved.
func TestFromIndexCollapsesRepeatedUnresolvedReferences(t *testing.T) {
	missing := index.SemanticDependency{
		FromKind: "semantic-data", From: "sales.order",
		Relation: "targets", ToKind: "semantic-data", To: "sales.absent",
	}
	d := index.Derived{
		SemanticElements: []index.SemanticElementEntry{{Kind: "data", ID: "sales.order"}},
		SemanticDependencies: []index.SemanticDependency{
			missing, missing,
			// A second reference to the same absent target is the same finding.
			{FromKind: "semantic-data", From: "sales.order", Relation: "measures",
				ToKind: "semantic-data", To: "sales.absent"},
		},
	}
	p := FromIndex(d, "c")
	for _, message := range p.Unresolved {
		if message != "unmaterialized-to:semantic-data:sales.absent" {
			t.Fatalf("unexpected unresolved message: %q", message)
		}
	}
	if len(p.Unresolved) != 1 {
		t.Fatalf("one absent target was reported %d times: %+v", len(p.Unresolved), p.Unresolved)
	}
	if len(p.Graph.Edges) != 0 {
		t.Fatalf("an edge to an absent target was kept: %+v", p.Graph.Edges)
	}
}

// The duplicate must not come back through the trace projection either: this is
// what a caller actually reads.
func TestFromIndexEdgeIdentityIncludesEveryPart(t *testing.T) {
	base := modelref.Edge{Relation: "realizes"}
	_ = base
	d := index.Derived{
		SemanticElements: []index.SemanticElementEntry{
			{Kind: "data", ID: "a"}, {Kind: "type", ID: "a"},
		},
		SemanticDependencies: []index.SemanticDependency{
			// Same ID "a" with two kinds: distinct edges, both kept.
			{FromKind: "semantic-data", From: "a", Relation: "targets", ToKind: "semantic-type", To: "a"},
			{FromKind: "semantic-type", From: "a", Relation: "targets", ToKind: "semantic-data", To: "a"},
		},
	}
	p := FromIndex(d, "c")
	if len(p.Graph.Edges) != 2 {
		t.Fatalf("edges differing only in kind produced %d, want 2: %+v", len(p.Graph.Edges), p.Graph.Edges)
	}
	seen := map[modelref.Key]bool{}
	for _, edge := range p.Graph.Edges {
		key := modelref.Key{Scope: edge.From.Scope, Subject: subject.Subject{Kind: edge.From.Subject.Kind, ID: edge.From.Subject.ID}}
		if seen[key] {
			t.Fatalf("duplicate from key after projection: %+v", key)
		}
		seen[key] = true
	}
}
