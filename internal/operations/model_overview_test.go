package operations

import (
	"testing"

	"tei/internal/application"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func subjectPtr(kind, id string) *sourceindex.Subject {
	s := sourceindex.Subject{Kind: kind, ID: id}
	return &s
}

func overviewFor(t *testing.T, diagnostics []validation.Diagnostic, findings []workspace.WorkspaceSourceRootFinding, limit int) ModelOverview {
	t.Helper()
	world := locationWorld()
	idx := sourceindex.Index{References: []sourceindex.Reference{
		{Subject: sourceindex.Subject{Kind: "semantic", ID: "sales"}, ArtifactPath: "semantic/sales.tei", Role: sourceindex.RoleDefinition},
		{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}, ArtifactPath: "semantic/sales.tei", Role: sourceindex.RoleDefinition},
	}}
	catalog := CatalogWorld(world, idx, "", "world", "source")
	return OverviewFromRead(catalog, world, diagnostics, findings, limit)
}

func sectionFor(t *testing.T, o ModelOverview, kind, id string) ModelOverviewSection {
	t.Helper()
	for _, s := range o.Sections {
		if s.Entry.Subject == (sourceindex.Subject{Kind: kind, ID: id}) {
			return s
		}
	}
	t.Fatalf("section %s %s missing: %+v", kind, id, o.Sections)
	return ModelOverviewSection{}
}

// G5: a problem deep inside a section is counted on that section, so the
// overview shows where to look without opening the detail.
func TestOverviewRollsProblemsUpToTheirSection(t *testing.T) {
	diagnostics := []validation.Diagnostic{
		{Code: "X", Severity: validation.Error, Subject: subjectPtr("semantic-attribute", "sales.order.note")},
		{Code: "Y", Severity: validation.Warning, Subject: subjectPtr("application-pattern", "order_entry.list")},
	}
	o := overviewFor(t, diagnostics, nil, 50)
	sales := sectionFor(t, o, "semantic", "sales")
	if sales.Attention.ValidationErrors != 1 || sales.Attention.ValidationWarnings != 0 {
		t.Fatalf("sales validation roll-up: %+v", sales.Attention)
	}
	// sales.order.carrier -> sales.carrier and sales.order.note -> sales.missing_type
	// are unresolved; both sit under the sales namespace.
	if sales.Attention.UnresolvedReferences != 2 {
		t.Fatalf("sales unresolved: %+v", sales.Attention)
	}
	roles := map[string]int{}
	for _, kc := range sales.Attention.UnresolvedReferencesByRole {
		roles[kc.Kind] = kc.Count
	}
	if roles[LinkRoleRelationTarget] != 1 || roles[LinkRoleAttributeType] != 1 {
		t.Fatalf("unresolved by role: %+v", sales.Attention.UnresolvedReferencesByRole)
	}
	if !sales.Attention.NeedsAttention {
		t.Fatal("a section with errors must need attention")
	}
	app := sectionFor(t, o, "application", "order_entry")
	if app.Attention.ValidationWarnings != 1 || app.Attention.ValidationErrors != 0 || app.Attention.UnresolvedReferences != 0 {
		t.Fatalf("application roll-up: %+v", app.Attention)
	}
	if app.Descendants != 2 {
		t.Fatalf("application descendants = %d", app.Descendants)
	}
}

// Workspace-declared elements the Source Index cannot point at are counted;
// the always-present prelude is not.
func TestOverviewCountsElementsWithoutSourceReferenceButNotThePrelude(t *testing.T) {
	o := overviewFor(t, nil, nil, 50)
	sales := sectionFor(t, o, "semantic", "sales")
	// sales and sales.order are indexed; the other 8 sales elements are not.
	if sales.Attention.WithoutSourceReference != sales.Descendants+1-2 {
		t.Fatalf("withoutSourceReference=%d descendants=%d", sales.Attention.WithoutSourceReference, sales.Descendants)
	}
	for _, s := range o.Sections {
		if s.Entry.Prelude && s.Attention.WithoutSourceReference != 0 {
			t.Fatalf("prelude section counted as missing Source: %+v", s)
		}
	}
}

