package operations

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// The world and the example are built the way this package's own tests build
// them: the rule reads one number and the shipped implementation decides it.
func notFetchedWorld() workspace.Result {
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
		RuleExamples: []workspace.RuleExample{{
			File: "semantic/sales/examples.tei.md", Data: "sales.order_line",
			Items: []workspace.RuleExampleItem{
				{Attribute: "sales.order_line.quantity", Symbol: "quantity", Numeric: true},
			},
			Rows: [][]workspace.RuleExampleCell{{{Text: "not-fetched"}}},
			Expectations: []workspace.RuleExampleExpectation{{
				Rule: "sales.order_line.quantity_non_negative", Row: 0,
				Expect: syntax.RuleExampleUndecided,
			}},
		}},
	}
}

// nonNegativeImplementation is the shipped implementation of the operator the
// rule reads, so the check asks the same code the product asks.
func nonNegativeImplementation() []validationeval.NamedImplementation {
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID != "tei.number.non-negative" {
			continue
		}
		return []validationeval.NamedImplementation{available}
	}
	return nil
}

// A rule example's item cell may write the word for "not read", and section 8.3
// says that writing does not depend on the type. The check then reads the cell as
// the state the word states, so the rule is asked about it and answers undecided
// rather than being unable to evaluate.
//
// The expectations are drawn from that section's table, not from the code.
func TestARuleExampleWithANotReadItemCellIsChecked(t *testing.T) {
	got := CheckRuleExamples(notFetchedWorld(), nonNegativeImplementation())
	for _, diagnostic := range got {
		if diagnostic.Diagnostic.Code == "RULE_EXAMPLE_NOT_CHECKED" {
			t.Fatalf("the rule was not asked about the example: %s", diagnostic.Diagnostic.Message)
		}
		if diagnostic.Diagnostic.Code == "RULE_EXAMPLE_MISMATCH" {
			t.Fatalf("the rule answered differently from the example: %s", diagnostic.Diagnostic.Message)
		}
	}
}

// The same example expecting satisfied disagrees with the rule, because a value
// nobody read is not a number above zero. The check reports where the
// expectation is written, which is the place to repair.
func TestARuleExampleWithANotReadItemCellDoesNotAnswerSatisfied(t *testing.T) {
	loaded := notFetchedWorld()
	loaded.RuleExamples[0].Expectations[0].Expect = syntax.RuleExampleSatisfied

	var found bool
	for _, diagnostic := range CheckRuleExamples(loaded, nonNegativeImplementation()) {
		if diagnostic.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" {
			continue
		}
		found = true
		if !strings.Contains(diagnostic.Diagnostic.Message, "undecided") {
			t.Fatalf("the answer is not undecided: %s", diagnostic.Diagnostic.Message)
		}
	}
	if !found {
		t.Fatal("a not-read cell was read as a number above zero")
	}
}
