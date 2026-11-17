package hook_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// rowSourceWorld declares a record pattern whose row-scoped hook names the query
// its rows come from. The world names the query too, so a conversion can be
// checked against a declaration that validation would accept.
func rowSourceWorld() ir.World {
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
	}
}

// amountColumns is the result mapping of the source query: one column, carried as
// the attribute the hook declares to read.
func amountColumns() []hook.ColumnName {
	return []hook.ColumnName{{Raw: "AMOUNT", Attribute: "sales.order.amount"}}
}

// TestRowsForDeclaredSourceConvertsTheDeclaredQueryResult checks that the source
// a hook names is the source whose result is converted: the rows reach the runner
// with their own values, not as one record.
func TestRowsForDeclaredSourceConvertsTheDeclaredQueryResult(t *testing.T) {
	rows, err := hook.RowsForDeclaredSourceStrings(rowSourceWorld(), "order_entry", "order_entry.order_record", "order_entry.order_list", amountColumns(), [][]any{{"10"}, {"20"}})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0]["sales.order.amount"] != "10" || rows[1]["sales.order.amount"] != "20" {
		t.Fatalf("rows=%v", rows)
	}

	implementation := &rowImplementation{}
	runner := hook.Runner{World: rowSourceWorld(), Implementation: implementation, Rows: rows}
	if err := runner.Run(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(implementation.seen) != 2 {
		t.Fatalf("the hook was not applied once per converted row: %v", implementation.seen)
	}
	if implementation.seen[0]["sales.order.amount"] != "10" || implementation.seen[1]["sales.order.amount"] != "20" {
		t.Fatalf("rows did not keep their own values: %v", implementation.seen)
	}
}

// TestRowsForDeclaredSourceRefusesAnotherQuerysResult checks that a page from a
// different query cannot be applied to a declaration that named this one: the
// rows would be the wrong subject even though their shape fits.
func TestRowsForDeclaredSourceRefusesAnotherQuerysResult(t *testing.T) {
	_, err := hook.RowsForDeclaredSourceStrings(rowSourceWorld(), "order_entry", "order_entry.order_record", "order_entry.other_list", amountColumns(), [][]any{{"10"}})
	if err == nil {
		t.Fatal("a result from another query was accepted")
	}
	var hookErr *hook.Error
	if !errors.As(err, &hookErr) || hookErr.Code != "HOOK_ROW_SOURCE_MISMATCH" {
		t.Fatalf("err=%v", err)
	}
}

// TestRowsForDeclaredSourceRefusesAnUndeclaredSource checks that a row scope with
// no source is not fed whatever rows the host happened to hold.
func TestRowsForDeclaredSourceRefusesAnUndeclaredSource(t *testing.T) {
	world := rowSourceWorld()
	world.Applications[0].Patterns[0].Interaction.Hooks[0].Rows = nil
	_, err := hook.RowsForDeclaredSourceStrings(world, "order_entry", "order_entry.order_record", "order_entry.order_list", amountColumns(), [][]any{{"10"}})
	if err == nil {
		t.Fatal("rows were accepted for a row scope with no declared source")
	}
	var hookErr *hook.Error
	if !errors.As(err, &hookErr) || hookErr.Code != "HOOK_ROW_SOURCE_UNDECLARED" {
		t.Fatalf("err=%v", err)
	}
}

// TestRowsForDeclaredSourceRefusesARowWithoutTheDeclaredRead checks that a row
// that cannot be decided is named instead of being handed over with a missing
// value, which the rule would have to guess about.
func TestRowsForDeclaredSourceRefusesARowWithoutTheDeclaredRead(t *testing.T) {
	columns := []hook.ColumnName{{Raw: "OTHER", Attribute: ""}}
	_, err := hook.RowsForDeclaredSourceStrings(rowSourceWorld(), "order_entry", "order_entry.order_record", "order_entry.order_list", columns, [][]any{{"x"}})
	if err == nil {
		t.Fatal("a row that carries no declared read was accepted")
	}
	var hookErr *hook.Error
	if !errors.As(err, &hookErr) || hookErr.Code != "HOOK_ROW_SOURCE_READ_MISSING" {
		t.Fatalf("err=%v", err)
	}
}

// TestRowsForDeclaredSourceRefusesUnevenValues checks that values are refused when
// they do not line up with the columns, through the same conversion that already
// refuses an uneven row.
func TestRowsForDeclaredSourceRefusesUnevenValues(t *testing.T) {
	columns := []hook.ColumnName{{Raw: "AMOUNT", Attribute: "sales.order.amount"}, {Raw: "NOTE", Attribute: "sales.order.note"}}
	_, err := hook.RowsForDeclaredSourceStrings(rowSourceWorld(), "order_entry", "order_entry.order_record", "order_entry.order_list", columns, [][]any{{"10"}})
	if err == nil {
		t.Fatal("a row with two columns and one value was accepted")
	}
	var hookErr *hook.Error
	if !errors.As(err, &hookErr) || hookErr.Code != "HOOK_ROW_SOURCE_CONVERSION_FAILED" {
		t.Fatalf("err=%v", err)
	}
}

// TestRowsForDeclaredSourceRefusesAPatternWithNoRowScope checks that the entry is
// not a generic "convert any result" door: a pattern without a row-scoped hook
// has no declaration to attribute the rows to.
func TestRowsForDeclaredSourceRefusesAPatternWithNoRowScope(t *testing.T) {
	world := rowSourceWorld()
	world.Applications[0].Patterns[0].Interaction.Hooks[0].Scope = "record"
	_, err := hook.RowsForDeclaredSourceStrings(world, "order_entry", "order_entry.order_record", "order_entry.order_list", amountColumns(), [][]any{{"10"}})
	if err == nil {
		t.Fatal("rows were accepted for a pattern with no row-scoped hook")
	}
	var hookErr *hook.Error
	if !errors.As(err, &hookErr) || hookErr.Code != "HOOK_ROW_SOURCE_NOT_ROW_SCOPED" {
		t.Fatalf("err=%v", err)
	}
}

// TestRowsForDeclaredSourceAcceptsAnEmptyPage checks that an empty result is not a
// failure: nothing matched is a real browse answer.
func TestRowsForDeclaredSourceAcceptsAnEmptyPage(t *testing.T) {
	rows, err := hook.RowsForDeclaredSourceStrings(rowSourceWorld(), "order_entry", "order_entry.order_record", "order_entry.order_list", amountColumns(), nil)
	if err != nil {
		t.Fatalf("an empty page was refused: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows=%v", rows)
	}
}
