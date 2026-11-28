package operations

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// A row example of an operator whose input reads presence states the row's
// state with `present`, `absent`, or `not-fetched` (decision section 8.6), and
// the example is checked against the row implementation the Workspace selected.
//
// The expectations are drawn from that table, not from the code: a row that
// holds no value refutes "every row holds a value", a row nobody read leaves it
// undecided, and an empty relation satisfies it.

// presenceRowExampleWorld declares the row-scoped presence operator and its
// rule, and selects the shipped implementation.
func presenceRowExampleWorld() workspace.Result {
	return workspace.Result{
		World: ir.World{Semantics: []semantic.Namespace{{
			ID: "sales",
			Validations: []semantic.Validation{{
				ID:        "sales.order.lines.every_price_stated",
				Kind:      semantic.ValidationOperator,
				Attribute: "sales.order_line.unit_price", Operator: "sales.every_price_stated",
				Relation: "sales.order.lines",
			}},
			Operators: []semantic.Operator{{
				ID: "sales.every_price_stated", Symbol: "every_price_stated",
				Input: semantic.OperatorInputPresence, InputName: "value",
				Inputs: []semantic.OperatorInputDeclaration{
					{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
				},
				RefutesOnSomeRows: true,
			}},
		}}},
		OperatorImplementations: []workspace.OperatorImplementationSelection{
			{Operator: "sales.every_price_stated", Implementation: "tei.presence.every-present"},
		},
	}
}

// presenceRowExample states one row of example per price cell, and one answer
// in the mark.
func presenceRowExample(expect syntax.RuleExampleExpect, cells ...string) workspace.RuleExample {
	example := workspace.RuleExample{
		File: "semantic/sales/examples.tei.md", Data: "sales.order_line",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order_line.unit_price", Symbol: "unitPrice"},
		},
		Expectations: []workspace.RuleExampleExpectation{{
			Rule: "sales.order.lines.every_price_stated", Row: -1, Expect: expect,
		}},
	}
	for _, cell := range cells {
		example.Rows = append(example.Rows, []workspace.RuleExampleCell{{Text: cell}})
	}
	return example
}

// Every row of the example holds a value, so the rule holds.
func TestAPresenceRowExampleIsCheckedAgainstTheRows(t *testing.T) {
	loaded := presenceRowExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		presenceRowExample(syntax.RuleExampleSatisfied, "present", "present"),
	}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleNotCheckedCode {
			t.Fatalf("the rule was not asked about the rows: %s", diagnostic.Diagnostic.Message)
		}
		if diagnostic.Diagnostic.Code == RuleExampleMismatchCode {
			t.Fatalf("the rule answered differently from the example: %s", diagnostic.Diagnostic.Message)
		}
	}
}

// A row that says `absent` holds no value, and the contract says a counterexample
// among the rows read is enough, so the rule does not hold.
func TestAPresenceRowExampleReadsAbsentAsNoValue(t *testing.T) {
	loaded := presenceRowExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		presenceRowExample(syntax.RuleExampleNotSatisfied, "present", "absent"),
	}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleNotCheckedCode {
			t.Fatalf("the rule was not asked about the rows: %s", diagnostic.Diagnostic.Message)
		}
		if diagnostic.Diagnostic.Code == RuleExampleMismatchCode {
			t.Fatalf("the rule answered differently from the example: %s", diagnostic.Diagnostic.Message)
		}
	}
}

// A row nobody read is not a row that holds no value, so an example that states
// `satisfied` is refused where the expectation is written, and the answer is
// named as undecided.
func TestAPresenceRowExampleWithAnUnreadRowIsUndecided(t *testing.T) {
	loaded := presenceRowExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		presenceRowExample(syntax.RuleExampleSatisfied, "present", "not-fetched"),
	}
	var found bool
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code != RuleExampleMismatchCode {
			continue
		}
		found = true
		if !strings.Contains(diagnostic.Diagnostic.Message, "undecided") {
			t.Fatalf("the answer is not undecided: %s", diagnostic.Diagnostic.Message)
		}
	}
	if !found {
		t.Fatal("a row nobody read was read as a row with a value")
	}
}

// The empty relation satisfies "every row holds a value": no row holds none.
func TestAPresenceRowExampleOfTheEmptyRelationIsSatisfied(t *testing.T) {
	loaded := presenceRowExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		presenceRowExample(syntax.RuleExampleSatisfied),
	}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleMismatchCode || diagnostic.Diagnostic.Code == RuleExampleNotCheckedCode {
			t.Fatalf("the empty relation was not answered as satisfied: %s", diagnostic.Diagnostic.Message)
		}
	}
}
