package operations

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	dbmigration "tei/internal/database/migration"
	dbrelease "tei/internal/database/release"
	"tei/internal/ir"
	"tei/internal/releaseplan"
)

// jsonMarshalForTest marshals the counts so the wire names can be asserted.
func jsonMarshalForTest(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// containsJSONText reports whether one JSON document contains a fragment.
func containsJSONText(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// deliveryWorld declares one migration, one application release, one database
// release and one solution release, and nothing else.
func deliveryWorld() ir.World {
	w := ir.World{}
	w.Migrations = []dbmigration.Definition{{
		ID:       "20261111_001",
		Database: core.MustCanonicalID("sales_core"),
	}}
	w.ApplicationReleases = []release.Definition{{
		Application: core.MustCanonicalID("order_entry"),
		Version:     "5.0.0",
	}}
	w.DatabaseReleases = []dbrelease.Definition{{
		Database: core.MustCanonicalID("sales_core"),
		Version:  "5.0.0",
	}}
	w.SolutionReleases = []releaseplan.Definition{{
		ID:      core.MustCanonicalID("status_index_rollout"),
		Version: "5.0.0",
	}}
	// A database is not a delivery declaration; it is counted separately, so
	// the total below stays at four.
	w.Databases = []database.Database{}
	return w
}

// TestDeclarationCountsIncludeTheDeliveryLayer pins #4128. A designer added a
// migration and releases, ran validate, and read `declarationCounts` — which
// did not move, so the counts implied semantic and application declarations
// are the whole model. The delivery layer is now counted too.
func TestDeclarationCountsIncludeTheDeliveryLayer(t *testing.T) {
	counts := CountDeclarations(deliveryWorld())

	if counts.Migrations != 1 {
		t.Fatalf("migrations = %d, want 1", counts.Migrations)
	}
	if counts.ApplicationReleases != 1 {
		t.Fatalf("applicationReleases = %d, want 1", counts.ApplicationReleases)
	}
	if counts.DatabaseReleases != 1 {
		t.Fatalf("databaseReleases = %d, want 1", counts.DatabaseReleases)
	}
	if counts.SolutionReleases != 1 {
		t.Fatalf("solutionReleases = %d, want 1", counts.SolutionReleases)
	}
	// The total must include them, or a caller reading only `total` still sees
	// no change when a release is added.
	if counts.Total != 4 {
		t.Fatalf("total = %d, want 4 (the four delivery declarations)", counts.Total)
	}
}

// A Workspace without delivery declarations must keep reporting zero, so the
// existing minimal-workspace expectations stay meaningful.
func TestDeclarationCountsWithoutDeliveryLayerStayZero(t *testing.T) {
	counts := CountDeclarations(ir.World{})
	if counts.Migrations != 0 || counts.ApplicationReleases != 0 ||
		counts.DatabaseReleases != 0 || counts.SolutionReleases != 0 {
		t.Fatalf("empty World reported delivery declarations: %+v", counts)
	}
	if counts.Total != 0 {
		t.Fatalf("empty World total = %d, want 0", counts.Total)
	}
}

// The JSON names are what callers branch on; they must not be invented by a
// future rename of the Go fields.
func TestDeliveryCountWireNames(t *testing.T) {
	got := DeclarationCounts{Migrations: 1, ApplicationReleases: 2, DatabaseReleases: 3, SolutionReleases: 4}
	raw, err := jsonMarshalForTest(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"migrations":1`, `"applicationReleases":2`, `"databaseReleases":3`, `"solutionReleases":4`} {
		if !containsJSONText(raw, want) {
			t.Fatalf("missing %s in %s", want, raw)
		}
	}
}
