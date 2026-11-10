package index

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// columnWorld builds a World with one Data, one Attribute and one persistence
// mapping that stores the Attribute in a column.
func columnWorld(column string) ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: core.MustCanonicalID("sales"),
			Data: []semantic.Data{{
				ID: core.MustCanonicalID("sales.order"), Symbol: "order",
				Attributes: []semantic.Attribute{{
					ID: core.MustCanonicalID("sales.order.total"), Symbol: "total",
					TypeRef: core.MustCanonicalID("std.decimal"),
				}},
			}},
		}},
		Persistence: []persistence.Mapping{{
			ID:           core.MustCanonicalID("sales_core.order"),
			Database:     core.MustCanonicalID("sales_core"),
			SemanticData: core.MustCanonicalID("sales.order"),
			Table:        "order",
			Columns: []persistence.ColumnMapping{{
				Attribute: core.MustCanonicalID("sales.order.total"),
				Column:    column,
			}},
		}},
	}
}

// TestColumnMappingBecomesAnEdge pins #4001. `column <attribute> -> <name>`
// states which column stores an attribute. The index kept the mapping but not
// the column correspondence, and the projection never made it an edge, so
// `trace -direction incoming` on an attribute reported zero even though the
// Workspace declared where it is stored.
func TestColumnMappingBecomesAnEdge(t *testing.T) {
	derived := Build(columnWorld("total_amount"))

	var found *DeclaredDependency
	for i := range derived.DeclaredDependencies {
		dep := derived.DeclaredDependencies[i]
		if dep.Relation == "maps-column" {
			found = &derived.DeclaredDependencies[i]
			_ = dep
			break
		}
	}
	if found == nil {
		t.Fatalf("no maps-column edge; declared edges: %+v", derived.DeclaredDependencies)
	}
	if found.FromKind != "semantic-data" || string(found.From) != "sales.order" {
		t.Fatalf("maps-column starts at %s:%s, want semantic-data:sales.order", found.FromKind, found.From)
	}
	if found.ToKind != "semantic-attribute" || string(found.To) != "sales.order.total" {
		t.Fatalf("maps-column points at %s:%s, want semantic-attribute:sales.order.total", found.ToKind, found.To)
	}
	// The column name is carried as the alias, so the edge says which column.
	if string(found.Alias) != "total_amount" {
		t.Fatalf("alias = %q, want the column name total_amount", found.Alias)
	}
}

// The correspondence must also be readable as data, not only as an edge: a
// caller that asks the index directly gets the same pair.
func TestColumnMappingIsRetainedInTheIndex(t *testing.T) {
	derived := Build(columnWorld("total_amount"))
	if len(derived.PersistenceMappings) != 1 {
		t.Fatalf("persistenceMappings = %d, want 1", len(derived.PersistenceMappings))
	}
	entry := derived.PersistenceMappings[0]
	if len(entry.Columns) != 1 {
		t.Fatalf("columns = %v, want one pair", entry.Columns)
	}
	if string(entry.Columns[0].Attribute) != "sales.order.total" || entry.Columns[0].Column != "total_amount" {
		t.Fatalf("column pair = %+v", entry.Columns[0])
	}
}

// A mapping with no declared column correspondence must not invent an edge.
func TestColumnMappingWithoutColumnsProducesNoEdge(t *testing.T) {
	w := columnWorld("total_amount")
	w.Persistence[0].Columns = nil
	derived := Build(w)
	for _, dep := range derived.DeclaredDependencies {
		if dep.Relation == "maps-column" {
			t.Fatalf("a column-less mapping produced %+v", dep)
		}
	}
	if len(derived.PersistenceMappings) != 1 {
		t.Fatalf("the mapping itself must still be indexed")
	}
}

// A column pair with an empty attribute or name is not a correspondence and
// must be skipped rather than become an edge the projection would report as
// unmaterialized.
func TestColumnMappingSkipsIncompletePairs(t *testing.T) {
	w := columnWorld("total_amount")
	w.Persistence[0].Columns = append(w.Persistence[0].Columns,
		persistence.ColumnMapping{Attribute: "", Column: "orphan"},
		persistence.ColumnMapping{Attribute: core.MustCanonicalID("sales.order.total"), Column: ""},
	)
	derived := Build(w)
	edges := 0
	for _, dep := range derived.DeclaredDependencies {
		if dep.Relation == "maps-column" {
			edges++
		}
	}
	if edges != 1 {
		t.Fatalf("maps-column edges = %d, want only the complete pair", edges)
	}
	if len(derived.PersistenceMappings[0].Columns) != 1 {
		t.Fatalf("an incomplete pair was retained: %v", derived.PersistenceMappings[0].Columns)
	}
}