// A cut list must not hide a problem: totals cover omitted sections too.
func TestOverviewTotalsCoverOmittedSections(t *testing.T) {
	diagnostics := []validation.Diagnostic{
		{Code: "X", Severity: validation.Error, Subject: subjectPtr("semantic-attribute", "sales.order.note")},
	}
	full := overviewFor(t, diagnostics, nil, 50)
	cut := overviewFor(t, diagnostics, nil, 1)
	if cut.SectionTotal != full.SectionTotal || len(cut.Sections) != 1 || cut.SectionsOmitted != full.SectionTotal-1 {
		t.Fatalf("cut: total=%d listed=%d omitted=%d", cut.SectionTotal, len(cut.Sections), cut.SectionsOmitted)
	}
	if cut.Totals.ValidationErrors != 1 || cut.Totals.UnresolvedReferences != full.Totals.UnresolvedReferences ||
		cut.Totals.WithoutSourceReference != full.Totals.WithoutSourceReference {
		t.Fatalf("totals changed with the limit:\n cut=%+v\n full=%+v", cut.Totals, full.Totals)
	}
}

// Findings that belong to no catalog element are kept, by kind, instead of
// disappearing from the overview.
func TestOverviewKeepsUnattributedFindings(t *testing.T) {
	diagnostics := []validation.Diagnostic{
		{Code: "A", Severity: validation.Error, Subject: subjectPtr("semantic-period", "sales.fy")},
		{Code: "B", Severity: validation.Warning},
		{Code: "C", Severity: validation.Error, Subject: subjectPtr("semantic-data", "sales.nowhere")},
	}
	findings := []workspace.WorkspaceSourceRootFinding{{Root: "x", Severity: "error"}, {Root: "y", Severity: "warning"}}
	o := overviewFor(t, diagnostics, findings, 50)
	u := o.Unattributed
	if u.ValidationErrors != 2 || u.ValidationWarnings != 1 || u.SourceRootErrors != 1 || u.SourceRootWarnings != 1 {
		t.Fatalf("unattributed: %+v", u)
	}
	kinds := map[string]int{}
	for _, kc := range u.DiagnosticsByKind {
		kinds[kc.Kind] = kc.Count
	}
	if kinds["semantic-period"] != 1 || kinds["(none)"] != 1 || kinds["semantic-data"] != 1 {
		t.Fatalf("unattributed kinds: %+v", u.DiagnosticsByKind)
	}
	if o.Totals.ValidationErrors != 0 {
		t.Fatalf("an unattributed finding was also counted on a section: %+v", o.Totals)
	}
}

// A duplicated declaration is counted once as an element and reported as
// ambiguous on its section.
func TestOverviewReportsAmbiguousDeclarations(t *testing.T) {
	world := locationWorld()
	world.Applications = append(world.Applications, application.Application{ID: "order_entry"})
	catalog := CatalogWorld(world, sourceindex.Index{}, "", "w", "s")
	o := OverviewFromRead(catalog, world, nil, nil, 50)
	count := 0
	for _, s := range o.Sections {
		if s.Entry.Subject == (sourceindex.Subject{Kind: "application", ID: "order_entry"}) {
			count++
			if s.Attention.AmbiguousElements == 0 {
				t.Fatalf("duplicate application not marked ambiguous: %+v", s.Attention)
			}
		}
	}
	if count == 0 {
		t.Fatal("application section missing")
	}
}

// model locate reports the same roll-up for the element it locates, so the
// detail and the overview cannot disagree.
func TestLocateAttentionMatchesTheOverviewSection(t *testing.T) {
	diagnostics := []validation.Diagnostic{
		{Code: "X", Severity: validation.Error, Subject: subjectPtr("semantic-attribute", "sales.order.note")},
	}
	world := locationWorld()
	catalog := CatalogWorld(world, sourceindex.Index{}, "", "w", "s")
	tree := newCatalogTree(catalog, world, diagnostics)
	o := OverviewFromRead(catalog, world, diagnostics, nil, 50)
	section := sectionFor(t, o, "semantic", "sales")
	got := tree.subtreeAttention(sourceindex.Subject{Kind: "semantic", ID: "sales"})
	if got.ValidationErrors != section.Attention.ValidationErrors ||
		got.UnresolvedReferences != section.Attention.UnresolvedReferences ||
		got.WithoutSourceReference != section.Attention.WithoutSourceReference {
		t.Fatalf("locate %+v vs overview %+v", got, section.Attention)
	}
	order := tree.subtreeAttention(sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"})
	if order.ValidationErrors != 1 || order.UnresolvedReferences != 2 {
		t.Fatalf("sales.order subtree: %+v", order)
	}
	customer := tree.subtreeAttention(sourceindex.Subject{Kind: "semantic-data", ID: "sales.customer"})
	if customer.ValidationErrors != 0 || customer.UnresolvedReferences != 0 {
		t.Fatalf("problem leaked into a sibling subtree: %+v", customer)
	}
}
