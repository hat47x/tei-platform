package derivedbridge_test

import (
	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"testing"
)

func TestCanonicalIndexProjectionKeepsTypedReferences(t *testing.T) {
	d := index.Derived{
		SemanticElements:     []index.SemanticElementEntry{{Kind: "data", ID: "sales.order_line"}, {Kind: "data", ID: "sales.order"}, {Kind: "relation", ID: "sales.order.lines"}},
		ApplicationPatterns:  []index.ApplicationPatternIndexEntry{{Application: "order_entry", Pattern: "order_entry.submit"}},
		SemanticDependencies: []index.SemanticDependency{{FromKind: "semantic-relation", From: "sales.order.lines", Relation: "targets", ToKind: "semantic-data", To: "sales.order_line"}},
		DeclaredDependencies: []index.DeclaredDependency{
			{FromKind: "application-pattern", From: "order_entry.submit", Relation: "data", ToKind: "semantic-data", To: "sales.order"},
			{FromKind: "application", From: "order_entry", Relation: "owns-pattern", ToKind: "application-pattern", To: "order_entry.submit"},
		},
	}
	p := derivedbridge.FromIndex(d, "order-workspace")
	if len(p.Unresolved) > 0 || len(p.Graph.Nodes) != 5 || len(p.Graph.Edges) != 3 {
		t.Fatalf("lost exact refs: %+v", p)
	}
	if ds := modelref.Validate(p.Graph, p.Rules); len(ds) > 0 {
		t.Fatalf("invalid typed index projection: %+v", ds)
	}
	for _, e := range p.Graph.Edges {
		if e.To.Scope != "order-workspace" {
			t.Fatalf("lost scope: %+v", e)
		}
	}
}
func TestUnknownDoesNotBecomeCanonical(t *testing.T) {
	d := index.Derived{SemanticElements: []index.SemanticElementEntry{{Kind: "data", ID: "sales.order"}}, SemanticDependencies: []index.SemanticDependency{{FromKind: "semantic-data", From: "sales.order", Relation: "unknown", ToKind: "semantic-data", To: "sales.not-present"}}}
	p := derivedbridge.FromIndex(d, "order-workspace")
	if len(p.Graph.Edges) != 0 || len(p.Unresolved) != 1 {
		t.Fatalf("invented dependency: %+v", p)
	}
	p = derivedbridge.FromIndex(d, "")
	if len(p.Graph.Nodes) != 0 || len(p.Unresolved) != 1 || p.Unresolved[0] != "scope-required" {
		t.Fatalf("invented scope: %+v", p)
	}
}

// SupportsSubjectKind must describe exactly what FromIndex materializes. A
// kind claimed here but never added would make `trace` answer SUBJECT_NOT_FOUND
// for a subject that the graph genuinely cannot contain (#3180).
func TestSupportsSubjectKindMatchesWhatFromIndexProjects(t *testing.T) {
	d := index.Derived{
		SemanticElements: []index.SemanticElementEntry{
			{Kind: "namespace", ID: "sales"},
			{Kind: "type", ID: "sales.code"},
			{Kind: "data", ID: "sales.order"},
			{Kind: "attribute", ID: "sales.order.order_id"},
			{Kind: "relation", ID: "sales.order.customer"},
			{Kind: "validation", ID: "sales.order.amount"},
			{Kind: "capability", ID: "sales.submit"},
		},
		ApplicationPatterns: []index.ApplicationPatternIndexEntry{{Application: "order_entry", Pattern: "order_entry.submit"}},
		Queries:             []index.QueryIndexEntry{{Query: "recent_orders"}},
	}
	p := derivedbridge.FromIndex(d, "sales-scope")
	projected := map[string]bool{}
	for _, node := range p.Graph.Nodes {
		projected[node.Key.Subject.Kind] = true
	}
	for _, kind := range []string{
		"semantic", "semantic-type", "semantic-data", "semantic-attribute",
		"semantic-relation", "semantic-validation", "semantic-capability",
		"application", "application-pattern", "query",
	} {
		if !derivedbridge.SupportsSubjectKind(kind) || !projected[kind] {
			t.Fatalf("%s: SupportsSubjectKind=%v projected=%v", kind, derivedbridge.SupportsSubjectKind(kind), projected[kind])
		}
	}
	// inspect reads these from the Canonical World, but the derived bridge has
	// no edge for them, so trace must refuse the kind instead of the ID.
	for _, kind := range []string{"semantic-candidate-key", "semantic-functional-dependency", "unknown-kind"} {
		if derivedbridge.SupportsSubjectKind(kind) || projected[kind] {
			t.Fatalf("%s: unprojected kind was accepted", kind)
		}
	}
}

func TestAmbiguousGenericSemanticReferenceNeverGuessed(t *testing.T) {
	d := index.Derived{
		SemanticElements: []index.SemanticElementEntry{
			{Kind: "type", ID: "sales.shared"},
			{Kind: "data", ID: "sales.shared"},
			{Kind: "capability", ID: "sales.consumer"},
		},
		SemanticDependencies: []index.SemanticDependency{
			{FromKind: "semantic-capability", From: "sales.consumer", Relation: "input", ToKind: "semantic-element", To: "sales.shared"},
		},
	}
	p := derivedbridge.FromIndex(d, "sales-scope")
	if len(p.Graph.Edges) != 0 || len(p.Unresolved) != 1 ||
		p.Unresolved[0] != "unresolved-semantic-element:sales.shared" {
		t.Fatalf("generic semantic identity was guessed: %+v", p)
	}
}
