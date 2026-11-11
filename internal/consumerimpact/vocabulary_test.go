package consumerimpact

import "testing"

// ChangeKind is written into a review result and read by a caller deciding what
// to report, so each value is shown here rather than only declared (#4032).
func TestChangeKindsAreCovered(t *testing.T) {
	kinds := []ChangeKind{ChangeAdded, ChangeWithdrawn, ChangeMajorChanged, ChangeMinorChanged}
	want := []string{"added", "withdrawn", "major-changed", "minor-changed"}
	if len(kinds) != len(want) {
		t.Fatalf("a change kind was dropped: %d values", len(kinds))
	}
	seen := map[string]bool{}
	for i, kind := range kinds {
		if string(kind) != want[i] {
			t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
		}
		if seen[string(kind)] {
			t.Fatalf("kind %q is declared twice", kind)
		}
		seen[string(kind)] = true
	}
}
