package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Section 8.2 decided the notation for an empty relation: the row-set example's
// mark carries the clause `over no rows`, and the table has the header and the
// delimiter with no body row. A table whose body the author left unwritten has
// no clause, so it is still reported rather than silently promoted to an
// example.

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

// The section 8.2 reason: a table the author left unwritten has no clause, so
// it is reported and the refusal names the clause that states an empty relation.
func TestAnOrdinaryTableStillNeedsABodyRow(t *testing.T) {
	const src = "<!-- tei: cases of sales.non_negative -->\n| value | expect |\n| --- | --- |\n"
	_, err := syntax.ScanTEIMarkdownDocument(src)
	if err == nil || !strings.Contains(err.Error(), "at least one row") {
		t.Fatalf("an ordinary table with no body row was accepted: %v", err)
	}
	if !strings.Contains(err.Error(), "over no rows") {
		t.Fatalf("the refusal does not name the clause that states an empty relation: %v", err)
	}
}

// A row-set example without the clause is the same case: the mark does not say
// the empty collection was meant, so the table keeps needing a body row.
func TestARowSetExampleWithoutTheClauseStillNeedsABodyRow(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
		"| value | limit |\n| --- | --- |\n"
	_, err := syntax.ScanTEIMarkdownDocument(src)
	if err == nil || !strings.Contains(err.Error(), "over no rows") {
		t.Fatalf("a clause-less empty row-set table was accepted: %v", err)
	}
}

// The clause and the body have to agree: rows under `over no rows` would be
// read as a collection the mark says is empty.
func TestTheClauseIsRefusedWhenTheTableStatesRows(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line_at_most expecting satisfied over no rows -->\n" +
		"| value | limit |\n| --- | --- |\n| 2 | 10 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan refused the table before the clause was read: %v", err)
	}
	_, err = syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err == nil || !strings.Contains(err.Error(), "the mark says `over no rows`, but the table states rows") {
		t.Fatalf("rows under `over no rows` were accepted: %v", err)
	}
}

// The answer is still read before the table, including in the empty form, so a
// clause cannot change what answers this reader knows.
func TestEmptyRowSetTableReadsTheAnswersItKnows(t *testing.T) {
	for _, tc := range []struct {
		word string
		want semantic.OperatorRowExpect
	}{
		{"satisfied", semantic.OperatorRowSatisfied},
		{"not-satisfied", semantic.OperatorRowNotSatisfied},
		{"undecided", semantic.OperatorRowUndecided},
	} {
		t.Run(tc.word, func(t *testing.T) {
			src := "<!-- tei: cases of sales.every_line_at_most expecting " + tc.word + " over no rows -->\n" +
				"| value | limit |\n| --- | --- |\n"
			document, err := syntax.ScanTEIMarkdownDocument(src)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			parsed, err := syntax.ParseOperatorRowCasesTable(document.Tables[0])
			if err != nil {
				t.Fatalf("%s was refused: %v", tc.word, err)
			}
			if parsed.Case.Answer != tc.want {
				t.Fatalf("answer=%q, want %q", parsed.Case.Answer, tc.want)
			}
		})
	}
	t.Run("unknown word", func(t *testing.T) {
		src := "<!-- tei: cases of sales.every_line_at_most expecting maybe over no rows -->\n" +
			"| value | limit |\n| --- | --- |\n"
		document, err := syntax.ScanTEIMarkdownDocument(src)
		if err != nil {
			t.Fatalf("scan: %v", err)
		}
		if _, err := syntax.ParseOperatorRowCasesTable(document.Tables[0]); err == nil {
			t.Fatal("an unknown answer was accepted")
		}
	})
}
