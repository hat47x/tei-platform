package workspace

import (
	"strings"
	"testing"
)

// A row example of an operator whose input reads presence states the row's
// state with one of three words (decision section 8.6): `present`, `absent`, or
// `not-fetched`. This file pins that the product Load reads such a table, that
// the three words are the only cell writing, and that a column which is not a
// presence input cannot borrow the words.

const presenceRowExampleDocument = "semantic/sales/zz-presence-rows.tei.md"

const presenceRowOperator = "```tei\nnamespace sales\n\n" +
	"operator sales.every_price_stated:\n" +
	"  takes value: presence of each row\n" +
	"  refutes on some rows\n\n" +
	"rule sales.order.lines.every_price_stated:\n" +
	"  sales.order_line.unit_price is sales.every_price_stated per sales.order.lines\n```\n"

// presenceRowExampleWorkspace builds the fixture with one row-scoped presence
// operator, the rule that asks it, and the example table this test reads.
func presenceRowExampleWorkspace(t *testing.T, table string) string {
	t.Helper()
	root := copyTEIMDRuleWorkspace(t)
	rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.authoring/v1")
	writeTEIMDTestFile(t, root, presenceRowExampleDocument,
		"# 明細の単価の有無\n\n"+presenceRowOperator+"\n"+table)
	return root
}

// The three words are read as the three states, and the table is attached to
// the operator the mark names.
func TestTEIMDReadsAPresenceRowExampleWrittenWithThreeWords(t *testing.T) {
	root := presenceRowExampleWorkspace(t,
		"<!-- tei: cases of sales.every_price_stated expecting not-satisfied -->\n"+
			"| value |\n| --- |\n| present |\n| absent |\n")
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("the Workspace carrying the presence row example did not load: %v", err)
	}
	var found bool
	for _, namespace := range loaded.World.Semantics {
		for _, operator := range namespace.Operators {
			if operator.ID != "sales.every_price_stated" {
				continue
			}
			found = true
			if len(operator.RowCases) != 1 {
				t.Fatalf("row cases=%d, want 1", len(operator.RowCases))
			}
			example := operator.RowCases[0]
			if len(example.Rows) != 2 {
				t.Fatalf("rows=%d, want 2", len(example.Rows))
			}
			want := []string{"present", "absent"}
			for i, row := range example.Rows {
				if got := row["value"]; got != want[i] {
					t.Fatalf("row %d value=%q, want %q", i+1, got, want[i])
				}
			}
			if example.Answer != "not-satisfied" {
				t.Fatalf("answer=%q, want not-satisfied", example.Answer)
			}
		}
	}
	if !found {
		t.Fatal("the operator carrying the row example was not loaded")
	}
}

// `not-fetched` is one of the three words: it states a row nobody read, which
// a row example can say while a one-record example cannot.
func TestTEIMDReadsANotFetchedPresenceRow(t *testing.T) {
	root := presenceRowExampleWorkspace(t,
		"<!-- tei: cases of sales.every_price_stated expecting undecided -->\n"+
			"| value |\n| --- |\n| not-fetched |\n")
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("a not-fetched presence row was refused: %v", err)
	}
	var found bool
	for _, namespace := range loaded.World.Semantics {
		for _, operator := range namespace.Operators {
			if operator.ID != "sales.every_price_stated" {
				continue
			}
			found = true
			if len(operator.RowCases) != 1 || len(operator.RowCases[0].Rows) != 1 {
				t.Fatalf("row cases=%+v, want one row", operator.RowCases)
			}
			if got := operator.RowCases[0].Rows[0]["value"]; got != "not-fetched" {
				t.Fatalf("cell=%q, want not-fetched", got)
			}
		}
	}
	if !found {
		t.Fatal("the operator carrying the not-fetched row was not loaded")
	}
}

// A cell that is not one of the three words is refused where the signature is
// known, so a number or a quoted text is not read as a presence state.
func TestTEIMDRefusesAPresenceRowCellThatIsNotOneOfTheThreeWords(t *testing.T) {
	for _, cell := range []string{"1", `"present"`} {
		t.Run(cell, func(t *testing.T) {
			root := presenceRowExampleWorkspace(t,
				"<!-- tei: cases of sales.every_price_stated expecting satisfied -->\n"+
					"| value |\n| --- |\n| "+cell+" |\n")
			_, err := Load(root)
			if err == nil {
				t.Fatalf("%q was read as a presence row cell", cell)
			}
			if !strings.Contains(err.Error(), "presence") {
				t.Fatalf("the refusal does not name the presence input: %v", err)
			}
		})
	}
}

// A column that is not a presence input cannot state `present`: the word says
// whether a value is there, and a value column states which value it holds.
func TestTEIMDRefusesAPresenceWordInAValueColumn(t *testing.T) {
	root := copyTEIMDRuleWorkspace(t)
	rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.authoring/v1")
	writeTEIMDTestFile(t, root, presenceRowExampleDocument,
		"# 明細の数量\n\n"+
			"```tei\nnamespace sales\n\n"+
			"operator sales.every_quantity:\n"+
			"  takes value: number of each row\n\n"+
			"rule sales.order.lines.every_quantity_5:\n"+
			"  sales.order_line.quantity is sales.every_quantity per sales.order.lines\n```\n\n"+
			"<!-- tei: cases of sales.every_quantity expecting satisfied -->\n"+
			"| value |\n| --- |\n| present |\n")
	_, err := Load(root)
	if err == nil {
		t.Fatal("a presence word in a number column was accepted")
	}
	if !strings.Contains(err.Error(), "present") {
		t.Fatalf("the refusal does not name the word: %v", err)
	}
}
