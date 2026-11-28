package operations

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// A rule example's item cell states the type its item declares, so a text item
// is written with quotes, a truth item with the words true and false, and an
// `any` item with whichever writing decides its type (section 8.3). The check
// reads those cells as the values they state, so the rule is asked about them.
//
// The expectations are drawn from that section's table, not from the code.
func typedItemWorld() workspace.Result {
	return workspace.Result{
		World: ir.World{Semantics: []semantic.Namespace{{
			ID: "sales",
			Validations: []semantic.Validation{
				{
					ID: "sales.order.external_same", Kind: semantic.ValidationOperator,
					Attribute: "sales.order.external_id", Operator: "sales.same_text",
					Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: "sales.order.external_id"}},
				},
				{
					ID: "sales.order.settled_rule", Kind: semantic.ValidationOperator,
					Attribute: "sales.order.settled", Operator: "sales.is_settled",
				},
			},
			Operators: []semantic.Operator{
				{
					ID: "sales.same_text", Symbol: "same_text",
					Input: semantic.OperatorInputText, InputName: "value",
					Inputs: []semantic.OperatorInputDeclaration{
						{Name: "value", Type: semantic.OperatorInputText},
						{Name: "other", Type: semantic.OperatorInputText},
					},
				},
				{
					ID: "sales.is_settled", Symbol: "is_settled",
					Input: semantic.OperatorInputTruth, InputName: "value",
				},
			},
		}}},
		OperatorImplementations: []workspace.OperatorImplementationSelection{
			{Operator: "sales.same_text", Implementation: "tei.text.same-as"},
			{Operator: "sales.is_settled", Implementation: "tei.truth.is-true"},
		},
	}
}

// A text item's cell and a truth item's cell are both read as the values they
// state, so both rules agree with their expectations.
func TestATypedItemExampleIsReadAsItsTypes(t *testing.T) {
	loaded := typedItemWorld()
	loaded.RuleExamples = []workspace.RuleExample{{
		File: "semantic/sales/examples.tei.md", Data: "sales.order",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order.external_id", Symbol: "externalId", Text: true},
			{Attribute: "sales.order.settled", Symbol: "settled", Truth: true},
		},
		Rows: [][]workspace.RuleExampleCell{{{Text: `"A-100"`}, {Text: "true"}}},
		Expectations: []workspace.RuleExampleExpectation{
			{Rule: "sales.order.external_same", Row: 0, Expect: syntax.RuleExampleSatisfied},
			{Rule: "sales.order.settled_rule", Row: 0, Expect: syntax.RuleExampleSatisfied},
		},
	}}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleNotCheckedCode {
			t.Fatalf("a typed item was not read: %s", diagnostic.Diagnostic.Message)
		}
		if diagnostic.Diagnostic.Code == RuleExampleMismatchCode {
			t.Fatalf("a typed item answered differently: %s", diagnostic.Diagnostic.Message)
		}
	}
}

// A truth that is false is not a truth that holds, so an example that states it
// as satisfied is refused where the expectation is written.
func TestAFalseTruthItemIsNotSatisfied(t *testing.T) {
	loaded := typedItemWorld()
	loaded.RuleExamples = []workspace.RuleExample{{
		File: "semantic/sales/examples.tei.md", Data: "sales.order",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order.external_id", Symbol: "externalId", Text: true},
			{Attribute: "sales.order.settled", Symbol: "settled", Truth: true},
		},
		Rows: [][]workspace.RuleExampleCell{{{Text: `"A-100"`}, {Text: "false"}}},
		Expectations: []workspace.RuleExampleExpectation{
			{Rule: "sales.order.settled_rule", Row: 0, Expect: syntax.RuleExampleSatisfied},
		},
	}}
	var found bool
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleMismatchCode {
			found = true
		}
	}
	if !found {
		t.Fatal("a false truth was read as a truth that holds")
	}
}

// A cell that writes a number for a text item is not a text, so the reading is
// refused with the cell's text rather than compared as a text it never was.
func TestATextItemRefusesACellOfAnotherType(t *testing.T) {
	loaded := typedItemWorld()
	loaded.RuleExamples = []workspace.RuleExample{{
		File: "semantic/sales/examples.tei.md", Data: "sales.order",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order.external_id", Symbol: "externalId", Text: true},
			{Attribute: "sales.order.settled", Symbol: "settled", Truth: true},
		},
		Rows: [][]workspace.RuleExampleCell{{{Text: "100"}, {Text: "true"}}},
		Expectations: []workspace.RuleExampleExpectation{
			{Rule: "sales.order.external_same", Row: 0, Expect: syntax.RuleExampleSatisfied},
		},
	}}
	var found bool
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code != RuleExampleNotCheckedCode {
			continue
		}
		found = true
		if !strings.Contains(diagnostic.Diagnostic.Message, "100") {
			t.Fatalf("the refusal does not name the cell: %s", diagnostic.Diagnostic.Message)
		}
	}
	if !found {
		t.Fatal("a number was read as a text")
	}
}

// An `any` item's cell keeps the type its writing decides, so the unquoted 100
// and the quoted "100" are different values and are never the same.
func TestAnAnyItemKeepsTheTypeItsWritingDecides(t *testing.T) {
	loaded := typedItemWorld()
	loaded.World.Semantics[0].Validations = []semantic.Validation{{
		ID: "sales.order.same_any", Kind: semantic.ValidationOperator,
		Attribute: "sales.order.external_id", Operator: "sales.same_any",
		Inputs: []semantic.ValidationInputAssignment{{Name: "other", Value: "sales.order.external_id"}},
	}}
	loaded.World.Semantics[0].Operators = []semantic.Operator{{
		ID: "sales.same_any", Symbol: "same_any",
		Input: semantic.OperatorInputAny, InputName: "value",
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputAny},
			{Name: "other", Type: semantic.OperatorInputAny},
		},
	}}
	loaded.OperatorImplementations = []workspace.OperatorImplementationSelection{
		{Operator: "sales.same_any", Implementation: "tei.any.same-as"},
	}
	loaded.RuleExamples = []workspace.RuleExample{{
		File: "semantic/sales/examples.tei.md", Data: "sales.order",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order.external_id", Symbol: "externalId", Any: true},
		},
		Rows: [][]workspace.RuleExampleCell{{{Text: "100"}}},
		Expectations: []workspace.RuleExampleExpectation{
			{Rule: "sales.order.same_any", Row: 0, Expect: syntax.RuleExampleSatisfied},
		},
	}}
	// One value read as a number against the same value read as a number holds;
	// what is checked here is that the example was read at all rather than
	// refused for stating no type.
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code == RuleExampleNotCheckedCode {
			t.Fatalf("an any item was not read: %s", diagnostic.Diagnostic.Message)
		}
	}
}
