package changeanalysis

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestTypeImpactUsesExactIdentityAndKeepsPreviousReferences(t *testing.T) {
	w := ir.World{
		Semantics: []semantic.Namespace{{ID: "sales", Types: []semantic.Type{{ID: "sales.derived", Underlying: "sales.order_id"}}, Data: []semantic.Data{{ID: "sales.order", Attributes: []semantic.Attribute{{ID: "sales.order.id", TypeRef: "sales.order_id"}, {ID: "sales.order.other", TypeRef: "sales.order_id_other"}}}}}},
		Databases: []database.Database{{ID: "db", Contracts: []database.Contract{{ID: "db.orders", Queries: []database.QueryContract{{Name: "find", Inputs: []database.QueryInput{{Name: "id", Type: "sales.order_id"}, {Name: "other", Type: "sales.order_id_other"}}}}}}}},
	}
	for _, kind := range []string{"changed", "removed"} {
		t.Run(kind, func(t *testing.T) {
			edges, unresolved := ImpactWithBefore(w, ir.World{}, []SubjectChange{{Kind: "type", ID: "sales.order_id", Change: kind}})
			wanted := map[string]bool{"type:sales.derived": false, "attribute:sales.order.id": false, "contract-query:db.orders/find": false}
			for _, edge := range edges {
				if edge.From != "type:sales.order_id" {
					t.Fatalf("wrong source: %+v", edge)
				}
				if _, ok := wanted[edge.To]; !ok {
					t.Fatalf("unrelated similar ID was included: %+v", edge)
				}
				wanted[edge.To] = true
			}
			for target, found := range wanted {
				if !found {
					t.Fatalf("missing historical target %s: %+v", target, edges)
				}
			}
			incomplete := false
			for _, issue := range unresolved {
				incomplete = incomplete || issue.Code == "SEMANTIC_TYPE_IMPACT_INCOMPLETE"
			}
			if !incomplete {
				t.Fatalf("coverage must stay unknown: %+v", unresolved)
			}
		})
	}
	if got := typeReferenceEdges(w, core.CanonicalID("sales.absent")); len(got) != 0 {
		t.Fatalf("unrelated type produced edges: %+v", got)
	}
	edges, unresolved := Impact(ir.World{}, []SubjectChange{{Kind: "type", ID: "sales.order_id", Change: "removed"}})
	if len(edges) != 0 {
		t.Fatalf("after-only removal invented a historical edge: %+v", edges)
	}
	found := false
	for _, issue := range unresolved {
		found = found || issue.Code == "SEMANTIC_TYPE_IMPACT_INCOMPLETE"
	}
	if !found {
		t.Fatal("after-only removal hid unknown impact")
	}
}
