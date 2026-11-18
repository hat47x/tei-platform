package operations

import (
	"testing"

	"tei/internal/database/migration"
	"tei/internal/database/release"
	"tei/internal/ir"
	"tei/internal/releaseplan"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// A Workspace that declares a migration and a solution release sees those kinds
// in declarationCounts, but the catalog lists no entry for them. Without
// sectionScope the catalog cannot say whether the declaration failed to load or
// simply is not addressable here, so a designer reads the empty result as a bug
// (#4128).
func TestCatalogPublishesKindsItDoesNotCover(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID:   "sales",
			Data: []semantic.Data{{ID: "sales.order", Symbol: "order"}},
		}},
		Migrations:       []migration.Definition{{ID: "20261111_001"}},
		DatabaseReleases: []release.Definition{{Database: "sales_core", Version: "5.0.0"}},
		SolutionReleases: []releaseplan.Definition{{ID: "status_index_rollout", Version: "5.0.0"}},
	}
	got := CatalogWorld(world, sourceindex.Index{}, "", "world", "source")

	if len(got.Entries) == 0 {
		t.Fatalf("catalog lists nothing: %+v", got)
	}
	if got.SectionScope.RootsAre != "catalog-root-entries" {
		t.Errorf("scope does not name its selection rule: %q", got.SectionScope.RootsAre)
	}
	notCovered := map[string]bool{}
	for _, kind := range got.SectionScope.NotCoveredKinds {
		notCovered[kind] = true
	}
	for _, kind := range []string{"migration", "database-release", "solution-release"} {
		if !notCovered[kind] {
			t.Errorf("%q is declared but not reported as out of catalog scope: %+v",
				kind, got.SectionScope.NotCoveredKinds)
		}
	}
	// A kind the catalog does list must not appear in the not-covered list.
	for _, kind := range got.SectionScope.CoveredKinds {
		if notCovered[kind] {
			t.Errorf("%q is both covered and not covered", kind)
		}
	}
	// The other direction matters just as much: a kind the catalog DOES list
	// must not be reported as out of scope. Without this, a scope built from an
	// empty covered set would pass every assertion above, because "nothing is
	// covered" also makes "nothing is both covered and not covered" true.
	listed := map[string]bool{}
	for _, entry := range got.Entries {
		listed[entry.Subject.Kind] = true
	}
	for kind := range listed {
		if notCovered[kind] {
			t.Errorf("%q has a catalog entry yet is reported as not covered", kind)
		}
	}
	if len(got.SectionScope.CoveredKinds) != len(listed) {
		t.Errorf("coveredKinds has %d kinds but the catalog lists %d",
			len(got.SectionScope.CoveredKinds), len(listed))
	}
}

// The catalog and model overview publish the same projection, so a caller
// cannot get two different answers about what the catalog covers.
func TestCatalogAndOverviewAgreeOnSectionScope(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID:   "sales",
			Data: []semantic.Data{{ID: "sales.order", Symbol: "order"}},
		}},
		Migrations: []migration.Definition{{ID: "20261111_001"}},
	}
	catalog := CatalogWorld(world, sourceindex.Index{}, "", "world", "source")
	overview := OverviewFromRead(catalog, world, nil, nil, 100)

	if len(catalog.SectionScope.NotCoveredKinds) != len(overview.SectionScope.NotCoveredKinds) {
		t.Fatalf("catalog and overview disagree: catalog=%v overview=%v",
			catalog.SectionScope.NotCoveredKinds, overview.SectionScope.NotCoveredKinds)
	}
	for i := range catalog.SectionScope.NotCoveredKinds {
		if catalog.SectionScope.NotCoveredKinds[i] != overview.SectionScope.NotCoveredKinds[i] {
			t.Fatalf("not-covered kinds differ at %d: %q vs %q", i,
				catalog.SectionScope.NotCoveredKinds[i], overview.SectionScope.NotCoveredKinds[i])
		}
	}
}
