package derivedbridge_test

import (
	"testing"

	"tei/internal/index"
	"tei/internal/modelref/derivedbridge"
)

// SupportsSubjectKind must describe exactly what FromIndex materializes. A
// kind claimed there but never added would make `trace` accept a subject the
// graph cannot contain; a kind projected but not claimed would make `trace`
// refuse a subject it could answer (#3180).
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
		// The data layer has no entry list of its own; it is named only as an
		// edge endpoint, so the fixture declares edges to make the projection
		// materialize the nodes (#4068).
		DeclaredDependencies: []index.DeclaredDependency{
			{FromKind: "application", From: "order_entry", Relation: "targets-database", ToKind: "database", To: "sales_core"},
			{FromKind: "application", From: "order_entry", Relation: "requires-contract", ToKind: "database-contract", To: "sales_core.order"},
			{FromKind: "application", From: "order_entry", Relation: "contract-query", ToKind: "database-contract-query", To: "sales_core.order/find"},
		},
	}
	projected := map[string]bool{}
	for _, node := range derivedbridge.FromIndex(d, "sales-scope").Graph.Nodes {
		projected[node.Key.Subject.Kind] = true
	}
	for _, kind := range []string{
		"semantic", "semantic-type", "semantic-data", "semantic-attribute",
		"semantic-relation", "semantic-validation", "semantic-capability",
		"application", "application-pattern", "query",
		"database", "database-contract", "database-contract-query",
	} {
		if !derivedbridge.SupportsSubjectKind(kind) {
			t.Fatalf("%s is projected but SupportsSubjectKind says no", kind)
		}
		if !projected[kind] {
			t.Fatalf("%s is claimed but the fixture graph does not carry it", kind)
		}
	}
	// inspect reads these straight from the Canonical World, but the projection
	// has no node for them, so trace must refuse the kind instead of the ID.
	for _, kind := range []string{"semantic-candidate-key", "semantic-functional-dependency", "unknown-kind"} {
		if derivedbridge.SupportsSubjectKind(kind) || projected[kind] {
			t.Fatalf("%s: unprojected kind was accepted", kind)
		}
	}
}
