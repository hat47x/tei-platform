package main

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/core"
)

// TestWriteViewTextPrintsTheRowSource checks that the printed projection shows
// where a row-scoped hook takes its rows from. A reader who cannot see the source
// cannot tell a declaration with a supplier from one without, which is the
// difference the warning on the validation side is about.
func TestWriteViewTextPrintsTheRowSource(t *testing.T) {
	view := applicationview.View{
		Pattern: "order_entry.order_record",
		Kind:    application.PatternRecord,
		Data:    "sales.order",
		Items:   []applicationview.Item{{Attribute: "sales.order.amount", Type: "std.decimal"}},
		Hooks: []applicationview.HookRef{{
			Phase: "before-update",
			Scope: "row",
			Rows:  &application.HookRowSource{Query: "order_entry.order_list"},
			Reads: []core.CanonicalID{"sales.order.amount"},
		}},
	}
	var out bytes.Buffer
	if code, err := writeViewText(&out, view); err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	got := out.String()
	if !strings.Contains(got, "scope row") {
		t.Fatalf("the printed hook does not show its scope: %s", got)
	}
	if !strings.Contains(got, "rows from query order_entry.order_list") {
		t.Fatalf("the printed hook does not show its row source: %s", got)
	}
}

// TestWriteViewTextPrintsNoRowSourceWhenNoneIsDeclared checks the other side: an
// absent source is printed as absent rather than as an empty source, so the
// declaration's silence stays visible.
func TestWriteViewTextPrintsNoRowSourceWhenNoneIsDeclared(t *testing.T) {
	view := applicationview.View{
		Pattern: "order_entry.order_record",
		Kind:    application.PatternRecord,
		Data:    "sales.order",
		Items:   []applicationview.Item{{Attribute: "sales.order.amount", Type: "std.decimal"}},
		Hooks: []applicationview.HookRef{{
			Phase: "before-update",
			Scope: "row",
			Reads: []core.CanonicalID{"sales.order.amount"},
		}},
	}
	var out bytes.Buffer
	if code, err := writeViewText(&out, view); err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if strings.Contains(out.String(), "rows from") {
		t.Fatalf("a hook with no row source was printed as having one: %s", out.String())
	}
}
