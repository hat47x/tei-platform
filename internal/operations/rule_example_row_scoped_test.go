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

// A rule whose operator reads every row answers one answer about the rows, so
// its example states that answer in the mark with `expecting` and lists the rows
// as columns. The check then reads those rows and asks the selected
// implementation, which is the same path the product read path uses.
//
// The expectations are drawn from section 8.1's row-set example, not from the
// code.
func rowScopedExampleWorld() workspace.Result {
	return workspace.Result{
		World: ir.World{Semantics: []semantic.Namespace{{
			ID: "sales",
			Validations: []semantic.Validation{{
				ID:        "sales.order_line.quantity_at_most",
				Kind:      semantic.ValidationOperator,
				Attribute: "sales.order_line.quantity", Operator: "sales.at_most",
				Relation: "sales.order.lines",
				// The rule names the second input by the attribute it reads,
				// which is what an operator rule that reads rows states.
				Inputs: []semantic.ValidationInputAssignment{{
					Name: "limit", Value: "sales.order_line.unit_price",
				}},
			}},
			Operators: []semantic.Operator{{
				ID: "sales.at_most", Symbol: "at_most",
				Input: semantic.OperatorInputNumber, InputName: "value",
				Inputs: []semantic.OperatorInputDeclaration{
					{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
					{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
				},
			}},
		}}},
		OperatorImplementations: []workspace.OperatorImplementationSelection{
			{Operator: "sales.at_most", Implementation: "tei.number.every-at-most"},
		},
	}
}

// shippedImplementations is what the binary carries, so the check asks the same
// code the product asks. The row-scoped example needs the row implementation,
// which is not the one the one-record examples select.
func shippedImplementations() []validationeval.NamedImplementation {
	return validationeval.BuiltinImplementations()
}

// rowScopedExample states two rows of the relation and one answer in the mark.
func rowScopedExample(expect syntax.RuleExampleExpect, values ...string) workspace.RuleExample {
	example := workspace.RuleExample{
		File: "semantic/sales/examples.tei.md", Data: "sales.order_line",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order_line.quantity", Symbol: "quantity", Numeric: true},
			{Attribute: "sales.order_line.unit_price", Symbol: "unitPrice", Numeric: true},
		},
		Expectations: []workspace.RuleExampleExpectation{{
			Rule: "sales.order_line.quantity_at_most", Row: -1, Expect: expect,
		}},
	}
	for i := 0; i+1 < len(values); i += 2 {
		example.Rows = append(example.Rows, []workspace.RuleExampleCell{
			{Text: values[i]}, {Text: values[i+1]},
		})
	}
	return example
}

// Every row of the example is within its own limit, so the rule holds.
func TestARowScopedRuleExampleIsCheckedAgainstTheRows(t *testing.T) {
	loaded := rowScopedExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		rowScopedExample(syntax.RuleExampleSatisfied, "5", "10", "6", "9"),
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

// One row exceeds its own limit, so the rule does not hold and the example
// states that.
func TestARowScopedRuleExampleStatesTheCounterexampleRow(t *testing.T) {
	loaded := rowScopedExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		rowScopedExample(syntax.RuleExampleNotSatisfied, "5", "10", "11", "9"),
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

// A row nobody read is not a row above its limit and not a row within it, so an
// example that states it as satisfied is refused where the expectation is
// written.
func TestARowScopedRuleExampleWithAnUnreadRowIsNotSatisfied(t *testing.T) {
	loaded := rowScopedExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		rowScopedExample(syntax.RuleExampleSatisfied, "5", "10", "not-fetched", "9"),
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
		t.Fatal("a row nobody read was read as a row within its limit")
	}
}
