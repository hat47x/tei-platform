package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// A row-set example is one table. The mark states the expected answer once,
// the columns are the operator's input names, and the body rows are the rows of
// one relation. These cases read the table alone: the signature is checked
// where the whole Workspace is known.

const rowSetOperatorDocument = "# 明細の上限\n\nすべての行が上限以下であることを確かめる。\n\n" +
	"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
	"| value | limit |\n" +
	"| --- | --- |\n" +
	"| 2 | 10 |\n" +
	"| 5 | 5 |\n\n" +
	"<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->\n" +
	"| value | limit |\n" +
	"| --- | --- |\n" +
	"| 2 | 10 |\n" +
	"| 5 | 4 |\n"

// TestRowSetCasesReadOneTableAsOneExample reads the two tables above and
// expects each to become one example holding the rows as written.
func TestRowSetCasesReadOneTableAsOneExample(t *testing.T) {
	document, err := syntax.ScanTEIMarkdownDocument(rowSetOperatorDocument)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(document.Tables) != 2 {
		t.Fatalf("tables=%d, want the two tables", len(document.Tables))
	}
	satisfied, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("the satisfied table was refused: %v", err)
	}
	if satisfied.Operator != "sales.every_line_at_most" {
		t.Fatalf("operator=%s, want sales.every_line_at_most", satisfied.Operator)
	}
	if len(satisfied.Cases) != 1 {
		t.Fatalf("cases=%d, want one example per table", len(satisfied.Cases))
	}
	example := satisfied.Cases[0].Value
	if !example.IsRowSetExample() {
		t.Fatal("the example was not read as a row set")
	}
	if example.RowExpect != semantic.RowExpectSatisfied {
		t.Fatalf("rowExpect=%q, want satisfied", example.RowExpect)
	}
	// The rows are the body rows, in the order written, with the values as
	// written.
	if len(example.Rows) != 2 {
		t.Fatalf("rows=%d, want 2", len(example.Rows))
	}
	if example.Rows[0]["value"] != "2" || example.Rows[0]["limit"] != "10" {
		t.Fatalf("first row=%v", example.Rows[0])
	}
	if example.Rows[1]["value"] != "5" || example.Rows[1]["limit"] != "5" {
		t.Fatalf("second row=%v", example.Rows[1])
	}
	// A row-set example states neither a single value nor one value per input.
	if example.Value != "" || example.Inputs != nil {
		t.Fatalf("the row set also carries a one-record value: value=%q inputs=%v", example.Value, example.Inputs)
	}
	// The bool is not an expectation for a row set: it cannot tell "expected
	// not-satisfied" from "stated no expectation".
	if example.Satisfied {
		t.Fatal("a row-set example reported satisfied=true without stating it")
	}

	notSatisfied, err := syntax.ParseOperatorCasesTable(document.Tables[1])
	if err != nil {
		t.Fatalf("the not-satisfied table was refused: %v", err)
	}
	if notSatisfied.Cases[0].Value.RowExpect != semantic.RowExpectNotSatisfied {
		t.Fatalf("rowExpect=%q, want not-satisfied", notSatisfied.Cases[0].Value.RowExpect)
	}
	if notSatisfied.Cases[0].Value.Rows[1]["limit"] != "4" {
		t.Fatalf("second row=%v", notSatisfied.Cases[0].Value.Rows[1])
	}
}

// A one-value example keeps the shape it had, so a reader that knows nothing
// about row sets reads it exactly as before.
func TestOneValueCasesKeepTheirShape(t *testing.T) {
	document, err := syntax.ScanTEIMarkdownDocument(operatorCasesDocument)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	for _, example := range parsed.Cases {
		if example.Value.IsRowSetExample() || example.Value.RowExpect != "" {
			t.Fatalf("a one-value example was read as a row set: %+v", example.Value)
		}
	}
}

func TestRowSetCasesRefusals(t *testing.T) {
	table := func(mark, header, rows string) string {
		cells := strings.Count(header, "|") - 1
		delimiter := "|" + strings.Repeat(" --- |", cells)
		return mark + "\n" + header + "\n" + delimiter + "\n" + rows + "\n"
	}
	const mark = "<!-- tei: cases of sales.every_line_at_most expecting satisfied -->"
	const header = "| value | limit |"
	cases := []struct {
		name, src, at, message string
	}{
		{"no answer in the mark",
			table("<!-- tei: cases of sales.every_line_at_most -->", header, "| 2 | 10 |"),
			"<!-- tei:", "needs the column \"expect\""},
		{"an answer this reader does not know",
			table("<!-- tei: cases of sales.every_line_at_most expecting undecided -->", header, "| 2 | 10 |"),
			"undecided", "satisfied or not-satisfied"},
		{"an expect column beside the answer",
			table(mark, "| value | limit | expect |", "| 2 | 10 | satisfied |"),
			"| expect |", "states its answer once in the mark"},
		{"a note column beside the answer",
			table(mark, "| value | limit | note |", "| 2 | 10 | x |"),
			"| note |", "states its answer once in the mark"},
		{"a blank cell",
			table(mark, header, "| 2 |  |"),
			"| 2 |  |", "is empty"},
		{"a value that is not a number",
			table(mark, header, "| 2 | none |"),
			"none", "not a decimal number"},
		{"a column written twice",
			table(mark, "| value | value |", "| 2 | 3 |"),
			"value | value |", "more than once"},
		{"an empty header cell",
			table(mark, "| value |  |", "| 2 | 3 |"),
			"| value |  |", "header is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(tc.src)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			_, err = syntax.ParseOperatorCasesTable(document.Tables[0])
			var parseErr *syntax.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("accepted or unlocated: %v", err)
			}
			if tc.message != "" && !strings.Contains(parseErr.Msg, tc.message) {
				t.Fatalf("message=%q, want %q", parseErr.Msg, tc.message)
			}
			want := uniqueOffset(t, tc.src, tc.at)
			if parseErr.Pos.Offset < want || parseErr.Pos.Offset > want+len(tc.at) {
				t.Fatalf("error offset=%d, want within %q at %d", parseErr.Pos.Offset, tc.at, want)
			}
		})
	}
}

// Two rows that hold the same numbers are two rows, not one repeated example.
// The order of the rows is the order the author wrote and is kept.
func TestRowSetCasesKeepRepeatedAndReorderedRows(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->\n" +
		"| value | limit |\n| --- | --- |\n| 5 | 4 |\n| 5 | 4 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	if len(parsed.Cases) != 1 {
		t.Fatalf("cases=%d, want one table", len(parsed.Cases))
	}
	if len(parsed.Cases[0].Value.Rows) != 2 {
		t.Fatalf("rows=%d, want the two written rows", len(parsed.Cases[0].Value.Rows))
	}
}

// The column order does not matter: the values are attached by the column
// name, so a table that writes limit first states the same rows.
func TestRowSetCasesReadColumnOrderByName(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
		"| limit | value |\n| --- | --- |\n| 10 | 2 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	row := parsed.Cases[0].Value.Rows[0]
	if row["value"] != "2" || row["limit"] != "10" {
		t.Fatalf("row=%v, want value=2 limit=10 regardless of column order", row)
	}
}
