package reconcile

import "testing"

// FindingKind is what a reconciliation pass reports about a database. An operator
// branches on it, so each value is listed here rather than only declared (#4032).
func TestFindingKindsAreCovered(t *testing.T) {
	kinds := []FindingKind{
		FindingPendingMigration, FindingFailedMigration,
		FindingDivergedMigration, FindingOrphanApplied, FindingSchemaDrift,
	}
	want := []string{
		"pending-migration", "failed-migration",
		"diverged-migration", "orphan-applied-migration", "schema-drift",
	}
	if len(kinds) != len(want) {
		t.Fatalf("a finding kind was dropped: %d values", len(kinds))
	}
	seen := map[string]bool{}
	for i, kind := range kinds {
		if string(kind) != want[i] {
			t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
		}
		if seen[string(kind)] {
			t.Fatalf("finding kind %q is declared twice", kind)
		}
		seen[string(kind)] = true
	}
}
