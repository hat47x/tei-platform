package plan

import "testing"

// NodeKind labels every node of a query plan. A reader of a plan branches on it
// to explain cost and shape, so the set is part of this package's contract
// (#4032).
func TestNodeKindsAreCovered(t *testing.T) {
	kinds := []NodeKind{
		NodeScan, NodeIndexScan, NodeJoin, NodeAggregate, NodeSort, NodeMaterialize,
		NodeLimit, NodeModify, NodeSetOperation, NodeGather, NodeOther,
	}
	want := []string{
		"scan", "index-scan", "join", "aggregate", "sort", "materialize",
		"limit", "modify", "set-operation", "gather", "other",
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
