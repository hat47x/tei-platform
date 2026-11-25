package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Two rows that state the same presence cell are one example written twice, so
// the reader refuses them the way it refuses the same decimal twice.
func TestCaseTableRefusesTheSamePresenceExampleTwice(t *testing.T) {
	const src = "<!-- tei: cases of sales.item_present -->\n" +
		"| value | expect |\n| --- | --- |\n| present | satisfied |\n| present | not-satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("the same presence example was accepted twice")
	}
	// The refusal points at the cell the author wrote the second time, not at
	// the start of the document.
	if !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("unexpected refusal: %v", err)
	}
	var parseErr *syntax.ParseError
	if errors.As(err, &parseErr) && parseErr.Pos.Line == 0 {
		t.Fatalf("the repeated example has no position: %#v", parseErr.Pos)
	}
}

// The two words are different examples, so a table that states one of each is
// not a table that repeats itself.
func TestCaseTableAcceptsBothPresenceWordsAsDifferentExamples(t *testing.T) {
	const src = "<!-- tei: cases of sales.item_present -->\n" +
		"| value | expect |\n| --- | --- |\n| present | satisfied |\n| absent | not-satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("the two words were refused: %v", err)
	}
	if len(parsed.Cases) != 2 {
		t.Fatalf("cases=%d, want the two", len(parsed.Cases))
	}
	if got := parsed.Cases[1].Value.Inputs["value"]; got != semantic.OperatorCasePresenceAbsent {
		t.Fatalf("value=%q, want %q", got, semantic.OperatorCasePresenceAbsent)
	}
}

// A decimal and a presence word are different examples too: one is not the
// same example as the other, however the operator declares its input.
func TestCaseTableKeepsADecimalAndAWordApart(t *testing.T) {
	left := semantic.OperatorCase{Value: "5"}
	right := semantic.OperatorCase{Inputs: map[string]string{"value": semantic.OperatorCasePresencePresent}}
	if semantic.SameOperatorCaseStates(left, right) {
		t.Fatal("a decimal and a presence word were read as the same example")
	}
}
