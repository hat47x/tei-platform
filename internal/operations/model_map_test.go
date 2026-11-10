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


// A unique declaration makes a typed navigation link "resolved" even when
// no managed Source was indexed. The resolution value MUST NOT be interpreted
// as Source-current, meaning-accepted, or edit-authorized.
func TestMapWorldDeclaredLinkIsNotSourceQualified(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{
			{ID: "sales.order", Relations: []semantic.Relation{{ID: "sales.order.lines", Target: "sales.line"}}},
			{ID: "sales.line"},
		},
	}}}
	catalog := CatalogWorld(world, sourceindex.Index{}, "", "synthetic-world", "empty-index")
	mapped := MapWorld(world, catalog)
	wantRelation := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.lines"}
	wantTarget := sourceindex.Subject{Kind: "semantic-data", ID: "sales.line"}
	var seen int
	for _, link := range mapped.Links {
		if link.From == wantRelation && link.To == wantTarget && link.Relation == "refers-to" {
			seen++
			if link.Resolution != "resolved" {
				t.Fatalf("unique declared relation was not resolved: %+v", link)
			}
		}
	}
	if seen != 1 {
		t.Fatalf("expected exactly one unique declared relation link, got %d", seen)
	}
	for _, entry := range mapped.Catalog.Entries {
		if (entry.Subject == wantRelation || entry.Subject == wantTarget) &&
			entry.SourceStatus != "not-indexed" {
			t.Fatalf("no Source Index was provided, yet Source was claimed: %+v", entry)
		}
	}
	if mapped.Coverage != "partial-projection" {
		t.Fatalf("declared links alone cannot close the Source World: %+v", mapped)
	}
}

// A forged Source Index Reference can be structurally valid but does not
// supply an actual Canonical World declaration or a Relation edge.
func TestMapWorldDoesNotInventRelationFromForgedSourceReference(t *testing.T) {
	phantom := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.phantom"}
	idx := sourceindex.Index{
		References: []sourceindex.Reference{{
			Subject: phantom, ArtifactPath: "semantic/source.tei", Role: sourceindex.RoleDefinition,
		}},
	}
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{{ID: "sales.order"}},
	}}}
	mapped := MapWorld(world, CatalogWorld(world, idx, "", "world", "index"))
	for _, entry := range mapped.Catalog.Entries {
		if entry.Subject == phantom {
			t.Fatalf("forged Source Index Reference invented a canonical declaration: %+v", entry)
		}
	}
	for _, link := range mapped.Links {
		if link.From == phantom || link.To == phantom {
			t.Fatalf("forged Source Index Reference invented a typed Model Map edge: %+v", link)
		}
	}
	// This is a purely in-memory representation test: it neither tests a
	// malicious filesystem nor proves that Source references are trustworthy.
}

// Two business relations can share a visible label, from Data and target Data.
// Their domain IDs must survive a typed Model Map; collapsing both into one
// from->to "reference" edge would make diagram selection irreversibly lossy.
func TestMapWorldRetainsParallelDomainRelationIDsWithoutGrantingSourceEdits(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{
			{ID: "sales.order", Relations: []semantic.Relation{
				{ID: "sales.order.line.primary", Symbol: "lines", Target: "sales.line"},
				{ID: "sales.order.line.secondary", Symbol: "lines", Target: "sales.line"},
			}},
			{ID: "sales.line"},
		},
	}}}
	mapped := MapWorld(world, CatalogWorld(world, sourceindex.Index{}, "", "world", "source"))
	parent := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	target := sourceindex.Subject{Kind: "semantic-data", ID: "sales.line"}
	for _, id := range []string{"sales.order.line.primary", "sales.order.line.secondary"} {
		relation := sourceindex.Subject{Kind: "semantic-relation", ID: id}
		for _, expected := range []ModelMapLink{
			{From: parent, To: relation, Relation: "contains", Resolution: "resolved"},
			{From: relation, To: target, Relation: "refers-to", Resolution: "resolved"},
		} {
			count := 0
			for _, actual := range mapped.Links {
				if actual == expected {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("separate relation identity collapsed or duplicated: %+v count=%d", expected, count)
			}
		}
		indexed := false
		for _, entry := range mapped.Catalog.Entries {
			if entry.Subject == relation && entry.SourceStatus == "indexed" {
				indexed = true
			}
		}
		if indexed {
			t.Fatalf("synthetic declaration %q has no actual indexed Source and cannot authorize patch", id)
		}
	}
	if mapped.Coverage != "partial-projection" {
		t.Fatalf("two parallel links do not imply complete world closure")
	}
}

