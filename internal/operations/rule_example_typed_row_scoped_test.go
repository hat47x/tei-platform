package operations

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// A rule whose operator reads every row in a type other than number is decided
// by reading each input's column in the type the declaration states (decision
// section 8.4). The example lists those rows as columns and states the one
// answer in the mark.
//
// The expectations are drawn from section 8.1's row-set example and section
// 8.3's writing per type, not from the code.
func typedRowExampleWorld() workspace.Result {
	return workspace.Result{
		World: ir.World{Semantics: []semantic.Namespace{{
			ID: "sales",
			Validations: []semantic.Validation{{
				ID:        "sales.order_line.promised_not_before",
				Kind:      semantic.ValidationOperator,
				Attribute: "sales.order_line.promised_on", Operator: "sales.not_before",
				Relation: "sales.order.lines",
				Inputs: []semantic.ValidationInputAssignment{{
					Name: "deadline", Value: "sales.order_line.promised_by",
				}},
			}},
			Operators: []semantic.Operator{{
				ID: "sales.not_before", Symbol: "not_before",
				Input: semantic.OperatorInputDate, InputName: "value",
				Inputs: []semantic.OperatorInputDeclaration{
					{Name: "value", Type: semantic.OperatorInputDate, OfEachRow: true},
					{Name: "deadline", Type: semantic.OperatorInputDate, OfEachRow: true},
				},
			}},
		}}},
		OperatorImplementations: []workspace.OperatorImplementationSelection{
			{Operator: "sales.not_before", Implementation: "tei.day.every-not-before"},
		},
	}
}

// typedRowExample states the two day columns of one row and one answer in the
// mark.
func typedRowExample(expect syntax.RuleExampleExpect, promisedOn string, promisedBy string) workspace.RuleExample {
	return workspace.RuleExample{
		File: "semantic/sales/examples.tei.md", Data: "sales.order_line",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order_line.promised_on", Symbol: "promisedOn", Date: true},
			{Attribute: "sales.order_line.promised_by", Symbol: "promisedBy", Date: true},
		},
		Rows: [][]workspace.RuleExampleCell{{{Text: promisedOn}, {Text: promisedBy}}},
		Expectations: []workspace.RuleExampleExpectation{{
			Rule: "sales.order_line.promised_not_before", Row: -1, Expect: expect,
		}},
	}
}

// The day is not before its deadline, so the rule holds.
func TestATypedRowScopedRuleExampleIsCheckedAsDays(t *testing.T) {
	loaded := typedRowExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		typedRowExample(syntax.RuleExampleSatisfied, "2026-04-01", "2026-03-01"),
	}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleNotCheckedCode {
			t.Fatalf("the rule was not asked about the rows as days: %s", diagnostic.Diagnostic.Message)
		}
		if diagnostic.Diagnostic.Code == RuleExampleMismatchCode {
			t.Fatalf("the rule answered differently from the example: %s", diagnostic.Diagnostic.Message)
		}
	}
}

// A day before its deadline is the counterexample, so the rule does not hold.
func TestATypedRowScopedRuleExampleStatesTheCounterexampleRow(t *testing.T) {
	loaded := typedRowExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		typedRowExample(syntax.RuleExampleNotSatisfied, "2026-03-01", "2026-04-01"),
	}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleNotCheckedCode {
			t.Fatalf("the rule was not asked about the rows as days: %s", diagnostic.Diagnostic.Message)
		}
		if diagnostic.Diagnostic.Code == RuleExampleMismatchCode {
			t.Fatalf("the rule answered differently from the example: %s", diagnostic.Diagnostic.Message)
		}
	}
}

// A deadline nobody read is not a deadline the day met, so an example that states
// it as satisfied is refused where the expectation is written.
func TestATypedRowScopedRuleExampleWithAnUnreadDeadlineIsNotSatisfied(t *testing.T) {
	loaded := typedRowExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		typedRowExample(syntax.RuleExampleSatisfied, "2026-04-01", "not-fetched"),
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
		t.Fatal("an unread deadline was read as a deadline the day met")
	}
}

// A cell that is not the input's type is refused with the text it holds, so the
// author can find the cell rather than reading a day nobody wrote.
func TestATypedRowScopedRuleExampleRefusesACellOfAnotherType(t *testing.T) {
	loaded := typedRowExampleWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		typedRowExample(syntax.RuleExampleSatisfied, "2026-04-01", "5"),
	}
	var found bool
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleNotCheckedCode {
			found = true
			if !strings.Contains(diagnostic.Diagnostic.Message, "5") {
				t.Fatalf("the refusal does not name the cell: %s", diagnostic.Diagnostic.Message)
			}
		}
	}
	if !found {
		t.Fatal("a decimal cell was read as a day")
	}
}
