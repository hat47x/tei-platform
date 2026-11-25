package syntax_test

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// A presence input reads whether the attribute holds a value, so its example
// cell states one of two words rather than a decimal. The reader accepts the
// words and carries them; the Workspace checks them against the input's type.
func TestCaseTableReadsAPresenceCellAsAWord(t *testing.T) {
	const src = "<!-- tei: cases of sales.mixed -->\n" +
		"| value | limit | expect |\n| --- | --- | --- |\n| 5 | present | satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("a presence cell was refused: %v", err)
	}
	if len(parsed.Cases) != 1 {
		t.Fatalf("cases=%d, want one", len(parsed.Cases))
	}
	if got := parsed.Cases[0].Value.Inputs["limit"]; got != semantic.OperatorCasePresencePresent {
		t.Fatalf("limit=%q, want %q", got, semantic.OperatorCasePresencePresent)
	}
}

// The two words are the only ones this reader carries. A word it does not know
// is refused at the cell, so the Workspace is not asked about text the table
// never stated.
func TestCaseTableRefusesAPresenceCellItDoesNotKnow(t *testing.T) {
	const src = "<!-- tei: cases of sales.mixed -->\n" +
		"| value | limit | expect |\n| --- | --- | --- |\n| 5 | yes | satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if _, err := syntax.ParseOperatorCasesTable(document.Tables[0]); err == nil {
		t.Fatal("an unknown presence cell was accepted")
	}
}

// A number cell keeps the decimal spelling. The reader still refuses a word
// that is not one of the two presence words, so the Workspace sees a decimal or
// nothing at all.
func TestCaseTableKeepsTheDecimalForANumberCell(t *testing.T) {
	const src = "<!-- tei: cases of sales.mixed -->\n" +
		"| value | expect |\n| --- | --- |\n| 5 | satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	if parsed.Cases[0].Value.Value != "5" {
		t.Fatalf("value=%q, want 5", parsed.Cases[0].Value.Value)
	}
}
