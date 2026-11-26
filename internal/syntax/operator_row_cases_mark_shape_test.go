package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A row-set mark states the answer in the mark, so a mark that leaves it out is
// refused at the mark rather than read as a one-row-per-example table. The
// refusal names the shape the author has to write.
//
// The expectations are drawn from section 8.2's shape
// `<!-- tei: cases of <operator> expecting <answer> -->`, not from the code.
func TestARowSetMarkWithoutExpectingIsRefusedAtTheMark(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("a row-set mark without expecting was accepted")
	}
	if !strings.Contains(err.Error(), "expecting") {
		t.Fatalf("the refusal does not name what to write: %v", err)
	}
	// The refusal points at the mark, because the missing word is in the mark,
	// not in the table's columns.
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("the refusal is not a parse error: %v", err)
	}
	if parseErr.Pos.Line != 1 || parseErr.Pos.Column != 1 {
		t.Fatalf("the refusal is not at the mark: %#v", parseErr.Pos)
	}
}

// A mark with the answer states the same shape as before, so the table is read.
func TestARowSetMarkWithExpectingIsRead(t *testing.T) {
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
