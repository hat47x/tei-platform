package index

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
)

// dataLayerWorld builds a World with one database, one contract and one
// contract query.
func dataLayerWorld() ir.World {
	return ir.World{Databases: []database.Database{{
		ID:   core.MustCanonicalID("sales_core"),
		Kind: "postgres",
		Contracts: []database.Contract{{
			ID: core.MustCanonicalID("sales_core.order"),
			Queries: []database.QueryContract{{
				Name: "find_by_id",
			}},
		}},
	}}}
}

// TestDataLayerSubjectsAreDeclared pins #4068. Every declared edge of the data
// layer names a database, a database-contract or a database-contract-query, and
// the projection can only keep an edge whose endpoints are nodes. The index kept
// no such declaration, so `targets-database`, `requires-contract`,
// `contract-query`, `reads` and `writes` were all dropped.
func TestDataLayerSubjectsAreDeclared(t *testing.T) {
	derived := Build(dataLayerWorld())

	want := map[string]string{
		"database":                "sales_core",
		"database-contract":       "sales_core.order",
		"database-contract-query": "sales_core.order/find_by_id",
	}
	got := map[string]string{}
	for _, e := range derived.DatabaseElements {
		got[e.Kind] = string(e.ID)
	}
	for kind, id := range want {
		if got[kind] != id {
			t.Fatalf("%s = %q, want %q (all: %+v)", kind, got[kind], id, derived.DatabaseElements)
		}
	}
}

// A contract query with no name declares nothing addressable and must not
// produce a node id with a trailing separator.
func TestContractQueryWithoutNameIsSkipped(t *testing.T) {
	w := dataLayerWorld()
	w.Databases[0].Contracts[0].Queries = append(w.Databases[0].Contracts[0].Queries,
		database.QueryContract{Name: ""})
	derived := Build(w)
	queries := 0
	for _, e := range derived.DatabaseElements {
		if e.Kind != "database-contract-query" {
			continue
		}
		queries++
		if e.ID == "sales_core.order/" {
			t.Fatalf("empty query name produced %q", e.ID)
		}
	}
	if queries != 1 {
		t.Fatalf("contract queries = %d, want the one named query", queries)
	}
}

// The declaration list must be ordered deterministically, like the other
// index lists, so the graph fingerprint does not depend on map iteration.
func TestDatabaseElementsAreSorted(t *testing.T) {
	w := dataLayerWorld()
	second := database.Database{
		ID:   core.MustCanonicalID("aaa_core"),
		Kind: "postgres",
		Contracts: []database.Contract{{
			ID: core.MustCanonicalID("aaa_core.thing"),
		}},
	}
	w.Databases = append(w.Databases, second)
	derived := Build(w)
	for i := 1; i < len(derived.DatabaseElements); i++ {
		prev, cur := derived.DatabaseElements[i-1], derived.DatabaseElements[i]
		if prev.Kind > cur.Kind || (prev.Kind == cur.Kind && prev.ID > cur.ID) {
			t.Fatalf("not sorted at %d: %+v then %+v", i, prev, cur)
		}
	}
}
