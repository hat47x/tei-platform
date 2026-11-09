package operations

import (
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

func locationWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{
				{
					ID: "sales.order", Symbol: "order",
					Attributes: []semantic.Attribute{
						{ID: "sales.order.id", Symbol: "id", TypeRef: "sales.code"},
						{ID: "sales.order.note", Symbol: "note", TypeRef: "sales.missing_type"},
					},
					Relations: []semantic.Relation{
						{ID: "sales.order.customer", Symbol: "customer", Target: "sales.customer"},
						{ID: "sales.order.carrier", Symbol: "carrier", Target: "sales.carrier"},
					},
				},
				{ID: "sales.customer", Symbol: "customer"},
			},
			Types:        []semantic.Type{{ID: "sales.code", Symbol: "code"}},
			Capabilities: []semantic.Capability{{ID: "sales.order.create", Symbol: "create"}},
		}},
		Applications: []application.Application{{ID: "order_entry", Patterns: []application.Pattern{
			{ID: "order_entry.list", Kind: application.PatternBrowse, Data: "sales.order"},
			{ID: "order_entry.customers", Kind: application.PatternBrowse, Data: "sales.customer"},
		}}},
	}
}

func locate(t *testing.T, world ir.World, subject sourceindex.Subject, limit int) ModelLocation {
	t.Helper()
	catalog := CatalogWorld(world, sourceindex.Index{}, "", "world", "source")
	got, failure := LocateInCatalog(catalog, world, subject, limit)
	if failure != nil {
		t.Fatalf("locate %v: %s %s", subject, failure.Code, failure.Message)
	}
	return got
}

// G1: the path from the top is derived from catalog parents, and the answer
// carries the catalog's fingerprints so it can be compared with a diagram.
func TestLocateReportsPathFromTheTopAndTheModelVersion(t *testing.T) {
	got := locate(t, locationWorld(), sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}, 50)
	if got.Effect != "read-only" || got.WorldFingerprint != "world" || got.SourceIndexFingerprint != "source" || got.ModelRevision == "" {
		t.Fatalf("header: %+v", got)
	}
	want := []sourceindex.Subject{{Kind: "semantic", ID: "sales"}, {Kind: "semantic-data", ID: "sales.order"}}
	if len(got.Path) != len(want) {
		t.Fatalf("path=%+v", got.Path)
	}
	for i := range want {
		if got.Path[i].Subject != want[i] {
			t.Fatalf("path[%d]=%v want %v", i, got.Path[i].Subject, want[i])
		}
	}
}

// G3: inbound references tell which pattern uses a data element; the role
// keeps a relation target apart from a pattern's data.
func TestLocateListsInboundAndOutboundDeclaredReferences(t *testing.T) {
	data := locate(t, locationWorld(), sourceindex.Subject{Kind: "semantic-data", ID: "sales.customer"}, 50)
	roles := map[string]sourceindex.Subject{}
	for _, link := range data.Incoming {
		roles[link.Role] = link.From
		if !link.Resolved {
			t.Fatalf("incoming link to an existing subject must be resolved: %+v", link)
		}
	}
	if roles[LinkRoleRelationTarget] != (sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}) ||
		roles[LinkRolePatternData] != (sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.customers"}) {
		t.Fatalf("incoming=%+v", data.Incoming)
	}

	pattern := locate(t, locationWorld(), sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.list"}, 50)
	if len(pattern.Outgoing) != 1 || pattern.Outgoing[0].Role != LinkRolePatternData ||
		pattern.Outgoing[0].To != (sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}) {
		t.Fatalf("pattern outgoing=%+v", pattern.Outgoing)
	}
}

