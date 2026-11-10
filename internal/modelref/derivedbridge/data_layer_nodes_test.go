package derivedbridge

import (
	"testing"

	"tei/internal/core"
	"tei/internal/index"
)

// dataLayerDerived builds an index with one application, one database, the
// contract it publishes and one contract query, plus the declared edges that
// name them.
func dataLayerDerived() index.Derived {
	d := index.Derived{}
	d.ApplicationPatterns = append(d.ApplicationPatterns,
		index.ApplicationPatternIndexEntry{Application: core.CanonicalID("order_entry"), Pattern: "pattern"})
	d.DatabaseElements = []index.DatabaseElementIndexEntry{
		{Kind: "database", ID: core.CanonicalID("sales_core")},
		{Kind: "database-contract", ID: core.CanonicalID("sales_core.order")},
		{Kind: "database-contract-query", ID: core.CanonicalID("sales_core.order/find_by_id")},
	}
	d.DeclaredDependencies = append(d.DeclaredDependencies,
		index.DeclaredDependency{FromKind: "application", From: core.CanonicalID("order_entry"),
			Relation: "targets-database", ToKind: "database", To: core.CanonicalID("sales_core")},
		index.DeclaredDependency{FromKind: "application", From: core.CanonicalID("order_entry"),
			Relation: "requires-contract", ToKind: "database-contract", To: core.CanonicalID("sales_core.order")},
		index.DeclaredDependency{FromKind: "application-pattern", From: core.CanonicalID("pattern"),
			Relation: "contract-query", ToKind: "database-contract-query", To: core.CanonicalID("sales_core.order/find_by_id")},
	)
	return d
}

// TestDataLayerNodesLetDeclaredEdgesSurvive pins #4068. The projection keeps an
// edge only when both endpoints are nodes. The declared edges name a database, a
// contract and a contract query, so without those nodes every data-layer edge
// was dropped and the projection reported it as unmaterialized.
func TestDataLayerNodesLetDeclaredEdgesSurvive(t *testing.T) {
	p := FromIndex(dataLayerDerived(), "c")

	want := map[string]bool{"targets-database": false, "requires-contract": false, "contract-query": false}
	for _, edge := range p.Graph.Edges {
		if _, tracked := want[edge.Relation]; tracked {
			want[edge.Relation] = true
		}
	}
	for relation, found := range want {
		if !found {
			t.Fatalf("the %s edge was dropped; edges are %v", relation, p.Graph.Edges)
		}
	}
	if len(p.Unresolved) != 0 {
		t.Fatalf("unexpected unresolved entries: %v", p.Unresolved)
	}
}

// The data-layer nodes exist as edge endpoints but must not widen the advertised
// subject-kind grammar: that list is the public query grammar, and widening it is
// a contract change (option B of #4068).
func TestDataLayerNodesAreNotAdvertisedSubjectKinds(t *testing.T) {
	for _, kind := range []string{"database", "database-contract", "database-contract-query"} {
		if SupportsSubjectKind(kind) {
			t.Fatalf("%s became an advertised subject kind", kind)
		}
	}
}

// A declaration without a matching DatabaseElements entry must still be reported
// rather than silently dropped: that is the state #4068 was filed about.
func TestMissingDataLayerNodeIsStillReported(t *testing.T) {
	d := dataLayerDerived()
	d.DatabaseElements = nil
	p := FromIndex(d, "c")
	found := false
	for _, entry := range p.Unresolved {
		if entry == "unmaterialized-to:database:sales_core" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the missing database node was not reported; unresolved = %v", p.Unresolved)
	}
}
