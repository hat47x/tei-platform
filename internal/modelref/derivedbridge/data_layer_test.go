package derivedbridge_test

import (
	"strings"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref/derivedbridge"
)

// dataLayerFixture declares the three edges the index emits toward the data
// layer. None of these kinds has a dedicated entry list, so before #4068 the
// projection had no node for them and dropped every one of these edges.
func dataLayerFixture() index.Derived {
	return index.Derived{
		ApplicationPatterns: []index.ApplicationPatternIndexEntry{
			{Application: "order_entry", Pattern: "order_entry.submit"},
		},
		SemanticElements: []index.SemanticElementEntry{
			{Kind: "namespace", ID: "sales"},
		},
		DeclaredDependencies: []index.DeclaredDependency{
			{FromKind: "application", From: "order_entry", Relation: "targets-database", ToKind: "database", To: "sales_core"},
			{FromKind: "application", From: "order_entry", Relation: "requires-contract", ToKind: "database-contract", To: "sales_core.order"},
			{FromKind: "application", From: "order_entry", Relation: "contract-query", ToKind: "database-contract-query", To: "sales_core.order/find_by_id"},
			{FromKind: "database", From: "sales_core", Relation: "uses-semantic", ToKind: "semantic", To: "sales"},
		},
	}
}

// A declared edge to the data layer must be projected, not silently dropped.
// Dropping it left `trace` claiming "unresolved: 0" while the data layer was
// absent from the graph (#4068).
func TestDataLayerEdgesAreProjectedNotDropped(t *testing.T) {
	p := derivedbridge.FromIndex(dataLayerFixture(), "c")

	want := map[string]bool{
		"targets-database":  false,
		"requires-contract": false,
		"contract-query":    false,
	}
	for _, edge := range p.Graph.Edges {
		if _, tracked := want[edge.Relation]; tracked {
			want[edge.Relation] = true
		}
	}
	for relation, found := range want {
		if !found {
			t.Errorf("edge %q to the data layer was dropped; declared edges must reach the graph", relation)
		}
	}
	for _, unresolved := range p.Unresolved {
		if strings.Contains(unresolved, "database") {
			t.Errorf("data layer reported as unmaterialized: %s", unresolved)
		}
	}
}

// The nodes must exist as subjects too, so a caller can ask what a database
// takes part in rather than only seeing it from the application side.
func TestDataLayerNodesAreQueryable(t *testing.T) {
	p := derivedbridge.FromIndex(dataLayerFixture(), "c")

	present := map[string]bool{}
	for _, node := range p.Graph.Nodes {
		present[node.Key.Subject.Kind] = true
	}
	for _, kind := range []string{"database", "database-contract", "database-contract-query"} {
		if !present[kind] {
			t.Errorf("%s node missing from the graph", kind)
		}
		if !derivedbridge.SupportsSubjectKind(kind) {
			t.Errorf("%s has a node but SupportsSubjectKind refuses it; trace would reject a subject the graph contains", kind)
		}
	}
}

// Every node kind in the graph must be one SupportsSubjectKind accepts. A node
// nobody can ask about is unusable. The other direction — a kind advertised but
// never projected — is covered by supports_kind_test.go, whose fixture carries
// every kind on purpose; this fixture is narrow by design.
func TestEveryProjectedNodeKindIsSupported(t *testing.T) {
	p := derivedbridge.FromIndex(dataLayerFixture(), "c")
	for _, node := range p.Graph.Nodes {
		if !derivedbridge.SupportsSubjectKind(node.Key.Subject.Kind) {
			t.Errorf("projected node kind %q is not accepted by SupportsSubjectKind", node.Key.Subject.Kind)
		}
	}
}
