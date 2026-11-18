package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/query"
)

// TestHookRowSourceRefusesTwoDifferentSources checks that a pattern cannot declare
// two row-scoped hooks with different sources.
//
// The runner is handed one row set per invocation, so two different sources cannot
// both be applied. Without this check the pattern passed validation and every
// execution refused it with HOOK_ROW_SOURCE_MISMATCH - admitted and never applied,
// which is the shape this repository keeps removing.
func TestHookRowSourceRefusesTwoDifferentSources(t *testing.T) {
	w := rowSourceHookWorld()
	// A second real query, so the two sources resolve and the only thing left
	// wrong is that they differ.
	w.Queries = append(w.Queries, query.EmbeddedSQL{ID: "order_entry.second_query", Application: "order_entry", Datasource: "sales"})
	interactionRecordPattern(&w).Interaction = &application.Interaction{
		Hooks: []application.Hook{
			{Phase: "before-update", Scope: "row", Rows: &application.HookRowSource{Query: "order_entry.recent_orders"}},
			{Phase: "after-update", Scope: "row", Rows: &application.HookRowSource{Query: "order_entry.second_query"}},
		},
	}
	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_DIFFERS") == nil {
		t.Fatalf("two row sources were accepted: %#v", ds)
	}
}

// TestHookRowSourceAcceptsOneSharedSource checks the other direction: two
// row-scoped hooks naming the same source are fine, because one row set serves
// both.
func TestHookRowSourceAcceptsOneSharedSource(t *testing.T) {
	w := rowSourceHookWorld()
	// A second real query, so the two sources resolve and the only thing left
	// wrong is that they differ.
	w.Queries = append(w.Queries, query.EmbeddedSQL{ID: "order_entry.second_query", Application: "order_entry", Datasource: "sales"})
	interactionRecordPattern(&w).Interaction = &application.Interaction{
		Hooks: []application.Hook{
			{Phase: "before-update", Scope: "row", Rows: &application.HookRowSource{Query: "order_entry.recent_orders"}},
			{Phase: "after-update", Scope: "row", Rows: &application.HookRowSource{Query: "order_entry.recent_orders"}},
		},
	}
	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_DIFFERS") != nil {
		t.Fatalf("one shared source was refused: %#v", ds)
	}
}
