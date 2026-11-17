package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/query"
)

// rowSourceHookWorld is hookWorld with a row-scoped hook that names its supplier.
//
// The source query's result mapping carries the attribute the hook reads, so the
// declaration is one the conversion can actually feed. The query is the existing
// browse query of the fixture; a record pattern cannot declare one itself, which
// is why the source is written on the hook.
func rowSourceHookWorld() ir.World {
	w := hookWorld()
	for i := range w.Queries {
		if w.Queries[i].ID != "order_entry.recent_orders" {
			continue
		}
		w.Queries[i].ResultColumns = []query.ResultColumnMapping{
			{Column: "ORDER_ID", Attribute: "sales.order.order_id"},
		}
	}
	declaration := &interactionRecordPattern(&w).Interaction.Hooks[0]
	declaration.Scope = "row"
	declaration.Reads = []core.CanonicalID{"sales.order.order_id"}
	declaration.Rows = &application.HookRowSource{Query: "order_entry.recent_orders"}
	return w
}

// TestHookRowSourceIsAcceptedAndFulfilsTheScope checks the shape this change
// exists for: a row-scoped hook whose source resolves passes validation without
// the "no supplier" warning.
func TestHookRowSourceIsAcceptedAndFulfilsTheScope(t *testing.T) {
	for _, diagnostic := range validateApplicationPatterns(rowSourceHookWorld()) {
		if diagnostic.Code == "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SCOPE_HAS_NO_SUPPLIER" {
			t.Fatalf("a declared row source still warns that no host supplies rows: %s", diagnostic.Message)
		}
		if diagnostic.Severity == Error {
			t.Fatalf("a resolvable row source was refused: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
}

// TestHookRowSourceWithoutADeclarationStillWarns checks the other side: a row
// scope with no source keeps the warning, because nothing names the rows.
func TestHookRowSourceWithoutADeclarationStillWarns(t *testing.T) {
	w := hookWorld()
	declaration := &interactionRecordPattern(&w).Interaction.Hooks[0]
	declaration.Scope = "row"
	declaration.Rows = nil
	var warned bool
	for _, diagnostic := range validateApplicationPatterns(w) {
		if diagnostic.Code != "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SCOPE_HAS_NO_SUPPLIER" {
			continue
		}
		warned = true
		if diagnostic.Severity != Warning {
			t.Fatalf("the note must not be an error: severity=%v", diagnostic.Severity)
		}
	}
	if !warned {
		t.Fatal("a row scope with no declared source was accepted silently")
	}
}

// TestHookRowSourceRefusesAnUnknownQuery checks that a source that resolves to
// nothing is refused, rather than admitted and silently skipped at run time.
func TestHookRowSourceRefusesAnUnknownQuery(t *testing.T) {
	w := rowSourceHookWorld()
	interactionRecordPattern(&w).Interaction.Hooks[0].Rows = &application.HookRowSource{Query: "order_entry.missing_list"}
	assertHasDiagnostic(t, validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE")
}

// TestHookRowSourceRefusesAnotherApplicationsQuery checks that the source belongs
// to the pattern's own application: rows of another application's query are not
// this application's record.
func TestHookRowSourceRefusesAnotherApplicationsQuery(t *testing.T) {
	w := rowSourceHookWorld()
	w.Applications = append(w.Applications, application.Application{ID: "picking"})
	for i := range w.Queries {
		if w.Queries[i].ID != "order_entry.recent_orders" {
			continue
		}
		w.Queries[i].ID = "picking.order_list"
		w.Queries[i].Application = "picking"
	}
	// The browse pattern that named the query must not be left pointing at it:
	// only the hook's source moves, so the expected diagnostic is about the source.
	for i := range w.Applications[0].Patterns {
		if w.Applications[0].Patterns[i].ID == "order_entry.recent_orders" {
			w.Applications[0].Patterns[i].Query = "picking.order_list"
		}
	}
	interactionRecordPattern(&w).Interaction.Hooks[0].Rows = &application.HookRowSource{Query: "picking.order_list"}
	assertHasDiagnostic(t, validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_APPLICATION_MISMATCH")
}

// TestHookRowSourceRefusesAReadTheMappingDoesNotCarry checks that the declared
// read has a supplier. A hook reading an attribute the query's mapping never
// produces would be accepted and then see no value for it on every row.
func TestHookRowSourceRefusesAReadTheMappingDoesNotCarry(t *testing.T) {
	w := rowSourceHookWorld()
	interactionRecordPattern(&w).Interaction.Hooks[0].Reads = []core.CanonicalID{"sales.order.customer_id"}
	assertHasDiagnostic(t, validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_READ_MISSING")
}

// TestHookRowSourceRefusesWhenTheScopeIsNotRow checks that the source is not
// admitted on a record-scoped hook: a supplier the declaration does not apply to
// rows would be inert.
func TestHookRowSourceRefusesWhenTheScopeIsNotRow(t *testing.T) {
	w := rowSourceHookWorld()
	interactionRecordPattern(&w).Interaction.Hooks[0].Scope = "record"
	assertHasDiagnostic(t, validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_SCOPE_MISMATCH")
}

// TestHookRowSourceRefusesAnEmptyQuery checks that a source with no query is
// refused rather than read as "no source".
func TestHookRowSourceRefusesAnEmptyQuery(t *testing.T) {
	w := rowSourceHookWorld()
	interactionRecordPattern(&w).Interaction.Hooks[0].Rows = &application.HookRowSource{}
	assertHasDiagnostic(t, validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_INVALID")
}
