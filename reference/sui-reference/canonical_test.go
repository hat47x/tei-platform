package suireference_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func loadWorld(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load("workspace")
	if err != nil {
		t.Fatalf("sui-reference workspace: %v", err)
	}
	return loaded.World
}

func findData(t *testing.T, world ir.World, id core.CanonicalID) semantic.Data {
	t.Helper()
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID == id {
				return data
			}
		}
	}
	t.Fatalf("data %s is not declared", id)
	return semantic.Data{}
}

func relationByID(data semantic.Data) map[core.CanonicalID]semantic.Relation {
	byID := map[core.CanonicalID]semantic.Relation{}
	for _, relation := range data.Relations {
		byID[relation.ID] = relation
	}
	return byID
}

// TestContainmentAndAffiliationAreSeparateRelations pins invariant I-02.
// Visual containment and cross-cutting affiliation are distinct declarations,
// so a consumer cannot fold them into one "belongsTo".
func TestContainmentAndAffiliationAreSeparateRelations(t *testing.T) {
	card := findData(t, loadWorld(t), "sui.card")
	relations := relationByID(card)

	containment, ok := relations["sui.card.contained_in"]
	if !ok {
		t.Fatal("sui.card.contained_in is not declared")
	}
	affiliation, ok := relations["sui.card.affiliated_with"]
	if !ok {
		t.Fatal("sui.card.affiliated_with is not declared")
	}
	if containment.Cardinality != semantic.CardinalityOne {
		t.Fatalf("containment cardinality = %q, want one", containment.Cardinality)
	}
	if affiliation.Cardinality != semantic.CardinalityMany {
		t.Fatalf("affiliation cardinality = %q, want many", affiliation.Cardinality)
	}
	if containment.Target != "sui.island" || affiliation.Target != "sui.island" {
		t.Fatalf("relation targets = %q / %q, want sui.island", containment.Target, affiliation.Target)
	}
	if containment.ID == affiliation.ID {
		t.Fatal("containment and affiliation share one identity")
	}
}

// TestNoTreeHierarchyIsDeclared pins invariant I-08. Island parent is a
// separate single-valued link; no Tree/Forest data is declared, and the
// containment relation is not named as a Tree shape.
func TestNoTreeHierarchyIsDeclared(t *testing.T) {
	world := loadWorld(t)

	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			switch data.ID {
			case "sui.tree", "sui.forest", "sui.branch", "sui.tree_node":
				t.Fatalf("Tree hierarchy %s must not be declared", data.ID)
			}
			if data.ID == "sui.card" {
				continue
			}
			for _, relation := range data.Relations {
				switch relation.Symbol {
				case "child", "children", "parentNode", "treeParent":
					t.Fatalf("%s.%s declares a Tree shape", data.ID, relation.Symbol)
				}
			}
		}
	}

	island := findData(t, world, "sui.island")
	parent, ok := relationByID(island)["sui.island.parent"]
	if !ok {
		t.Fatal("sui.island.parent is not declared")
	}
	if parent.Cardinality != semantic.CardinalityOne {
		t.Fatalf("parent cardinality = %q, want one", parent.Cardinality)
	}
}

// TestCardBodyAndIslandTitleAreNotDeclared pins the read-only framing:
// content is not an attribute of this reference, so no consumer can treat a
// readable Card as an accepted meaning that carries body text.
func TestCardBodyAndIslandTitleAreNotDeclared(t *testing.T) {
	world := loadWorld(t)
	forbidden := []string{"text", "title", "body", "content", "summary", "narrative"}
	for _, id := range []core.CanonicalID{"sui.card", "sui.island", "sui.document"} {
		data := findData(t, world, id)
		for _, attribute := range data.Attributes {
			if attribute.Required {
				t.Fatalf("%s.%s is required", id, attribute.Symbol)
			}
			for _, name := range forbidden {
				if attribute.Symbol == name {
					t.Fatalf("%s declares content attribute %q", id, name)
				}
			}
		}
	}
}

// TestReferenceTypesResolveToStandardPrimitives pins the type vocabulary:
// only std.* primitives and sui.* aliases are used, so the reference does not
// invent a second meaning registry.
func TestReferenceTypesResolveToStandardPrimitives(t *testing.T) {
	world := loadWorld(t)

	known := map[core.CanonicalID]bool{}
	for _, namespace := range world.Semantics {
		for _, typ := range namespace.Types {
			known[typ.ID] = true
		}
	}

	checked := 0
	for _, namespace := range world.Semantics {
		if namespace.ID != "sui" {
			continue
		}
		for _, data := range namespace.Data {
			for _, attribute := range data.Attributes {
				if !known[attribute.TypeRef] {
					t.Fatalf("%s.%s resolves to unknown type %q", data.ID, attribute.Symbol, attribute.TypeRef)
				}
				checked++
			}
		}
	}
	if checked == 0 {
		t.Fatal("no sui attributes were checked")
	}
}
