package main

import (
	"bytes"
	"strings"
	"testing"
)

// The text form and the HTML form are two renderings of one destinationRead.
// The text form already refuses to let a self-contradictory binding read as a
// clean "covers every row" declaration; the HTML form used to print only the
// relation and drop the contradiction. Both must say the same thing, otherwise
// a reader of the page is told something the terminal denies (#5408).
func TestDestinationReadHTMLKeepsAContradictoryDeclarationVisible(t *testing.T) {
	read := destinationRead{
		Pattern:  "order_entry.order_lines",
		Relation: "sales.order.lines",
		Coverage: "partial",
		Declaration: &destinationDeclaration{
			CoversRelation: "sales.order.lines",
			CoversSome:     true,
		},
	}
	var out bytes.Buffer
	if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	page := out.String()
	if !strings.Contains(page, "contradictory") {
		t.Fatalf("a contradictory declaration is presented as clean:\n%s", page)
	}
	// Both halves must survive: dropping either one hides what the binding
	// actually said.
	if !strings.Contains(page, "covers sales.order.lines") || !strings.Contains(page, "covers some") {
		t.Fatalf("the declaration lost one of its halves:\n%s", page)
	}
	if !strings.Contains(page, "unchecked") {
		t.Fatalf("the declaration is not marked unchecked:\n%s", page)
	}
}

// The two non-contradictory shapes keep their distinct wording: "covers every
// row" and "covers some rows" license different conclusions.
func TestDestinationReadHTMLPrintsTheTwoDeclarationShapesDistinctly(t *testing.T) {
	cases := map[string]struct {
		declaration *destinationDeclaration
		want        string
	}{
		"covers the relation": {
			declaration: &destinationDeclaration{CoversRelation: "sales.order.lines"},
			want:        "covers sales.order.lines (unchecked)",
		},
		"covers some rows": {
			declaration: &destinationDeclaration{CoversSome: true},
			want:        "covers some (unchecked)",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			read := destinationRead{
				Application: "order_entry",
				Pattern:     "order_entry.order_lines",
				Relation:    "sales.order.lines",
				Coverage:    "partial",
				Rows:        []destinationRow{{Key: `["1001"]`}},
				PerRow:      map[string]map[string]string{`["1001"]`: {}},
				Declaration: tc.declaration,
			}
			var out bytes.Buffer
			if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
				t.Fatalf("code=%d err=%v", code, err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("missing %q in page", tc.want)
			}
		})
	}
}

// An undeclared binding prints no declaration line at all, which stays
// distinguishable from "declared but unchecked".
func TestDestinationReadHTMLPrintsNoDeclarationWhenUndeclared(t *testing.T) {
	read := destinationRead{
		Application: "order_entry",
		Pattern:     "order_entry.order_lines",
		Relation:    "sales.order.lines",
		Coverage:    "partial",
		Rows:        []destinationRow{{Key: `["1001"]`}},
		PerRow:      map[string]map[string]string{`["1001"]`: {}},
	}
	var out bytes.Buffer
	if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if strings.Contains(out.String(), "declaration:") {
		t.Fatalf("an undeclared binding printed a declaration:\n%s", out.String())
	}
}

// The page's answers and the read's answers are the same values. This guards
// against the page inventing an answer, which would make the browser and the
// terminal disagree about one read.
func TestDestinationReadHTMLAnswersComeFromTheRead(t *testing.T) {
	read := destinationRead{
		Application: "order_entry",
		Pattern:     "order_entry.order_lines",
		Relation:    "sales.order.lines",
		Coverage:    "partial",
		Rows:        []destinationRow{{Key: `["1001"]`}, {Key: `["1002"]`}},
		PerRow: map[string]map[string]string{
			`["1001"]`: {"sales.order_line.quantity_positive": "satisfied"},
			`["1002"]`: {"sales.order_line.quantity_positive": "not-satisfied"},
		},
		PerRelation: map[string]string{"sales.order.lines.line_number_unique": "undecided"},
		PerRelationUndecided: map[string]string{
			"sales.order.lines.line_number_unique": "no fetch coverage declared",
		},
	}
	page := destinationReadPageOf(read)

	// Every row of the read is on the page, in sorted key order.
	if len(page.Rows) != len(read.Rows) {
		t.Fatalf("page rows=%d read rows=%d", len(page.Rows), len(read.Rows))
	}
	if page.Rows[0].Key != `["1001"]` || page.Rows[1].Key != `["1002"]` {
		t.Fatalf("rows are not in key order: %v", []string{page.Rows[0].Key, page.Rows[1].Key})
	}
	for i, want := range []string{"satisfied", "not-satisfied"} {
		answers := page.Rows[i].Answers
		if len(answers) != 1 {
			t.Fatalf("row %d has %d answers", i, len(answers))
		}
		if answers[0].Answer != want {
			t.Fatalf("row %d answer=%q want %q", i, answers[0].Answer, want)
		}
	}
	if len(page.RelationAnswers) != 1 || page.RelationAnswers[0].Answer != "undecided" {
		t.Fatalf("relation answers=%+v", page.RelationAnswers)
	}
	if page.RelationAnswers[0].Reason != "no fetch coverage declared" {
		t.Fatalf("the relation answer lost its reason: %+v", page.RelationAnswers[0])
	}
	// The page holds the same read, not a copy that could drift.
	if len(page.Read.Rows) != len(read.Rows) {
		t.Fatalf("the page copied the read instead of holding it")
	}
}
