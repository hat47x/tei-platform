package syntax_test

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// A presence input reads whether the attribute holds a value, so its example
// cell states one of two words rather than a decimal. The single-value form
// carries the word under the input's own name, and the Workspace reads it
// against the input's type.
func TestCaseTableSingleValueFormCarriesAPresenceCell(t *testing.T) {
	const src = "<!-- tei: cases of sales.item_present -->\n" +
		"| value | expect |\n| --- | --- |\n| present | satisfied |\n"
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
	// The word is carried under the single-value form's own column name, so the
	// Workspace reads it as that input's type rather than as the one value.
	if got := parsed.Cases[0].Value.Inputs["value"]; got != semantic.OperatorCasePresencePresent {
		t.Fatalf("value=%q, want the word %q", got, semantic.OperatorCasePresencePresent)
	}
	if parsed.Cases[0].Value.Value != "" {
		t.Fatalf("value=%q, want empty: a presence cell is not a decimal", parsed.Cases[0].Value.Value)
	}
}

// A number input keeps the decimal spelling, so the single-value form keeps
// its one-value shape for it.
func TestCaseTableSingleValueFormKeepsTheDecimal(t *testing.T) {
	const src = "<!-- tei: cases of sales.non_negative -->\n" +
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
	if parsed.Cases[0].Value.Inputs != nil {
		t.Fatalf("inputs=%v, want none", parsed.Cases[0].Value.Inputs)
	}
}
