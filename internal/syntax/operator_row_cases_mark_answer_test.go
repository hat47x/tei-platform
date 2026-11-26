package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A mark whose three words are `cases of <operator>` and nothing else is the
// one-record form's own shape. Read by the row reader it is an unknown mark,
// whose message names the row form's shape and therefore the answer word the
// mark has to carry.
//
// The expectations are drawn from section 8.2's shape
// `<!-- tei: cases of <operator> expecting <answer> -->`, not from the code.
func TestAThreeWordMarkIsRefusedAsUnknownByTheRowReader(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("a three-word mark was accepted by the row reader")
	}
	if !strings.Contains(err.Error(), "unknown tei mark") {
		t.Fatalf("the refusal is not the unknown-mark one: %v", err)
	}
	// The refusal names the shape the author has to write, which includes the
	// answer word.
	if !strings.Contains(err.Error(), "expecting") {
		t.Fatalf("the refusal does not name the answer word: %v", err)
	}
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("the refusal is not a parse error: %v", err)
	}
	if parseErr.Pos.Line != 1 || parseErr.Pos.Column != 1 {
		t.Fatalf("the refusal is not at the mark: %#v", parseErr.Pos)
	}
}

// The row reader's refusal for that mark names the answer word, and the
// one-record reader's refusal for the same mark asks for its own column. They
// are different messages, and the loader's word count decides which reader the
// mark reaches, so no author is told to write a word their form does not have.
func TestAThreeWordMarkIsRefusedByEachReaderWithItsOwnMessage(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, rowErr := syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if rowErr == nil {
		t.Fatal("the row reader accepted a three-word mark")
	}
	if !strings.Contains(rowErr.Error(), "expecting") {
		t.Fatalf("the row reader's refusal does not name the answer word: %v", rowErr)
	}
	_, recordErr := syntax.ParseOperatorCasesTable(document.Tables[0])
	if recordErr == nil {
		t.Fatal("the one-record reader accepted a table with no expect column")
	}
	if !strings.Contains(recordErr.Error(), "expect") {
		t.Fatalf("the one-record reader's refusal does not name its column: %v", recordErr)
	}
}

// A mark with the answer states the row form's own shape, so the row reader
// reads it.
func TestARowSetMarkWithTheAnswerWordIsReadByTheRowReader(t *testing.T) {
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

// A mark of an entirely different shape is refused as an unknown mark too, so
// the refusal is about the shape and not about the word count alone.
func TestAMarkOfAnotherShapeIsRefusedAsUnknownByTheRowReader(t *testing.T) {
	const src = "<!-- tei: rows of sales.every_line -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("a mark of another shape was accepted")
	}
	if !strings.Contains(err.Error(), "unknown tei mark") {
		t.Fatalf("the refusal is not the unknown-mark one: %v", err)
	}
}
