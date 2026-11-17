package main

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
)

// TestDestinationReadTextPrintsEveryObservedRowAndAnswer pins the human
// rendering of a `view -environment` read. The expected lines are written out
// here rather than copied from the product's JSON, so a change to the
// decisions cannot silently change the text as well.
func TestDestinationReadTextPrintsEveryObservedRowAndAnswer(t *testing.T) {
	read := destinationRead{
		Application:   "order_entry",
		Pattern:       "order_entry.order_lines",
		Environment:   "development",
		Relation:      "sales.order.lines",
		KeyAttributes: []core.CanonicalID{"sales.order_line.order_line_id"},
		Coverage:      "partial",
		Rows: []destinationRow{
			{Key: "1"},
			{Key: "2"},
		},
		PerRow: map[string]map[string]string{
			"1": {"sales.order_line.quantity_positive": "satisfied"},
			"2": {"sales.order_line.quantity_positive": "not-satisfied"},
		},
		PerRowUndecided: map[string]map[string]string{
			"2": {"sales.order_line.slot_unique": "relation-partially-read"},
		},
		PerRelation: map[string]string{
			"sales.order.lines.line_number_unique": "undecided",
		},
		PerRelationUndecided: map[string]string{
			"sales.order.lines.line_number_unique": "relation-partially-read",
		},
	}

	var out bytes.Buffer
	code, err := writeDestinationReadText(&out, read)
	if err != nil || code != 0 {
		t.Fatalf("text rendering failed: code=%d err=%v", code, err)
	}

	got := out.String()
	want := []string{
		"application order_entry",
		"pattern order_entry.order_lines",
		"environment development",
		"relation sales.order.lines",
		"coverage partial",
		"key sales.order_line.order_line_id",
		"rows 2",
		"row 1",
		"  sales.order_line.quantity_positive satisfied",
		"row 2",
		"  sales.order_line.quantity_positive not-satisfied",
		"  sales.order_line.slot_unique undecided (relation-partially-read)",
		"sales.order.lines.line_number_unique undecided (relation-partially-read)",
	}
	for _, line := range want {
		if !strings.Contains(got, line+"\n") {
			t.Fatalf("text output is missing %q; got:\n%s", line, got)
		}
	}

	// An empty read must not claim rows it did not observe.
	var emptyOut bytes.Buffer
	if code, err := writeDestinationReadText(&emptyOut, destinationRead{
		Application: "order_entry", Pattern: "order_entry.order_lines",
		Environment: "development", Relation: "sales.order.lines", Coverage: "partial",
	}); err != nil || code != 0 {
		t.Fatalf("empty text rendering failed: code=%d err=%v", code, err)
	}
	if !strings.Contains(emptyOut.String(), "rows 0\n") {
		t.Fatalf("empty read did not report zero rows: %s", emptyOut.String())
	}
}
