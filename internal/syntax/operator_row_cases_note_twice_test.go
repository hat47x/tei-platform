package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A note column states what the rows show, not what an input holds, so two of
// them are the same word written twice. The one-record reader refuses a
// repeated column, and a row example's columns are read the same way, so the
// refusal has to hold here too: a table whose columns are read has one column
// per name.
//
// The expectation is drawn from section 8.1's treatment of `note` as a word
// that is not read, not from the code.
func TestRowCaseTableRefusesANoteColumnWrittenTwice(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting not-satisfied -->\n" +
		"| value | limit | note | note |\n| --- | --- | --- | --- |\n" +
		"| 5 | 4 | 上限超え | 上限超え |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("a note column written twice was accepted")
	}
	if !strings.Contains(err.Error(), "appears more than once") {
		t.Fatalf("the refusal is not the repeated-column one: %v", err)
	}
	// The refusal points at the second note cell, which is the one to remove.
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("the refusal is not a parse error: %v", err)
	}
	if parseErr.Pos.Line != 2 {
		t.Fatalf("the refusal is not in the header: %#v", parseErr.Pos)
	}
	if parseErr.Pos.Column < 20 {
		t.Fatalf("the refusal is not at the second note cell: %#v", parseErr.Pos)
	}
}

// One note column is still a note: the rows state their inputs and the note is
// not one of them.
func TestRowCaseTableKeepsOneNoteColumn(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting not-satisfied -->\n" +
		"| value | limit | note |\n| --- | --- | --- |\n" +
		"| 5 | 4 | 上限超え |\n| 6 | 5 | 上限超え |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("the table was refused: %v", err)
	}
	if len(parsed.Case.Rows) != 2 {
		t.Fatalf("rows=%d, want the two", len(parsed.Case.Rows))
	}
	for index, row := range parsed.Case.Rows {
		if len(row) != 2 {
			t.Fatalf("row %d states %d values, want the two inputs: %v", index+1, len(row), row)
		}
	}
}
