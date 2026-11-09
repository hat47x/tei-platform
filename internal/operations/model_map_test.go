package operations

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

func TestMapWorldPlacesNativePatternInDataQueryGraphWithoutInventingEventFlow(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order",
				Attributes: []semantic.Attribute{{
					ID: "sales.order.id", TypeRef: "std.text",
				}},
				Relations: []semantic.Relation{{
					ID: "sales.order.customer", Target: "sales.customer", // NOT declared.
				}},
			}},
		}},
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID: "order_entry.browse", Kind: application.PatternBrowse,
				Data: "sales.order", Query: "order_entry.lookup",
			}},
		}},
		Queries: []query.EmbeddedSQL{{
			ID: "order_entry.lookup", Application: "order_entry",
			ResultColumns: []query.ResultColumnMapping{{
				Column: "id", Attribute: "sales.order.id",
			}},
		}},
	}
	catalog := CatalogWorld(world, sourceindex.Index{}, "", "world-fp", "source-fp")
	got := MapWorld(world, catalog)
	if got.Effect != "read-only" || got.Coverage != "partial-projection" ||
		got.SchemaVersion != "tei.operations.model-map/v0" ||
		got.Catalog.WorldFingerprint != "world-fp" || len(got.UnqualifiedRoles) == 0 {
		t.Fatalf("map invented complete model or lost catalog evidence: %+v", got)
	}
	has := func(kind, fromKind, fromID, toKind, toID, resolution string) bool {
		t.Helper()
		for _, l := range got.Links {
			if l.Relation == kind &&
				l.From == (sourceindex.Subject{Kind: fromKind, ID: fromID}) &&
				l.To == (sourceindex.Subject{Kind: toKind, ID: toID}) &&
				l.Resolution == resolution {
				return true
			}
		}
		return false
	}
	if !has("contains", "application", "order_entry", "application-pattern", "order_entry.browse", "resolved") ||
		!has("uses-data", "application-pattern", "order_entry.browse", "semantic-data", "sales.order", "resolved") ||
		!has("uses-query", "application-pattern", "order_entry.browse", "query", "order_entry.lookup", "resolved") ||
		!has("owned-by", "query", "order_entry.lookup", "application", "order_entry", "resolved") ||
		!has("maps-result-to", "query", "order_entry.lookup", "semantic-attribute", "sales.order.id", "resolved") {
		t.Fatalf("native Pattern cannot navigate to authored Data/Query: %+v", got.Links)
	}
	if !has("refers-to", "semantic-relation", "sales.order.customer", "semantic-data", "sales.customer", "unresolved") {
		t.Fatalf("undeclared Relation target was hidden or treated as accepted: %+v", got.Links)
	}
	if !has("has-type", "semantic-attribute", "sales.order.id", "semantic-type", "std.text", "unresolved") {
		t.Fatalf("undeclared primitive type was falsely accepted (no std prelude in this synthetic World)")
	}
	for _, l := range got.Links {
		if l.From.Kind == "business-event" || l.To.Kind == "lifecycle-transition" ||
			l.Relation == "authorized" || l.Relation == "committed" {
			t.Fatalf("event/authority/effect was invented from a Pattern: %+v", l)
		}
	}
	if !reflect.DeepEqual(got, MapWorld(world, catalog)) {
		t.Fatal("read-only projection is not deterministic")
	}
}

func TestMapWorldAmbiguousTargetIsNeverFirstWins(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{ID: "one", Data: []semantic.Data{{ID: "same.data"}}},
		{ID: "two", Data: []semantic.Data{{ID: "same.data"}}},
	}, Applications: []application.Application{{
		ID: "app", Patterns: []application.Pattern{{
			ID: "app.list", Kind: application.PatternBrowse, Data: "same.data",
		}},
	}}}
	got := MapWorld(world, CatalogWorld(world, sourceindex.Index{}, "", "w", "s"))
	for _, l := range got.Links {
		if l.Relation == "uses-data" {
			if l.Resolution != "ambiguous" {
				t.Fatalf("duplicate same-id Data silently resolved by first-wins: %+v", l)
			}
			return
		}
	}
	t.Fatal("Pattern->ambiguous Data link was omitted")
}

func TestMapWorkspaceUsesActualOrgSourceWithoutInventingTestOnlyPattern(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-headless-workspace")
	got, err := MapWorkspace(root)
	if err != nil {
		t.Fatalf("read-only native organization workspace Map: %v", err)
	}
	if got.Catalog.Workspace == nil || got.Catalog.ModelRevision == "" ||
		got.Catalog.WorldFingerprint == "" || got.Catalog.SourceIndexFingerprint == "" ||
		got.Coverage != "partial-projection" {
		t.Fatalf("Source/Scope qualification lost: %+v", got)
	}
	found := false
	for _, l := range got.Links {
		if l.From == (sourceindex.Subject{Kind: "semantic-relation", ID: "organization.case_record.handling_unit"}) &&
			l.To == (sourceindex.Subject{Kind: "semantic-data", ID: "organization.org_unit"}) &&
			l.Relation == "refers-to" && l.Resolution == "resolved" {
			found = true
		}
		if l.From.ID == "org_view.case_browse" || l.To.ID == "org_view.case_browse" {
			t.Fatalf("only #3500's copied test workspace defines case_browse; native Source does not: %+v", l)
		}
	}
	if !found {
		t.Fatalf("actual CaseRecord handlingUnit Relation missing from read-only map: %+v", got.Links)
	}
}
