package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// The four type words the design states (section 8.3) are read at authoring
// time. Before this they were refused as unknown inputs, so a declaration that
// named them could not exist and no reader was ever asked for one.
func TestRuleAuthoringAcceptsTheFourValueTypeWords(t *testing.T) {
	for _, word := range []string{"number", "date", "text", "truth", "any", "presence"} {
		t.Run(word, func(t *testing.T) {
			src := "namespace sales\noperator sales.op: takes value: " + word + "\n"
			anchored, err := syntax.ParseRuleAuthoringAnchored(src)
			if err != nil {
				t.Fatalf("takes value: %s was refused: %v", word, err)
			}
			got := anchored.Value.Operators[0]
			if string(got.Input) != word {
				t.Fatalf("input=%q, want %q", got.Input, word)
			}
			if len(got.Inputs) != 1 || string(got.Inputs[0].Type) != word {
				t.Fatalf("inputs=%+v, want one %s input", got.Inputs, word)
			}
			mapped, ok := syntax.OperatorInputFor(syntax.OperatorInputWord(word))
			if !ok || string(mapped) != word {
				t.Fatalf("OperatorInputFor(%s)=(%q,%v), want (%q,true)", word, mapped, ok, word)
			}
		})
	}
}

// A cell states what the input's type says it states (section 8.3). The table
// reader knows the column names but not their types, so it reads the notation
// and the type check is the Workspace's; what is pinned here is that the
// notation is read and that a cell none of the types can state is refused at
// its own cell.
func TestOperatorCasesTableReadsTheCellNotation(t *testing.T) {
	src := "<!-- tei: cases of sales.op -->\n" +
		"| value | expect |\n" +
		"| --- | --- |\n" +
		"| 2026-04-01 | satisfied |\n" +
		"| \"A-100\" | not-satisfied |\n" +
		"| true | satisfied |\n" +
		"| 100 | not-satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("the cell notation was refused: %v", err)
	}
	want := []string{"2026-04-01", `"A-100"`, "true", "100"}
	if len(parsed.Cases) != len(want) {
		t.Fatalf("cases=%d, want %d", len(parsed.Cases), len(want))
	}
	for i, text := range want {
		if parsed.Cases[i].Value.Value != text {
			t.Fatalf("case %d states %q, want %q", i, parsed.Cases[i].Value.Value, text)
		}
	}
}

// The empty text and the empty string are two different cells: the first states
// that the row holds no value, the second a string of length zero. Only the
// first is written as an empty cell, and it is refused in the one-record form
// where every example states a value.
func TestAnEmptyCellIsNotTheEmptyString(t *testing.T) {
	document, err := syntax.ScanTEIMarkdownDocument(
		"<!-- tei: cases of sales.op -->\n| value | expect |\n| --- | --- |\n| \"\" | satisfied |\n")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf(`the empty string "" was refused: %v`, err)
	}
	if parsed.Cases[0].Value.Value != `""` {
		t.Fatalf("case states %q, want %q", parsed.Cases[0].Value.Value, `""`)
	}
	cell, err := semantic.ParseOperatorCaseCell(`""`)
	if err != nil {
		t.Fatalf(`"" is not a readable cell: %v`, err)
	}
	if cell.Kind != semantic.OperatorCaseCellText || cell.Text != "" {
		t.Fatalf("cell=%+v, want the empty text", cell)
	}
}

// A cell whose quotes do not close is refused rather than read as a shorter
// string, and so is a boolean written in other words.
func TestOperatorCasesTableRefusesAnUnclosedQuote(t *testing.T) {
	for _, tc := range []struct{ name, src, at, message string }{
		{"unclosed quote", "<!-- tei: cases of sales.op -->\n| value | expect |\n| --- | --- |\n| \"A-100 | satisfied |\n", `"A-100`, "not a decimal number"},
		{"a quote inside", "<!-- tei: cases of sales.op -->\n| value | expect |\n| --- | --- |\n| \"A\"100\" | satisfied |\n", `"A"100"`, "not a decimal number"},
		{"a word for a boolean", "<!-- tei: cases of sales.op -->\n| value | expect |\n| --- | --- |\n| yes | satisfied |\n", "yes", "not a decimal number"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(tc.src)
			if err != nil {
				t.Fatal(err)
			}
			_, err = syntax.ParseOperatorCasesTable(document.Tables[0])
			var parseErr *syntax.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("accepted or unlocated: %v", err)
			}
			if !strings.Contains(parseErr.Msg, tc.message) {
				t.Fatalf("message=%q, want %q", parseErr.Msg, tc.message)
			}
		})
	}
}