// Reusing one canonical relation ID in distinct parent Data is ambiguous,
// even when the relation label, type and target happen to be identical.
func TestMapWorldDuplicateRelationIDDoesNotFirstWin(t *testing.T) {
	const duplicateID = "sales.shared.line"
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{
			{ID: "sales.order", Relations: []semantic.Relation{{
				ID: duplicateID, Symbol: "line", Target: "sales.line",
			}}},
			{ID: "sales.quote", Relations: []semantic.Relation{{
				ID: duplicateID, Symbol: "line", Target: "sales.line",
			}}},
			{ID: "sales.line"},
		},
	}}}
	mapped := MapWorld(world, CatalogWorld(world, sourceindex.Index{}, "", "world", "source"))
	duplicate := sourceindex.Subject{Kind: "semantic-relation", ID: duplicateID}
	seenEntries := 0
	for _, entry := range mapped.Catalog.Entries {
		if entry.Subject == duplicate {
			seenEntries++
			if !entry.Ambiguous {
				t.Fatalf("duplicate canonical Relation was treated as unique: %+v", entry)
			}
		}
	}
	if seenEntries != 2 {
		t.Fatalf("duplicate Relation declarations were not retained for review: %d", seenEntries)
	}
	contains := 0
	refers := 0
	for _, link := range mapped.Links {
		if link.To == duplicate && link.Relation == "contains" {
			contains++
			if link.Resolution != "ambiguous" {
				t.Fatalf("duplicate Relation parent incorrectly resolved: %+v", link)
			}
		}
		if link.From == duplicate && link.Relation == "refers-to" {
			refers++
			if link.Resolution != "ambiguous" {
				t.Fatalf("duplicate Relation target incorrectly resolved: %+v", link)
			}
		}
	}
	if contains != 2 || refers != 1 {
		t.Fatalf("expected two ambiguous parent links and one deduplicated ambiguous target link: contains=%d refers=%d", contains, refers)
	}
}

// Typed navigation topology is derived from World declarations, whereas
// Source provenance has its own revision. A source move or byte change
// must not masquerade as a semantic graph edit (or preserve edit authority).
func TestMapWorldSourceOnlyRevisionMustNotBeConfusedWithSemanticTopology(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{
			{ID: "sales.order", Relations: []semantic.Relation{{
				ID: "sales.order.lines", Target: "sales.line",
			}}},
			{ID: "sales.line"},
		},
	}}}
	relation := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.lines"}
	mapFor := func(path string, bytes []byte) ModelMap {
		t.Helper()
		artifact, err := sourceindex.NewArtifact(path, bytes)
		if err != nil {
			t.Fatal(err)
		}
		idx := sourceindex.Index{
			Artifacts: []sourceindex.Artifact{artifact},
			References: []sourceindex.Reference{{
				Subject: relation, ArtifactPath: artifact.Path, Role: sourceindex.RoleDefinition,
			}},
		}
		worldFP, sourceFP, err := workspaceProjectionFingerprints(world, idx)
		if err != nil {
			t.Fatal(err)
		}
		return MapWorld(world, CatalogWorld(world, idx, "", worldFP, sourceFP))
	}
	a := mapFor("semantic/sales.tei", []byte("revision-a"))
	moved := mapFor("semantic/moved-sales.tei", []byte("revision-a"))
	changed := mapFor("semantic/sales.tei", []byte("revision-b"))
	backToA := mapFor("semantic/sales.tei", []byte("revision-a"))
	for name, other := range map[string]ModelMap{
		"source-move": moved,
		"source-byte-change": changed,
	} {
		if !reflect.DeepEqual(a.Links, other.Links) {
			t.Fatalf("%s changed semantic topology without changing World: %+v", name, other.Links)
		}
		if a.Catalog.WorldFingerprint != other.Catalog.WorldFingerprint {
			t.Fatalf("%s unexpectedly changed World fingerprint", name)
		}
		if a.Catalog.SourceIndexFingerprint == other.Catalog.SourceIndexFingerprint ||
			a.Catalog.ModelRevision == other.Catalog.ModelRevision {
			t.Fatalf("%s was hidden by the semantic graph fingerprint", name)
		}
	}
	for _, entry := range moved.Catalog.Entries {
		if entry.Subject == relation && entry.SourceStatus != "indexed" {
			t.Fatalf("source move was incorrectly treated as no Source: %+v", entry)
		}
	}
	if !reflect.DeepEqual(a, backToA) {
		t.Fatal("same current Source/World after A-to-B-to-A should produce the same descriptive map")
	}
	// Identical current fingerprints after A->B->A are NOT evidence that
	// B never existed. This projection is read-only and does not implement
	// source-history retention, acceptance or edit authorization.
}
