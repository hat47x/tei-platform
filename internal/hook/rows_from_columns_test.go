package hook_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/hook"
)

// mappedRowsWorld declares a row-scoped hook that reads one attribute, so a
// converted query result can be applied to it.
func mappedRowsWorld() *hook.Runner {
	return &hook.Runner{World: rowScopeWorld("row"), Implementation: &rowImplementation{}}
}

// TestRowsFromColumnsKeepsTheDeclaredAttributes checks that a mapped query result
// becomes the row set a row-scoped hook is applied to.
//
// The mapping is explicit: a column becomes an attribute because the result
// mapping named it, so the conversion does not guess a name from the raw column.
func TestRowsFromColumnsKeepsTheDeclaredAttributes(t *testing.T) {
	converted, err := hook.RowsFromColumns([]hook.ColumnRow{
		{
			Columns: []hook.ColumnName{
				{Raw: "AMOUNT", Attribute: "sales.order.amount"},
				{Raw: "NOTE", Attribute: ""},
			},
			Values: []any{"10", "not-declared"},
		},
		{
			Columns: []hook.ColumnName{{Raw: "AMOUNT", Attribute: "sales.order.amount"}},
			Values:  []any{20},
		},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(converted) != 2 {
		t.Fatalf("rows=%d", len(converted))
	}
	if converted[0]["sales.order.amount"] != "10" {
		t.Fatalf("row 0 amount=%q", converted[0]["sales.order.amount"])
	}
	if _, carried := converted[0][core.CanonicalID("NOTE")]; carried {
		t.Fatal("a column with no mapped attribute was carried into the row set")
	}
	// A number keeps its printed form so a declared read has a value to compare.
	if converted[1]["sales.order.amount"] != "20" {
		t.Fatalf("row 1 amount=%q", converted[1]["sales.order.amount"])
	}
}

// TestRowsFromColumnsMakesANilValuePresentNotMissing checks the row type's limit
// rather than hiding it: a nil value is carried as empty, so the attribute counts
// as read.
func TestRowsFromColumnsMakesANilValuePresentNotMissing(t *testing.T) {
	converted, err := hook.RowsFromColumns([]hook.ColumnRow{{
		Columns: []hook.ColumnName{{Raw: "AMOUNT", Attribute: "sales.order.amount"}},
		Values:  []any{nil},
	}})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	value, present := converted[0]["sales.order.amount"]
	if !present || value != "" {
		t.Fatalf("a nil value was not carried as an empty string: present=%v value=%q", present, value)
	}
}

// TestRowsFromColumnsRefusesAnUnevenRow checks that a row whose columns and values
// disagree is refused instead of being silently shortened.
func TestRowsFromColumnsRefusesAnUnevenRow(t *testing.T) {
	if _, err := hook.RowsFromColumns([]hook.ColumnRow{{
		Columns: []hook.ColumnName{{Raw: "A", Attribute: "sales.a"}, {Raw: "B", Attribute: "sales.b"}},
		Values:  []any{"1"},
	}}); err == nil {
		t.Fatal("a row with two columns and one value was accepted")
	}
}

// TestConvertedRowsReachTheRowHook checks the two pieces together: a mapped result
// becomes rows, and the row-scoped hook is applied once per row with that row's
// declared value.
func TestConvertedRowsReachTheRowHook(t *testing.T) {
	converted, err := hook.RowsFromColumns([]hook.ColumnRow{
		{Columns: []hook.ColumnName{{Raw: "AMOUNT", Attribute: "sales.order.amount"}}, Values: []any{"10"}},
		{Columns: []hook.ColumnName{{Raw: "AMOUNT", Attribute: "sales.order.amount"}}, Values: []any{"20"}},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	implementation := &rowImplementation{}
	runner := hook.Runner{World: rowScopeWorld("row"), Implementation: implementation, Rows: converted}
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
