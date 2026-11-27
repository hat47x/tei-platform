package operations

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// ruleExampleWorld builds the world and the selection the shipped
// implementation of the rule's operator needs, so the check asks the same code
// the product asks.
func ruleExampleWorld() workspace.Result {
	return workspace.Result{
		World: ir.World{Semantics: []semantic.Namespace{{
			ID: "sales",
			Validations: []semantic.Validation{{
				ID:        "sales.order_line.quantity_non_negative",
				Kind:      semantic.ValidationOperator,
				Attribute: "sales.order_line.quantity", Operator: "sales.non_negative",
			}},
			Operators: []semantic.Operator{{
				ID: "sales.non_negative", Symbol: "non_negative",
				Input: semantic.OperatorInputNumber, InputName: "value",
			}},
		}}},
		OperatorImplementations: []workspace.OperatorImplementationSelection{
			{Operator: "sales.non_negative", Implementation: "tei.number.non-negative"},
		},
	}
}

// A rule example is a table of records, and section 8.1 says the same value
// written twice in one table is kept rather than refused as a duplicate: each
// row is a record the author states, and a record may hold the same value as
// another. What is refused is an expectation that disagrees with the rule.
func TestARuleExampleKeepsTwoRowsOfTheSameValue(t *testing.T) {
	loaded := ruleExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{{
		File: "semantic/sales/examples.tei.md", Data: "sales.order_line",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order_line.quantity", Symbol: "quantity", Numeric: true},
		},
		Rows: [][]workspace.RuleExampleCell{
			{{Text: "300"}},
			{{Text: "300"}},
		},
		Expectations: []workspace.RuleExampleExpectation{
			{Rule: "sales.order_line.quantity_non_negative", Row: 0, Expect: syntax.RuleExampleSatisfied},
			{Rule: "sales.order_line.quantity_non_negative", Row: 1, Expect: syntax.RuleExampleNotSatisfied},
		},
	}}

	// The two rows are kept: the first expectation agrees and the second does
	// not, and the disagreement is reported where it is written.
	var mismatch int
	for _, diagnostic := range CheckRuleExamples(loaded, nonNegativeImplementation()) {
		if diagnostic.Diagnostic.Code != RuleExampleMismatchCode {
			continue
		}
		mismatch++
		if !strings.Contains(diagnostic.Diagnostic.Message, "not-satisfied") {
			t.Fatalf("the finding is not about the expectation that disagrees: %s", diagnostic.Diagnostic.Message)
		}
	}
	if mismatch != 1 {
		t.Fatalf("findings=%d, want the one expectation that disagrees", mismatch)
	}
}

// Two spellings of one number are the same value, so a row that states 300 and
// a row that states 300.0 are two rows of one value. The expectation that
// disagrees with the rule is reported either way.
func TestARuleExampleReadsTwoSpellingsOfOneNumber(t *testing.T) {
	loaded := ruleExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{{
		File: "semantic/sales/examples.tei.md", Data: "sales.order_line",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order_line.quantity", Symbol: "quantity", Numeric: true},
		},
		Rows: [][]workspace.RuleExampleCell{
			{{Text: "300"}},
			{{Text: "300.0"}},
		},
		Expectations: []workspace.RuleExampleExpectation{
			{Rule: "sales.order_line.quantity_non_negative", Row: 0, Expect: syntax.RuleExampleSatisfied},
			{Rule: "sales.order_line.quantity_non_negative", Row: 1, Expect: syntax.RuleExampleNotSatisfied},
		},
	}}

	var mismatch int
	for _, diagnostic := range CheckRuleExamples(loaded, nonNegativeImplementation()) {
		if diagnostic.Diagnostic.Code == RuleExampleMismatchCode {
			mismatch++
		}
	}
	if mismatch != 1 {
		t.Fatalf("findings=%d, want the one expectation that disagrees: 300 and 300.0 are one value", mismatch)
	}
}

// A row whose value was not read is kept the same way, so an author can state
// one record that was read and one that was not in the same table.
func TestARuleExampleKeepsTwoRowsOfNotFetched(t *testing.T) {
	loaded := ruleExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{{
		File: "semantic/sales/examples.tei.md", Data: "sales.order_line",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order_line.quantity", Symbol: "quantity", Numeric: true},
		},
		Rows: [][]workspace.RuleExampleCell{
			{{Text: "300"}},
			{{Text: "not-fetched"}},
		},
		Expectations: []workspace.RuleExampleExpectation{
			{Rule: "sales.order_line.quantity_non_negative", Row: 0, Expect: syntax.RuleExampleSatisfied},
			{Rule: "sales.order_line.quantity_non_negative", Row: 1, Expect: syntax.RuleExampleUndecided},
		},
	}}

	for _, diagnostic := range CheckRuleExamples(loaded, nonNegativeImplementation()) {
		t.Fatalf("a table that states one read row and one unread row was refused: %s %s",
			diagnostic.Diagnostic.Code, diagnostic.Diagnostic.Message)
	}
}
