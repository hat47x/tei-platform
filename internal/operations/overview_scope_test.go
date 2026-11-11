package operations

import (
	"testing"

	"tei/internal/workspace"
)

// referenceOverview builds the overview for the reference Workspace, the same
// way the operation does.
func referenceOverview(t *testing.T) ModelOverview {
	t.Helper()
	loaded, err := workspace.Load("../../reference/workspace")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	idx, err := workspace.BuildSourceIndex("../../reference/workspace", loaded)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	catalog := CatalogWorld(loaded.World, idx, "", "world", "source")
	return OverviewFromRead(catalog, loaded.World, nil, nil, OverviewDefaultLimit)
}

// The overview must say what its sections cover, because a section count reads
// like a count of everything (#4128).
func TestOverviewSectionScopeIsStated(t *testing.T) {
	out := referenceOverview(t)
	if out.SectionScope.RootsAre != "catalog-root-entries" {
		t.Errorf("rootsAre is %q, want catalog-root-entries", out.SectionScope.RootsAre)
	}
	if len(out.SectionScope.CoveredKinds) == 0 {
		t.Fatal("coveredKinds is empty, so the statement says nothing")
	}
	// The reference workspace declares migrations and releases, and the catalog
	// does not list them. If that ever changes, this test should change with it.
	for _, kind := range []string{"migration", "database-release", "solution-release"} {
		if !contains(out.SectionScope.NotCoveredKinds, kind) {
			t.Errorf("%q is declared in the reference workspace but is not named as not covered: %v",
				kind, out.SectionScope.NotCoveredKinds)
		}
	}
}

// Every section root is a catalog entry, so the statement about roots matches
// what the sections actually are.
func TestEverySectionRootIsACatalogEntry(t *testing.T) {
	out := referenceOverview(t)
	kinds := map[string]bool{}
	for _, kind := range out.SectionScope.CoveredKinds {
		kinds[kind] = true
	}
	for _, section := range out.Sections {
		if !kinds[section.Entry.Subject.Kind] {
			t.Errorf("section %s:%s has kind %q, which coveredKinds does not list",
				section.Entry.Subject.Kind, section.Entry.Subject.ID, section.Entry.Subject.Kind)
		}
	}
}

// The table of declared layers must not silently miss a layer the reference
// workspace declares: every entry with a count must be either covered or named.
func TestDeclaredLayerKindsAreAccountedFor(t *testing.T) {
	out := referenceOverview(t)
	covered := map[string]bool{}
	for _, kind := range out.SectionScope.CoveredKinds {
		covered[kind] = true
	}
	named := map[string]bool{}
	for _, kind := range out.SectionScope.NotCoveredKinds {
		named[kind] = true
	}
	loaded, err := workspace.Load("../../reference/workspace")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, layer := range declaredLayerKinds {
		if layer.count(loaded.World) == 0 {
			continue // not declared here, so the overview need not mention it
		}
		if !covered[layer.kind] && !named[layer.kind] {
			t.Errorf("%q is declared but appears in neither list", layer.kind)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// Every not-covered kind must be either readable by name or absent from
// inspect's accepted kinds; nothing may be listed in both answers or in neither.
func TestReadableWithoutSectionMatchesInspect(t *testing.T) {
	out := referenceOverview(t)
	notCovered := map[string]bool{}
	for _, kind := range out.SectionScope.NotCoveredKinds {
		notCovered[kind] = true
	}
	readable := map[string]bool{}
	for _, kind := range out.SectionScope.ReadableWithoutSection {
		if !notCovered[kind] {
			t.Errorf("%q is readable but is not in notCoveredKinds", kind)
		}
		if readable[kind] {
			t.Errorf("%q appears twice in readableWithoutSection", kind)
		}
		readable[kind] = true
	}
	// The split must account for every not-covered kind: one of the two lists.
	for _, kind := range out.SectionScope.NotCoveredKinds {
		if readable[kind] {
			continue
		}
		if supportedInspectionKind(kind) {
			t.Errorf("%q is accepted by inspect but was not listed as readable", kind)
		}
	}
}

// A designer reading model overview must be able to tell that a declared kind
// with no section is still readable (#4128).
func TestReadableWithoutSectionIsNotEmptyForDeclaredLayers(t *testing.T) {
	out := referenceOverview(t)
	if len(out.SectionScope.ReadableWithoutSection) == 0 {
		t.Fatalf("no readable-without-section kinds although the reference declares several: %v",
			out.SectionScope.NotCoveredKinds)
	}
	for _, kind := range []string{"database", "environment", "persistence-mapping"} {
		found := false
		for _, read := range out.SectionScope.ReadableWithoutSection {
			if read == kind {
				found = true
			}
		}
		if !found {
			t.Errorf("%q should be reported as readable without a section", kind)
		}
	}
}
