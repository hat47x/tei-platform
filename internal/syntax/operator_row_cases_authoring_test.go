package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Decision section 8.1: a row-scoped operator's example is one table whose
// columns are its input names and whose answer is in the mark. The cases below
// are the section's own acceptance table.
func TestRowExampleTableIsReadAsOneExample(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting not-satisfied -->\n" +
		"| value | limit |\n| --- | --- |\n| 2 | 10 |\n| 5 | 4 |\n"
	parsed, err := syntax.ParseOperatorRowCasesTable(mustRowTable(t, src))
	if err != nil {
		t.Fatalf("the row example was refused: %v", err)
	}
	if parsed.Operator != "sales.every_line" {
		t.Fatalf("operator=%q", parsed.Operator)
	}
	if parsed.Case.Answer != semantic.OperatorRowNotSatisfied {
		t.Fatalf("answer=%q, want not-satisfied", parsed.Case.Answer)
	}
	if len(parsed.Case.Rows) != 2 {
		t.Fatalf("rows=%d, want the two rows", len(parsed.Case.Rows))
	}
	if parsed.Case.Rows[1]["value"] != "5" || parsed.Case.Rows[1]["limit"] != "4" {
		t.Fatalf("row 1=%v", parsed.Case.Rows[1])
	}
}

// The column order does not matter: the values are matched to their names, so
// a reversed table states the same example.
func TestRowExampleTableKeepsValuesWithTheirNames(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting satisfied -->\n" +
		"| limit | value |\n| --- | --- |\n| 10 | 2 |\n"
	parsed, err := syntax.ParseOperatorRowCasesTable(mustRowTable(t, src))
	if err != nil {
		t.Fatalf("the row example was refused: %v", err)
	}
	if parsed.Case.Rows[0]["value"] != "2" || parsed.Case.Rows[0]["limit"] != "10" {
		t.Fatalf("row 0=%v", parsed.Case.Rows[0])
	}
}

// A table with an expect column is the one-record form, so it is refused here
// rather than read as a row example.
func TestRowExampleTableRefusesAnExpectColumn(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting satisfied -->\n" +
		"| value | limit | expect |\n| --- | --- | --- |\n| 2 | 10 | satisfied |\n"
	_, err := syntax.ParseOperatorRowCasesTable(mustRowTable(t, src))
	if err == nil || !strings.Contains(err.Error(), "no expect column") {
		t.Fatalf("a table with an expect column was accepted: %v", err)
	}
}

// An example may state that its rows cannot be compared: the answer is
// `undecided` and one of the rows holds no value or was not read
// (decision section 8.1's table of three-valued answers).
func TestRowExampleTableReadsAnUndecidedAnswer(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting undecided -->\n" +
		"| value | limit |\n| --- | --- |\n|  | 10 |\n"
	parsed, err := syntax.ParseOperatorRowCasesTable(mustRowTable(t, src))
	if err != nil {
		t.Fatalf("an undecided answer was refused: %v", err)
	}
	if parsed.Case.Answer != semantic.OperatorRowUndecided {
		t.Fatalf("answer=%q, want undecided", parsed.Case.Answer)
	}
	if len(parsed.Case.Rows) != 1 {
		t.Fatalf("rows=%v, want one row", parsed.Case.Rows)
	}
	// The empty cell states that the row holds no value; it is not zero, and
	// it is not a missing column.
	if value, stated := parsed.Case.Rows[0]["value"]; !stated || value != semantic.RowCellNoValue {
		t.Fatalf("value cell=%q stated=%v, want the empty text", value, stated)
	}
}

// An unread row is written as the word `not-fetched`, so it is never
// confused with a row that holds no value.
func TestRowExampleTableReadsANotFetchedRow(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting undecided -->\n" +
		"| value | limit |\n| --- | --- |\n| not-fetched | 10 |\n"
	parsed, err := syntax.ParseOperatorRowCasesTable(mustRowTable(t, src))
	if err != nil {
		t.Fatalf("a not-fetched row was refused: %v", err)
	}
	if value := parsed.Case.Rows[0]["value"]; value != semantic.RowCellNotFetched {
		t.Fatalf("value cell=%q, want the word not-fetched", value)
	}
}

// An answer that is none of the three is still refused, and the cell says
// which answers exist.
func TestRowExampleTableRefusesAnUnknownAnswer(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting maybe -->\n" +
		"| value | limit |\n| --- | --- |\n| 2 | 10 |\n"
	_, err := syntax.ParseOperatorRowCasesTable(mustRowTable(t, src))
	if err == nil || !strings.Contains(err.Error(), "satisfied, not-satisfied or undecided") {
		t.Fatalf("an unknown answer was accepted: %v", err)
	}
}

// A cell that is not a decimal is refused at its own cell.
func TestRowExampleTableRefusesANonDecimalCell(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting satisfied -->\n" +
		"| value | limit |\n| --- | --- |\n| 1e3 | 10 |\n"
	_, err := syntax.ParseOperatorRowCasesTable(mustRowTable(t, src))
	if err == nil || !strings.Contains(err.Error(), "row 1") {
		t.Fatalf("a non-decimal cell was accepted or unlocated: %v", err)
	}
}

func mustRowTable(t *testing.T, src string) syntax.MarkdownTable {
	t.Helper()
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scanning the document failed: %v", err)
	}
	if len(document.Tables) != 1 {
		t.Fatalf("the document has %d tables, want 1", len(document.Tables))
	}
	return document.Tables[0]
}
