package operations

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// A rule example states records, so an item's cell states its value: any
// written cell is a value the record holds, an empty cell is a record that holds
// nothing, and the word for "not read" is that state (section 8.3).
//
// A rule whose operator asks whether the attribute holds a value is answered from
// those three states. The word `not-fetched` therefore has to be read as "nobody
// read it" rather than as a value that is there: reading it as present answers
// the presence question from a value nobody looked at.
func presenceItemWorld() workspace.Result {
	return workspace.Result{
		World: ir.World{Semantics: []semantic.Namespace{{
			ID: "sales",
			Validations: []semantic.Validation{{
				ID: "sales.order_line.code_present", Kind: semantic.ValidationOperator,
				Attribute: "sales.order_line.item_code", Operator: "sales.holds",
			}},
			Operators: []semantic.Operator{{
				ID: "sales.holds", Symbol: "holds",
				Input: semantic.OperatorInputPresence, InputName: "value",
			}},
		}}},
		OperatorImplementations: []workspace.OperatorImplementationSelection{
			{Operator: "sales.holds", Implementation: "tei.presence.is-present"},
		},
	}
}

func presenceItemExample(expect syntax.RuleExampleExpect, cell string) workspace.RuleExample {
	return workspace.RuleExample{
		File: "semantic/sales/examples.tei.md", Data: "sales.order_line",
		Items: []workspace.RuleExampleItem{
			{Attribute: "sales.order_line.item_code", Symbol: "itemCode", Text: true},
		},
		Rows: [][]workspace.RuleExampleCell{{{Text: cell}}},
		Expectations: []workspace.RuleExampleExpectation{{
			Rule: "sales.order_line.code_present", Row: 0, Expect: expect,
		}},
	}
}

// A record that writes a value holds one, so the presence question is answered
// "yes".
func TestAPresenceRuleReadsAWrittenCellAsAValue(t *testing.T) {
	loaded := presenceItemWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		presenceItemExample(syntax.RuleExampleSatisfied, `"A-1"`),
	}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code != "" {
			t.Fatalf("a written cell was not read as a value: %s %s",
				diagnostic.Diagnostic.Code, diagnostic.Diagnostic.Message)
		}
	}
}

// A record that writes no value holds none, so the presence question is
// answered "no" and the rule is decided rather than left undecided.
func TestAPresenceRuleReadsAnEmptyCellAsNoValue(t *testing.T) {
	loaded := presenceItemWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		presenceItemExample(syntax.RuleExampleNotSatisfied, ""),
	}
	for _, diagnostic := range CheckRuleExamples(loaded, shippedImplementations()) {
		if diagnostic.Diagnostic.Code != "" {
			t.Fatalf("an empty cell was not read as no value: %s %s",
				diagnostic.Diagnostic.Code, diagnostic.Diagnostic.Message)
		}
	}
}

// The word `not-fetched` states that nobody read the row, so the presence
// question has no answer: the example states undecided, and a table that states
// satisfied is refused where the expectation is written.
func TestAPresenceRuleDoesNotReadTheNotReadWordAsAValue(t *testing.T) {
	loaded := presenceItemWorld()
	loaded.RuleExamples = []workspace.RuleExample{
		presenceItemExample(syntax.RuleExampleSatisfied, "not-fetched"),
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
		t.Fatal("a row nobody read was read as a value the record holds")
	}
}
