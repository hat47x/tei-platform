package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A mark whose four words are `cases of <operator> expecting` and nothing else
// names the answer word but leaves the answer out. The mark's shape is right, so
// the refusal says which word is missing and shows the mark that would be
// right, rather than the shape, which needs no repair.
//
// The expectation is drawn from section 8.2's shape
// `<!-- tei: cases of <operator> expecting <answer> -->`, not from the code.
func TestAMarkWithExpectingAndNoAnswerNamesTheWord(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("a mark with `expecting` and no answer was accepted")
	}
	// The refusal is not the unknown-mark one: the shape is right, so sending
	// the author to the shape would send them where nothing needs repair.
	if strings.Contains(err.Error(), "unknown tei mark") {
		t.Fatalf("the refusal names the shape instead of the missing word: %v", err)
	}
	if !strings.Contains(err.Error(), "expecting") {
		t.Fatalf("the refusal does not name the word that is there: %v", err)
	}
	// The refusal shows the mark that would be right, so the answer is not
	// guessed at.
	if !strings.Contains(err.Error(), "sales.every_line") {
		t.Fatalf("the refusal does not name the operator it is about: %v", err)
	}
	if !strings.Contains(err.Error(), "not-satisfied") {
		t.Fatalf("the refusal does not show an answer to write: %v", err)
	}
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("the refusal is not a parse error: %v", err)
	}
	if parseErr.Pos.Line != 1 || parseErr.Pos.Column != 1 {
		t.Fatalf("the refusal is not at the mark: %#v", parseErr.Pos)
	}
}

// A mark whose answer word is missing entirely is a different shape, so it keeps
// the unknown-mark refusal rather than the answer-word one.
func TestAMarkWithNoExpectingKeepsTheUnknownMarkRefusal(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("a mark with no expecting was accepted")
	}
	if !strings.Contains(err.Error(), "unknown tei mark") {
		t.Fatalf("the refusal is not the unknown-mark one: %v", err)
	}
}

// A mark whose answer is a word the shape does not accept is refused at the
// answer's own cell, which is a different refusal from the one about a missing
// answer.
func TestAMarkWithAnAnswerInOtherWordsIsRefusedAtTheCell(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting ok -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("an answer in other words was accepted")
	}
	if !strings.Contains(err.Error(), "satisfied, not-satisfied or undecided") {
		t.Fatalf("the refusal does not name the answers the shape accepts: %v", err)
	}
}

// A mark with the answer states the row form's own shape, so the table is read.
func TestAMarkWithTheAnswerIsRead(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting not-satisfied -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("the table was refused: %v", err)
	}
	if len(parsed.Case.Rows) != 1 || parsed.Case.Rows[0]["value"] != "5" {
		t.Fatalf("rows=%v, want the one row", parsed.Case.Rows)
	}
}
