package query

import "testing"

// RelNodeKind labels every node of a relational plan. A reader explains cost and
// shape from it, so the set is part of this package's contract (#4032).
func TestRelationalNodeKindsAreCovered(t *testing.T) {
	kinds := []RelNodeKind{
		RelSource, RelJoin, RelFilter, RelAggregate, RelProject,
		RelDistinct, RelSort, RelLimit, RelOpaque,
	}
	want := []string{
		"source", "join", "filter", "aggregate", "project",
		"distinct", "sort", "limit", "opaque",
	}
	if len(kinds) != len(want) {
		t.Fatalf("a node kind was dropped: %d values", len(kinds))
	}
	seen := map[string]bool{}
	for i, kind := range kinds {
		if string(kind) != want[i] {
			t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
		}
		if seen[string(kind)] {
			t.Fatalf("node kind %q is declared twice", kind)
		}
		seen[string(kind)] = true
	}
}
