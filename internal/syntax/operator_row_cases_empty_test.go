package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Section 8.2: the empty relation is one example of a row-scoped operator. It
// is written as the header and the delimiter of a row-set table whose mark says
// `over no rows`. A row-set table without that phrase still needs a body row, so
// a table an author left unwritten is reported rather than read as an empty one.

func TestRowSetTableStatesTheEmptyRelation(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line_at_most expecting satisfied over no rows -->\n" +
		"| value | limit |\n| --- | --- |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("the empty row-set table was refused: %v", err)
	}
	if len(document.Tables) != 1 {
		t.Fatalf("tables=%d, want the one table", len(document.Tables))
	}
	parsed, err := syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("the empty example was refused: %v", err)
	}
	// The rows are an empty collection, not a missing one: the example was
	// stated and it says the relation has no rows.
	if parsed.Case.Rows == nil {
		t.Fatal("the example lost its rows: they are nil, not empty")
	}
	if len(parsed.Case.Rows) != 0 {
		t.Fatalf("rows=%v, want none", parsed.Case.Rows)
	}
	if parsed.Case.Answer != semantic.OperatorRowSatisfied {
		t.Fatalf("answer=%q, want satisfied", parsed.Case.Answer)
	}
	// The columns are still checked, so a header that names no declared input
	// is reported even when there are no rows to read.
	if len(parsed.InputHeaderSpans) != 2 {
		t.Fatalf("columns=%v, want the two named columns", parsed.InputHeaderSpans)
	}
}

// An ordinary table is unchanged: with no body row it is still reported, so a
// table left unwritten is not silently read as an empty collection.
func TestAnOrdinaryTableStillNeedsABodyRow(t *testing.T) {
	const src = "<!-- tei: cases of sales.non_negative -->\n| value | expect |\n| --- | --- |\n"
	_, err := syntax.ScanTEIMarkdownDocument(src)
	if err == nil || !strings.Contains(err.Error(), "at least one row") {
		t.Fatalf("an ordinary table with no body row was accepted: %v", err)
	}
}

// Section 8.2: the phrase is what makes an empty body mean the empty relation.
// Without it, the same table is reported and the report names the phrase, so a
// row forgotten by the author is not silently promoted to an example.
func TestAnEmptyRowSetTableWithoutThePhraseIsRefused(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
		"| value | limit |\n| --- | --- |\n"
	_, err := syntax.ScanTEIMarkdownDocument(src)
	if err == nil {
		t.Fatal("an empty row-set table without the phrase was read as the empty relation")
	}
	if !strings.Contains(err.Error(), "over no rows") {
		t.Fatalf("the refusal does not name the phrase: %v", err)
	}
}

// A row-set mark with an unknown answer is still refused before the table is
// read, so an empty example cannot smuggle in an answer this reader does not
// know.
func TestEmptyRowSetTableRefusesAnUnknownAnswer(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line_at_most expecting undecided over no rows -->\n" +
		"| value | limit |\n| --- | --- |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	// `undecided` is a known answer; this case states that the reader accepts
	// it. The unknown-answer refusal is covered where it belongs, so this
	// table is read as an example that answers undecided.
	parsed, err := syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("an undecided empty example was refused: %v", err)
	}
	if parsed.Case.Answer != semantic.OperatorRowUndecided {
		t.Fatalf("answer=%q, want undecided", parsed.Case.Answer)
	}
}
