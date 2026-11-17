package main

import (
	"context"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/query"
)

// rowSourceHookWorld declares a record pattern whose row-scoped hook names where
// its rows come from, plus the source query the hook points at. The shape is the
// one validation accepts: the source resolves and its mapping carries the read.
func rowSourceHookWorld() ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{Hooks: []application.Hook{{
					Phase: "before-update",
					Scope: "row",
					Rows:  &application.HookRowSource{Query: "order_entry.order_list"},
					Reads: []core.CanonicalID{"sales.order.amount"},
				}}},
			}},
		}},
		Queries: []query.EmbeddedSQL{{
			ID:          "order_entry.order_list",
			Application: "order_entry",
			ResultColumns: []query.ResultColumnMapping{
				{Column: "AMOUNT", Attribute: "sales.order.amount"},
			},
		}},
	}
}

// TestRunDeclaredHooksForRowSourceAppliesTheHookPerRow is the host wiring check:
// the source result the host already holds becomes the row set, and the
// row-scoped hook runs once per row with that row's value.
func TestRunDeclaredHooksForRowSourceAppliesTheHookPerRow(t *testing.T) {
	// The mapped columns are the query's own result mapping, so this test also
	// pins that the host passes what the query declared rather than a column list
	// it invented.
	world := rowSourceHookWorld()
	result := queryResultRowSource("order_entry.order_list", world.Queries[0].ResultColumns, [][]any{{"10"}, {"20"}, {"30"}})
	rule := &countingRowRule{}
	report, err := runDeclaredHooksForRowSource(context.Background(), world, "order_entry", "order_entry.order_record", result, rule)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rule.values) != 3 || rule.values[0] != "10" || rule.values[1] != "20" || rule.values[2] != "30" {
		t.Fatalf("the hook was not applied once per row: %v", rule.values)
	}
	if report == nil || len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("report=%+v", report)
	}
}

// TestRunDeclaredHooksForRowSourceRefusesAnotherQuerysResult makes the declaration
// the guard: a page from a query the hook did not name is refused rather than
// applied to rows that belong to another subject.
func TestRunDeclaredHooksForRowSourceRefusesAnotherQuerysResult(t *testing.T) {
	world := rowSourceHookWorld()
	result := queryResultRowSource("order_entry.other_list", world.Queries[0].ResultColumns, [][]any{{"10"}})
	if _, err := runDeclaredHooksForRowSource(context.Background(), world, "order_entry", "order_entry.order_record", result, &countingRowRule{}); err == nil {
		t.Fatal("a result from another query was applied to the declaration")
	}
}

// TestRunDeclaredHooksForRowSourceReportsAnEmptyPageAsSkipped pins the honest
// limit of this wiring: an empty page cannot be told from "the caller supplied no
// row set", because hook.Runner.Rows only carries the rows themselves. The source
// was consulted and matched nothing, but the phase is reported as skipped with a
// reason rather than as executed, so an empty page never reads as a passed phase
// and a supplied empty page never reads as a missing one either.
//
// Closing this needs the row set to carry its own "was supplied" bit — see the
// evidence record. The limit is pinned here so a later fix announces itself by
// failing this test.
func TestRunDeclaredHooksForRowSourceReportsAnEmptyPageAsSkipped(t *testing.T) {
	world := rowSourceHookWorld()
	result := queryResultRowSource("order_entry.order_list", world.Queries[0].ResultColumns, nil)
	rule := &countingRowRule{}
	report, err := runDeclaredHooksForRowSource(context.Background(), world, "order_entry", "order_entry.order_record", result, rule)
	if err != nil {
		t.Fatalf("an empty page was refused: %v", err)
	}
	if len(rule.values) != 0 {
		t.Fatalf("a rule ran for an empty page: %v", rule.values)
	}
	if report == nil {
		t.Fatal("no hook report for a pattern that declares a hook")
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "before-update" {
		t.Fatalf("skipped=%v executed=%v", report.Skipped, report.Executed)
	}
	if reason := report.Reasons["before-update"]; !strings.Contains(reason, "order_entry.order_list") {
		t.Fatalf("the reason does not name the declared source: %q", reason)
	}
}
