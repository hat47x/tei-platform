package main

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
)

func destinationReadFixture() destinationRead {
	return destinationRead{
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
			`["1001"]`: {"sales.order_line.quantity_positive": "satisfied"},
			`["1002"]`: {"sales.order_line.quantity_positive": "not-satisfied"},
		},
		PerRowUndecided: map[string]map[string]string{
			`["1002"]`: {"sales.order_line.line_number_unique": "no fetch coverage declared"},
		},
		PerRelation: map[string]string{"sales.order.lines.line_number_unique": "undecided"},
	}
}

// plan/NOW.md's road item 2 needs the per-row answers visible in a browser.
// The page is a rendering of one destinationRead, so this test compares it
// against that read rather than against the template (#1119).
func TestWriteDestinationReadHTMLShowsRowsAndAnswers(t *testing.T) {
	var out bytes.Buffer
	if code, err := writeDestinationReadHTML(&out, destinationReadFixture()); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	page := out.String()

	for _, want := range []string{
		"<!doctype html>",
		"<title>TEI view -environment — order_entry.order_lines</title>",
		"order_entry.order_lines",
		"sales.order.lines",
		"environment: development",
		"coverage: partial",
		"rows: 2",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q in page", want)
		}
	}
	// Both row keys appear, in the sorted order the read implies rather than
	// the order the rows arrived in.
	if strings.Index(page, `["1001"]`) > strings.Index(page, `["1002"]`) {
		t.Errorf("row keys are not in a stable order")
	}
	// Each answered rule is shown with the answer the evaluation recorded.
	for _, want := range []string{
		"sales.order_line.quantity_positive",
		"satisfied",
		"not-satisfied",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("an answer is missing: %q", want)
		}
	}
	// An undecided rule keeps its reason; it must not be shown as satisfied.
	if !strings.Contains(page, "no fetch coverage declared") {
		t.Errorf("an undecided reason is missing from the page")
	}
	// The relation-level answer is on the page too.
	if !strings.Contains(page, "sales.order.lines.line_number_unique") {
		t.Errorf("the relation answer is missing from the page")
	}
}

// A value from the Workspace must not become markup. html/template escapes
// it, and this checks the escaping is really applied to the fields the page
// prints.
func TestWriteDestinationReadHTMLEscapesValues(t *testing.T) {
	read := destinationReadFixture()
	read.Rows = []destinationRow{{Key: `<script>alert(1)</script>`}}
	read.PerRow = map[string]map[string]string{
		`<script>alert(1)</script>`: {`rule"><img src=x>`: "satisfied"},
	}
	var out bytes.Buffer
	if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	page := out.String()
	if strings.Contains(page, "<script>alert(1)</script>") {
		t.Fatalf("a row key was not escaped")
	}
	if strings.Contains(page, `<img src=x>`) {
		t.Fatalf("a rule name was not escaped")
	}
	if !strings.Contains(page, "&lt;script&gt;") {
		t.Fatalf("the escaped form is missing, so the value may have been dropped")
	}
}

// No rows is a normal answer, not an error, and the page says so.
func TestWriteDestinationReadHTMLSaysWhenThereAreNoRows(t *testing.T) {
	var out bytes.Buffer
	read := destinationRead{Pattern: "order_entry.order_lines", Relation: "sales.order.lines", Coverage: "partial"}
	if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(out.String(), "明細の行はない。") {
		t.Fatalf("an empty read did not say so")
	}
	if !strings.Contains(out.String(), "rows: 0") {
		t.Fatalf("the row count is missing")
	}
}

// The declaration is printed as stated and marked unchecked, so a reader of the
// page cannot mistake it for measured coverage.
func TestWriteDestinationReadHTMLMarksTheDeclarationUnchecked(t *testing.T) {
	read := destinationReadFixture()
	read.Declaration = &destinationDeclaration{CoversRelation: "sales.order.lines", Checked: false}
	var out bytes.Buffer
	if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(out.String(), "covers sales.order.lines (unchecked)") {
		t.Fatalf("the declaration is not marked unchecked")
	}
}
