package main

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
)

// plan/NOW.md asks that rows and their rule answers be readable without a JSON
// reader. This fixes the shape of that text: the row key, then each rule and
// the answer the evaluation produced, and an undecided rule keeps its reason
// instead of being dropped or printed as satisfied (#2586).
func TestWriteEnvironmentReadTextNamesRowsAndAnswers(t *testing.T) {
	read := destinationRead{
		Application:   "order_entry",
		Pattern:       "order_entry.order_lines",
		Environment:   "development",
		Relation:      "sales.order.lines",
		KeyAttributes: []core.CanonicalID{"sales.order_line.order_line_id"},
		Coverage:      "partial",
		Rows: []destinationRow{
			{Key: `["1002"]`, Fields: map[string]destinationField{"sales.order_line.quantity": {State: "present", Value: "2"}}},
			{Key: `["1001"]`, Fields: map[string]destinationField{"sales.order_line.quantity": {State: "present", Value: "3"}}},
		},
		PerRow: map[string]map[string]string{
			`["1001"]`: {"order_entry.order_lines.quantity_positive": "satisfied"},
			`["1002"]`: {"order_entry.order_lines.quantity_positive": "satisfied"},
		},
		PerRowUndecided: map[string]map[string]string{
			`["1002"]`: {"order_entry.order_lines.line_number_unique": "no fetch coverage declared"},
		},
	}

	var out bytes.Buffer
	if code, err := writeEnvironmentReadText(&out, read); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	got := out.String()

	for _, want := range []string{
		"pattern order_entry.order_lines (order_entry)",
		"relation sales.order.lines",
		"environment development",
		"coverage partial",
		"rows: 2",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing header line %q in:\n%s", want, got)
		}
	}
	// Rows must be printed in a stable order, so two runs are comparable.
	if strings.Index(got, `["1001"]`) > strings.Index(got, `["1002"]`) {
		t.Errorf("rows are not in a stable order:\n%s", got)
	}
	// Each answered rule names the answer the evaluation gave.
	if !strings.Contains(got, "order_entry.order_lines.quantity_positive: satisfied") {
		t.Errorf("an answered rule is missing:\n%s", got)
	}
	// An undecided rule is not printed as a plain answer: it keeps its reason.
	if !strings.Contains(got, "line_number_unique: no fetch coverage declared") {
		t.Errorf("an undecided rule lost its reason:\n%s", got)
	}
	// A rule with an answer must not also be listed as undecided.
	if strings.Count(got, "quantity_positive") != 2 {
		t.Errorf("an answered rule was printed %d times, want once per row:\n%s",
			strings.Count(got, "quantity_positive"), got)
	}
}

// An empty read still says it has no rows, rather than printing nothing.
func TestWriteEnvironmentReadTextSaysWhenThereAreNoRows(t *testing.T) {
	read := destinationRead{Pattern: "order_entry.order_lines", Relation: "sales.order.lines", Coverage: "partial"}
	var out bytes.Buffer
	if code, err := writeEnvironmentReadText(&out, read); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(out.String(), "rows: (none)") {
		t.Fatalf("an empty read did not say so:\n%s", out.String())
	}
}

// The relation-level answers are printed once, outside the row blocks.
func TestWriteEnvironmentReadTextPrintsRelationAnswers(t *testing.T) {
	read := destinationRead{
		Pattern:     "order_entry.order_lines",
		Relation:    "sales.order.lines",
		Coverage:    "partial",
		Rows:        []destinationRow{{Key: `["1001"]`}},
		PerRow:      map[string]map[string]string{`["1001"]`: {"order_entry.order_lines.quantity_positive": "satisfied"}},
		PerRelation: map[string]string{"order_entry.order_lines.line_number_unique": "partial"},
		PerRelationUndecided: map[string]string{
			"order_entry.order_lines.line_number_unique": "no fetch coverage declared",
		},
	}
	var out bytes.Buffer
	if code, err := writeEnvironmentReadText(&out, read); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	got := out.String()
	if !strings.Contains(got, "line_number_unique: partial (no fetch coverage declared)") {
		t.Fatalf("a relation answer lost its reason:\n%s", got)
	}
}
