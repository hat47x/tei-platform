package physical

import "testing"

// TableKind is what a physical schema table is: a table, a view, or a materialized
// view. A reader branches on it when comparing a schema with a declaration, so
// each value is listed here (#4032).
func TestTableKindsAreCovered(t *testing.T) {
	kinds := []TableKind{TableKindTable, TableKindView, TableKindMaterializedView}
	want := []string{"table", "view", "materialized-view"}
	if len(kinds) != len(want) {
		t.Fatalf("a table kind was dropped: %d values", len(kinds))
	}
	seen := map[string]bool{}
	for i, kind := range kinds {
		if string(kind) != want[i] {
			t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
		}
		if seen[string(kind)] {
			t.Fatalf("table kind %q is declared twice", kind)
		}
		seen[string(kind)] = true
	}
}
