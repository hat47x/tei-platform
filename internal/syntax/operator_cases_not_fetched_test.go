package syntax_test

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Section 8.3 says the writing for "not read" does not depend on the input's
// type, so a one-record example's cell may write the same word the row-set form
// writes. It is not a decimal, so the reader carries it as written and the
// Workspace reads it against the input's type.
//
// The expectation is drawn from that section's table, not from the code.
func TestCaseTableSingleValueFormCarriesANotFetchedCell(t *testing.T) {
	const src = "<!-- tei: cases of sales.non_negative -->\n" +
		"| value | expect |\n| --- | --- |\n| not-fetched | not-satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("a not-fetched cell was refused: %v", err)
	}
	if len(parsed.Cases) != 1 {
		t.Fatalf("cases=%d, want one", len(parsed.Cases))
	}
	if got := parsed.Cases[0].Value.Inputs["value"]; got != semantic.RowCellNotFetched {
		t.Fatalf("value=%q, want the word %q", got, semantic.RowCellNotFetched)
	}
}

// The multi-input form carries the word the same way, so a table that names its
// inputs can state that one of them was not read.
func TestCaseTableMultiInputFormCarriesANotFetchedCell(t *testing.T) {
	const src = "<!-- tei: cases of sales.mixed -->\n" +
		"| value | limit | expect |\n| --- | --- | --- |\n| 5 | not-fetched | not-satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("a not-fetched cell was refused: %v", err)
	}
	if got := parsed.Cases[0].Value.Inputs["limit"]; got != semantic.RowCellNotFetched {
		t.Fatalf("limit=%q, want the word %q", got, semantic.RowCellNotFetched)
	}
	if got := parsed.Cases[0].Value.Inputs["value"]; got != "5" {
		t.Fatalf("value=%q, want 5: the other inputs keep their decimal", got)
	}
}

// An empty cell is still refused: section 8.3 writes "not read" with the word,
// and a cell nobody wrote is not a value anybody stated.
func TestCaseTableStillRefusesAnEmptyCell(t *testing.T) {
	const src = "<!-- tei: cases of sales.mixed -->\n" +
		"| value | limit | expect |\n| --- | --- | --- |\n| 5 |  | undecided |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if _, err := syntax.ParseOperatorCasesTable(document.Tables[0]); err == nil {
		t.Fatal("an empty cell was accepted")
	}
}
