package workspace

import (
	"strings"
	"testing"

	"tei/internal/semantic"
)

// A row-set example is one table of rows with one answer in the mark. These
// cases read the table through the product Load, where the operator's signature
// is known, so a column that is not a declared input and a declared input with
// no column are both refused at the table.

const rowSetCasesDocument = "semantic/sales/z-row-cases.tei.md"

// rowSetCasesWorkspace declares one row-scoped operator whose two inputs are
// columns of the rows, and states two row-set examples of it.
func rowSetCasesWorkspace(t *testing.T) string {
	t.Helper()
	root := copyTEIMDRuleWorkspace(t)
	rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.authoring/v1")
	rewriteTEIMDTestFile(t, root, "semantic/sales/order.tei",
		"    attribute itemName type std.text\n",
		"    attribute itemName type std.text\n    attribute discount type std.decimal\n")
	writeTEIMDTestFile(t, root, rowSetCasesDocument,
		"# 行の上限\n\nすべての行が自分の上限以下であることを確かめる。\n\n"+
			"```tei\nnamespace sales\n\n"+
			"operator sales.every_line_at_most:\n"+
			"  takes value: number of each row\n"+
			"  takes limit: number of each row\n"+
			"  refutes on some rows\n```\n\n"+
			"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n"+
			"| value | limit |\n| --- | --- |\n| 2 | 10 |\n| 5 | 5 |\n\n"+
			"<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->\n"+
			"| value | limit |\n| --- | --- |\n| 2 | 10 |\n| 5 | 4 |\n")
	return root
}

// The two tables are two examples of the same operator, kept in the order they
// were written, with the rows of each.
func TestTEIMDRowSetExamplesAreAttachedAsSeparateExamples(t *testing.T) {
	loaded, err := Load(rowSetCasesWorkspace(t))
	if err != nil {
		t.Fatalf("the row-set Workspace did not load: %v", err)
	}
	var found bool
	for _, operator := range operatorsOf(loaded) {
		if operator.ID != "sales.every_line_at_most" {
			continue
		}
		found = true
		if len(operator.Cases) != 2 {
			t.Fatalf("cases=%d, want the two tables", len(operator.Cases))
		}
		if operator.Cases[0].RowExpect != semantic.RowExpectSatisfied ||
			operator.Cases[1].RowExpect != semantic.RowExpectNotSatisfied {
			t.Fatalf("the answers were not kept in the order written: %q then %q",
				operator.Cases[0].RowExpect, operator.Cases[1].RowExpect)
		}
		if len(operator.Cases[0].Rows) != 2 || len(operator.Cases[1].Rows) != 2 {
			t.Fatalf("the rows were not kept: %d and %d", len(operator.Cases[0].Rows), len(operator.Cases[1].Rows))
		}
		// The counterexample is the second row of the second table, so a reader
		// that dropped the rows would keep an example nobody can check.
		if operator.Cases[1].Rows[1]["limit"] != "4" {
			t.Fatalf("second table's second row=%v", operator.Cases[1].Rows[1])
		}
	}
	if !found {
		t.Fatal("the operator was not declared")
	}
}

// A column that names no declared input, and a declared input with no column,
// are refused at the table: an example that cannot be read is not attached as
// one the implementation would be checked against.
func TestTEIMDRowSetExamplesRefuseAColumnThatIsNotADeclaredInput(t *testing.T) {
	root := rowSetCasesWorkspace(t)
	writeTEIMDTestFile(t, root, rowSetCasesDocument,
		"# 行の上限\n\n```tei\nnamespace sales\n\n"+
			"operator sales.every_line_at_most:\n"+
			"  takes value: number of each row\n"+
			"  takes limit: number of each row\n"+
			"  refutes on some rows\n```\n\n"+
			"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n"+
			"| value | ceiling |\n| --- | --- |\n| 2 | 10 |\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), `declares no input named "ceiling"`) {
		t.Fatalf("a column that is not an input was accepted: %v", err)
	}
}

func TestTEIMDRowSetExamplesRefuseADeclaredInputWithNoColumn(t *testing.T) {
	root := rowSetCasesWorkspace(t)
	writeTEIMDTestFile(t, root, rowSetCasesDocument,
		"# 行の上限\n\n```tei\nnamespace sales\n\n"+
			"operator sales.every_line_at_most:\n"+
			"  takes value: number of each row\n"+
			"  takes limit: number of each row\n"+
			"  refutes on some rows\n```\n\n"+
			"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n"+
			"| value |\n| --- |\n| 2 |\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), `declares input "limit"`) {
		t.Fatalf("a missing input column was accepted: %v", err)
	}
}

// One operator's examples use one form: a row-set table and a one-value table
// for the same operator ask two different questions, so both are not kept.
func TestTEIMDRowSetExamplesAreNotMixedWithOneValueExamples(t *testing.T) {
	root := rowSetCasesWorkspace(t)
	writeTEIMDTestFile(t, root, rowSetCasesDocument,
		"# 行の上限\n\n```tei\nnamespace sales\n\n"+
			"operator sales.every_line_at_most:\n"+
			"  takes value: number of each row\n"+
			"  takes limit: number of each row\n"+
			"  refutes on some rows\n```\n\n"+
			"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n"+
			"| value | limit |\n| --- | --- |\n| 2 | 10 |\n\n"+
			"<!-- tei: cases of sales.every_line_at_most -->\n"+
			"| value | expect |\n| --- | --- |\n| 5 | satisfied |\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "do not mix one-value examples with row-set examples") {
		t.Fatalf("the two forms were mixed: %v", err)
	}
}

// A row-set table for a one-record operator says something the declaration
// never asks, so the example is refused rather than attached.
func TestTEIMDRowSetExamplesAreRefusedForAOneRecordOperator(t *testing.T) {
	root := copyTEIMDRuleWorkspace(t)
	rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.authoring/v1")
	writeTEIMDTestFile(t, root, teiMDOperatorDeclarations,
		"# 値引きの検査に使う演算子\n\n```tei\nnamespace sales\n\noperator sales.non_negative: takes number\n```\n")
	writeTEIMDTestFile(t, root, rowSetCasesDocument,
		"# 行の上限\n\n<!-- tei: cases of sales.non_negative expecting satisfied -->\n"+
			"| value |\n| --- |\n| 2 |\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH") {
		t.Fatalf("a row-set example was accepted for a one-record operator: %v", err)
	}
}
