package syntax_test

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Decision section 8.1 sends the empty relation to this change: "the empty
// collection is not read as one NULL row". An example of no rows is written as
// a table with a header and no body row, and its answer says what the operator
// states about no rows at all.
//
// The expectations below are written by hand: the table states no row, the
// answer is in the mark, and the two answers differ.
func TestARowExampleStatesAnEmptyRelation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		src    string
		answer semantic.OperatorRowExpect
	}{
		{
			name: "no rows, the rule holds",
			src: "<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
				"| value | limit |\n| --- | --- |\n\nprose after\n",
			answer: semantic.OperatorRowSatisfied,
		},
		{
			name: "no rows, the rule does not hold",
			src: "<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->\n" +
				"| value | limit |\n| --- | --- |\n\nprose after\n",
			answer: semantic.OperatorRowNotSatisfied,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(tc.src)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			if len(document.Tables) != 1 {
				t.Fatalf("tables=%d, want one", len(document.Tables))
			}
			parsed, err := syntax.ParseOperatorRowCasesTable(document.Tables[0])
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !parsed.Case.Empty {
				t.Fatal("the example does not state that it is the empty relation")
			}
			if len(parsed.Case.Rows) != 0 {
				t.Fatalf("rows=%v, want none", parsed.Case.Rows)
			}
			if parsed.Case.Answer != tc.answer {
				t.Fatalf("answer=%q, want %q", parsed.Case.Answer, tc.answer)
			}
		})
	}
}

// An empty relation and a relation with one row that holds no value are two
// examples, not one written twice.
func TestAnEmptyRelationIsNotOneRowWithoutAValue(t *testing.T) {
	empty := semantic.OperatorRowCase{Answer: semantic.OperatorRowSatisfied, Empty: true}
	nullRow := semantic.OperatorRowCase{
		Answer: semantic.OperatorRowUndecided,
		Rows:   []map[string]string{{"value": semantic.RowCellNoValue, "limit": "10"}},
	}
	if semantic.SameOperatorRowCaseStates(empty, nullRow) {
		t.Fatal("an empty relation and a row that holds no value are the same example")
	}
	other := semantic.OperatorRowCase{Answer: semantic.OperatorRowNotSatisfied, Empty: true}
	if !semantic.SameOperatorRowCaseStates(empty, other) {
		t.Fatal("two empty relations are different examples")
	}
}
