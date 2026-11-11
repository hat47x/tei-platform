package releasegraph

import "testing"

// NodeKind labels every node of a release graph. A reader branches on it to know
// what is being released, so the set is part of this package's contract (#4032).
func TestReleaseNodeKindsAreCovered(t *testing.T) {
	kinds := []NodeKind{NodeDatabaseState, NodeDatabaseReadiness, NodeApplicationRelease}
	want := []string{"database-state", "database-readiness", "application-release"}
	if len(kinds) != len(want) {
		t.Fatalf("a node kind was dropped: %d values", len(kinds))
	}
	for i, kind := range kinds {
		if string(kind) != want[i] {
			t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
		}
	}
}
