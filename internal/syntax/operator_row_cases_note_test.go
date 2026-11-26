package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

// A row-set example is about the rows together, and a note column says which
// rows those are. The one-row-per-example table already reads `note` as a note
// rather than an input (TEI_MD_RULE_READER_AND_CORE_STRUCTURE_DECISION_2026-11
// says "note は説明で、読み込まない"), and the row-set table has to say the
// same thing, or an author who writes the same column in both places is
// refused in one and accepted in the other.
func TestRowCaseTableReadsANoteColumnAsANote(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting not-satisfied -->\n" +
		"| value | limit | note |\n| --- | --- | --- |\n" +
		"| 5 | 4 | 上限超え |\n| 6 | 5 | 上限超え |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("a note column was refused: %v", err)
	}
	if len(parsed.Case.Rows) != 2 {
		t.Fatalf("rows=%d, want the two", len(parsed.Case.Rows))
	}
	for index, row := range parsed.Case.Rows {
		// The note is not a value of an input, so the row states the two
		// inputs and nothing else.
		if len(row) != 2 {
			t.Fatalf("row %d states %d values, want the two inputs: %v", index+1, len(row), row)
		}
		if row["value"] == "" || row["limit"] == "" {
			t.Fatalf("row %d lost an input value: %v", index+1, row)
		}
		if _, present := row["note"]; present {
			t.Fatalf("row %d read the note as an input: %v", index+1, row)
		}
	}
}

// A note column is not an input, so it does not have to be a decimal: it is a
// sentence about the rows.
func TestRowCaseTableANoteColumnIsNotADecimal(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting not-satisfied -->\n" +
		"| value | note |\n| --- | --- |\n| 5 | 受注明細のいずれかが上限を超える |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorRowCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("a sentence in a note column was refused: %v", err)
	}
	if len(parsed.Case.Rows) != 1 || parsed.Case.Rows[0]["value"] != "5" {
		t.Fatalf("rows=%v, want the one row", parsed.Case.Rows)
	}
}

// The expect column stays refused in a row example: the answer is in the mark,
// because it is about the rows together. A note column does not change that.
func TestRowCaseTableStillRefusesAnExpectColumn(t *testing.T) {
	const src = "<!-- tei: cases of sales.every_line expecting not-satisfied -->\n" +
		"| value | expect |\n| --- | --- |\n| 5 | satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if _, err := syntax.ParseOperatorRowCasesTable(document.Tables[0]); err == nil {
		t.Fatal("an expect column was accepted in a row example")
	}
}