// A reference to an ID that does not exist is reported unresolved, not
// dropped and not attached to something with the same name.
func TestLocateKeepsUnresolvedReferences(t *testing.T) {
	rel := locate(t, locationWorld(), sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.carrier"}, 50)
	if len(rel.Outgoing) != 1 || rel.Outgoing[0].Resolved ||
		rel.Outgoing[0].To != (sourceindex.Subject{Kind: "semantic-data", ID: "sales.carrier"}) {
		t.Fatalf("unresolved relation target lost: %+v", rel.Outgoing)
	}
	attr := locate(t, locationWorld(), sourceindex.Subject{Kind: "semantic-attribute", ID: "sales.order.note"}, 50)
	if len(attr.Outgoing) != 1 || attr.Outgoing[0].Resolved || attr.Outgoing[0].Role != LinkRoleAttributeType {
		t.Fatalf("unresolved attribute type lost: %+v", attr.Outgoing)
	}
}

// Lists are bounded; Total, Omitted and KindCounts describe the full set so a
// cut answer is never read as the whole.
func TestLocateBoundsListsAndReportsWhatWasOmitted(t *testing.T) {
	got := locate(t, locationWorld(), sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}, 1)
	if got.Children.Total != 4 || len(got.Children.Entries) != 1 || got.Children.Omitted != 3 {
		t.Fatalf("children=%+v", got.Children)
	}
	counts := map[string]int{}
	for _, kc := range got.Children.KindCounts {
		counts[kc.Kind] = kc.Count
	}
	if counts["semantic-attribute"] != 2 || counts["semantic-relation"] != 2 {
		t.Fatalf("kind counts=%+v", got.Children.KindCounts)
	}
	// Siblings share the parent namespace: the other data, the type and the
	// capability.
	if got.Siblings.Total != 3 {
		t.Fatalf("siblings=%+v", got.Siblings)
	}
}

// Top-level siblings are the other top-level elements of the same kind, not
// every root in the World.
func TestLocateTopLevelSiblingsShareTheKind(t *testing.T) {
	got := locate(t, locationWorld(), sourceindex.Subject{Kind: "application", ID: "order_entry"}, 50)
	if len(got.Path) != 0 || got.Siblings.Total != 0 || got.Children.Total != 2 {
		t.Fatalf("application location: path=%+v siblings=%+v children=%+v", got.Path, got.Siblings, got.Children)
	}
}

// The families this answer does not derive are named, so an empty list is not
// read as "no event or implementation touches this".
func TestLocateNamesTheFamiliesItDoesNotDerive(t *testing.T) {
	got := locate(t, locationWorld(), sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}, 50)
	need := map[string]bool{"business-event-flow": false, "lifecycle-state-transition": false, "implementation-binding": false}
	for _, family := range got.Coverage.NotDerived {
		if _, ok := need[family]; ok {
			need[family] = true
		}
	}
	for family, found := range need {
		if !found {
			t.Fatalf("notDerived omits %s: %+v", family, got.Coverage)
		}
	}
}

func TestLocateRejectsUnknownAmbiguousAndBadLimit(t *testing.T) {
	world := locationWorld()
	catalog := CatalogWorld(world, sourceindex.Index{}, "", "w", "s")
	if _, failure := LocateInCatalog(catalog, world, sourceindex.Subject{Kind: "semantic-data", ID: "sales.nothing"}, 50); failure == nil || failure.Code != LocationCodeSubjectNotFound {
		t.Fatalf("unknown subject: %+v", failure)
	}
	// Same ID under another kind is a different subject, not a match.
	if _, failure := LocateInCatalog(catalog, world, sourceindex.Subject{Kind: "semantic-type", ID: "sales.order"}, 50); failure == nil || failure.Code != LocationCodeSubjectNotFound {
		t.Fatalf("kind mismatch must not match: %+v", failure)
	}
	if _, failure := LocateInCatalog(catalog, world, sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}, 0); failure == nil || failure.Code != LocationCodeInvalidLimit {
		t.Fatalf("limit 0: %+v", failure)
	}
	if _, failure := LocateInCatalog(catalog, world, sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}, LocationMaxLimit+1); failure == nil || failure.Code != LocationCodeInvalidLimit {
		t.Fatalf("limit over max: %+v", failure)
	}

	dup := locationWorld()
	dup.Semantics[0].Data = append(dup.Semantics[0].Data, semantic.Data{ID: "sales.customer", Symbol: "customer2"})
	dupCatalog := CatalogWorld(dup, sourceindex.Index{}, "", "w", "s")
	if _, failure := LocateInCatalog(dupCatalog, dup, sourceindex.Subject{Kind: "semantic-data", ID: "sales.customer"}, 50); failure == nil || failure.Code != LocationCodeSubjectAmbiguous {
		t.Fatalf("duplicate declaration must be ambiguous: %+v", failure)
	}
}
