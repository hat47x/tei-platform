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

// RelationQueryMode and RelationFetchMode are each one closed vocabulary: a
// configured value outside the set is rejected by Valid(), so the declared
// values and the Valid() list have to agree. Naming them here is what keeps a
// new mode from being added without a decision about whether it is valid.
func TestRelationQueryModesAreCovered(t *testing.T) {
	modes := []RelationQueryMode{
		RelationQueryModeTree, RelationQueryModeGraph, RelationQueryModePath,
	}
	want := []string{"tree", "graph", "path"}
	if len(modes) != len(want) || len(RelationQueryModes) != len(want) {
		t.Fatalf("mode list drifted: %d declared, %d in RelationQueryModes, want %d",
			len(modes), len(RelationQueryModes), len(want))
	}
	for i, mode := range modes {
		if string(mode) != want[i] {
			t.Fatalf("modes[%d]=%q want %q", i, mode, want[i])
		}
		if !mode.Valid() {
			t.Fatalf("declared mode %q is not accepted by Valid()", mode)
		}
	}
}

func TestRelationFetchModesAreCovered(t *testing.T) {
	modes := []RelationFetchMode{
		RelationFetchModeEager, RelationFetchModeLazy, RelationFetchModeBatch,
	}
	want := []string{"eager", "lazy", "batch"}
	if len(modes) != len(want) || len(RelationFetchModes) != len(want) {
		t.Fatalf("fetch mode list drifted: %d declared, %d in RelationFetchModes, want %d",
			len(modes), len(RelationFetchModes), len(want))
	}
	for i, mode := range modes {
		if string(mode) != want[i] {
			t.Fatalf("fetch modes[%d]=%q want %q", i, mode, want[i])
		}
		if !mode.Valid() {
			t.Fatalf("declared fetch mode %q is not accepted by Valid()", mode)
		}
	}
}

func TestUnknownRelationModesAreRejected(t *testing.T) {
	if RelationQueryMode("unknown").Valid() || RelationFetchMode("unknown").Valid() {
		t.Fatal("unknown query or fetch mode was accepted")
	}
}
