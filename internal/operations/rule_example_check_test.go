package operations

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// These tests ask CheckRuleExamples about examples built by hand, without a
// document and without the loader. The expected findings are written from the
// meaning of the rules.

func exampleCheckWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "stock",
		Validations: []semantic.Validation{
			{ID: "stock.item.quantity_positive", Symbol: "quantity_positive", Kind: semantic.ValidationPositive, Attribute: "stock.item.quantity"},
			{ID: "stock.shelf.items.code_unique", Symbol: "code_unique", Kind: semantic.ValidationRelationUnique,
				Attribute: "stock.item.code", Relation: "stock.shelf.items"},
			{ID: "stock.shelf.items.slot_unique", Symbol: "slot_unique", Kind: semantic.ValidationRelationUnique,
				Attribute: "stock.item.slot", Relation: "stock.shelf.items"},
			{ID: "stock.item.quantity_even", Symbol: "quantity_even", Kind: semantic.ValidationOperator,
				Attribute: "stock.item.quantity", Operator: "stock.even"},
		},
		Operators: []semantic.Operator{{ID: "stock.even", Symbol: "even", Input: semantic.OperatorInputNumber}},
	}}}
}

func exampleCell(text string, offset int) workspace.RuleExampleCell {
	return workspace.RuleExampleCell{Text: text, Range: sourceindex.Range{Start: sourceindex.Position{Offset: offset}}}
}

func exampleAt(offset int) sourceindex.Range {
	return sourceindex.Range{Start: sourceindex.Position{Offset: offset, Line: 9, Column: 3}, End: sourceindex.Position{Offset: offset + 9}}
}

// stockExample has three records. The slot is a number, the code is a text.
//
//	code  slot  quantity
//	A-1   7     4
//	a-1   7.0   3
//	      8
func stockExample(expectations ...workspace.RuleExampleExpectation) workspace.RuleExample {
	return workspace.RuleExample{
		File: "semantic/stock/examples.tei.md", Data: "stock.item",
		Items: []workspace.RuleExampleItem{
			{Attribute: "stock.item.code", Symbol: "code"},
			{Attribute: "stock.item.slot", Symbol: "slot", Numeric: true},
			{Attribute: "stock.item.quantity", Symbol: "quantity", Numeric: true},
		},
		Rows: [][]workspace.RuleExampleCell{
			{exampleCell("A-1", 10), exampleCell("7", 14), exampleCell("4", 16)},
			{exampleCell("a-1", 20), exampleCell("7.0", 24), exampleCell("3", 28)},
			{exampleCell("", 30), exampleCell("8", 32), exampleCell("", 34)},
		},
		Expectations: expectations,
	}
}

var evenImplementation = validationeval.NamedImplementation{
	ID: "test.even", Input: semantic.OperatorInputNumber,
	Number: func(value *big.Rat) (bool, error) {
		return value.IsInt() && new(big.Int).Rem(value.Num(), big.NewInt(2)).Sign() == 0, nil
	},
}

func expectAnswer(rule core.CanonicalID, row int, expect syntax.RuleExampleExpect, offset int) workspace.RuleExampleExpectation {
	return workspace.RuleExampleExpectation{Rule: rule, Row: row, Expect: expect, Range: exampleAt(offset)}
}

func TestCheckRuleExamplesAgreesWithAnswersWrittenFromTheRules(t *testing.T) {
	loaded := workspace.Result{
		World:                   exampleCheckWorld(),
		OperatorImplementations: []workspace.OperatorImplementationSelection{{Operator: "stock.even", Implementation: "test.even"}},
		RuleExamples: []workspace.RuleExample{stockExample(
			// Codes differ in case and one is empty: texts are compared as written.
			expectAnswer("stock.shelf.items.code_unique", -1, syntax.RuleExampleSatisfied, 100),
			// 7 and 7.0 are the same slot.
			expectAnswer("stock.shelf.items.slot_unique", -1, syntax.RuleExampleNotSatisfied, 110),
			expectAnswer("stock.item.quantity_positive", 0, syntax.RuleExampleSatisfied, 120),
			expectAnswer("stock.item.quantity_positive", 2, syntax.RuleExampleUndecided, 130),
			expectAnswer("stock.item.quantity_even", 0, syntax.RuleExampleSatisfied, 140),
			expectAnswer("stock.item.quantity_even", 1, syntax.RuleExampleNotSatisfied, 150),
			expectAnswer("stock.item.quantity_even", 2, syntax.RuleExampleUndecided, 160),
		)},
	}
	if got := CheckRuleExamples(loaded, []validationeval.NamedImplementation{evenImplementation}); len(got) != 0 {
		t.Fatalf("findings=%+v", got)
	}
}

func TestCheckRuleExamplesReportsEachDifferentAnswerWhereItIsWritten(t *testing.T) {
	loaded := workspace.Result{
		World:                   exampleCheckWorld(),
		OperatorImplementations: []workspace.OperatorImplementationSelection{{Operator: "stock.even", Implementation: "test.even"}},
		RuleExamples: []workspace.RuleExample{stockExample(
			expectAnswer("stock.shelf.items.code_unique", -1, syntax.RuleExampleNotSatisfied, 100),
			expectAnswer("stock.shelf.items.slot_unique", -1, syntax.RuleExampleSatisfied, 110),
			expectAnswer("stock.item.quantity_positive", 0, syntax.RuleExampleSatisfied, 120), // agrees
			expectAnswer("stock.item.quantity_positive", 2, syntax.RuleExampleNotSatisfied, 130),
			expectAnswer("stock.item.quantity_even", 1, syntax.RuleExampleSatisfied, 150),
		)},
	}
	got := CheckRuleExamples(loaded, []validationeval.NamedImplementation{evenImplementation})
	type finding struct {
		code, rule, answer string
		offset             int
	}
	var findings []finding
	for _, diagnostic := range got {
		if len(diagnostic.Sources) != 1 || diagnostic.Sources[0].ArtifactPath != "semantic/stock/examples.tei.md" ||
			diagnostic.Sources[0].Role != sourceindex.RoleAuthoring || diagnostic.Diagnostic.Subject == nil ||
			diagnostic.Diagnostic.Subject.Kind != "semantic-validation" {
			t.Fatalf("diagnostic=%+v", diagnostic)
		}
		answer := diagnostic.Diagnostic.Message[strings.LastIndex(diagnostic.Diagnostic.Message, " answers ")+len(" answers "):]
		findings = append(findings, finding{diagnostic.Diagnostic.Code, diagnostic.Diagnostic.Subject.ID, answer, diagnostic.Sources[0].Range.Start.Offset})
	}
	want := []finding{
		{RuleExampleMismatchCode, "stock.shelf.items.code_unique", "satisfied", 100},
		{RuleExampleMismatchCode, "stock.shelf.items.slot_unique", "not-satisfied", 110},
		{RuleExampleMismatchCode, "stock.item.quantity_positive", "undecided (value-null (attribute stock.item.quantity))", 130},
		{RuleExampleMismatchCode, "stock.item.quantity_even", "not-satisfied", 150},
	}
	if !reflect.DeepEqual(findings, want) {
		t.Fatalf("findings=%+v\nwant=%+v", findings, want)
	}
}

// An example that could not be checked is a finding of its own. It is never
// counted among the examples that agree.
func TestCheckRuleExamplesNamesWhatItCouldNotAsk(t *testing.T) {
	expectations := []workspace.RuleExampleExpectation{
		expectAnswer("stock.item.quantity_even", 0, syntax.RuleExampleSatisfied, 140),
		expectAnswer("stock.item.quantity_positive", 0, syntax.RuleExampleSatisfied, 120),
		expectAnswer("stock.item.quantity_gone", 0, syntax.RuleExampleSatisfied, 170),
	}
	codes := func(loaded workspace.Result, available ...validationeval.NamedImplementation) map[string]string {
		out := map[string]string{}
		for _, diagnostic := range CheckRuleExamples(loaded, available) {
			out[diagnostic.Diagnostic.Subject.ID] = diagnostic.Diagnostic.Code
		}
		return out
	}

	t.Run("no implementation is selected", func(t *testing.T) {
		loaded := workspace.Result{World: exampleCheckWorld(), RuleExamples: []workspace.RuleExample{stockExample(expectations...)}}
		want := map[string]string{"stock.item.quantity_even": RuleExampleNotCheckedCode, "stock.item.quantity_gone": RuleExampleNotCheckedCode}
		if got := codes(loaded, evenImplementation); !reflect.DeepEqual(got, want) {
			t.Fatalf("got=%v", got)
		}
	})
	t.Run("the selected implementation is not available", func(t *testing.T) {
		loaded := workspace.Result{World: exampleCheckWorld(), RuleExamples: []workspace.RuleExample{stockExample(expectations...)},
			OperatorImplementations: []workspace.OperatorImplementationSelection{{Operator: "stock.even", Implementation: "test.even"}}}
		want := map[string]string{"stock.item.quantity_even": RuleExampleNotCheckedCode, "stock.item.quantity_gone": RuleExampleNotCheckedCode}
		if got := codes(loaded); !reflect.DeepEqual(got, want) {
			t.Fatalf("got=%v", got)
		}
	})
	t.Run("a Workspace without examples has no findings", func(t *testing.T) {
		if got := CheckRuleExamples(workspace.Result{World: exampleCheckWorld()}, nil); got != nil {
			t.Fatalf("got=%+v", got)
		}
	})
}

// A column the example does not have is unread for every row, so a rule over
// all the rows cannot be decided from it.
func TestCheckRuleExamplesTreatsAMissingColumnAsUnread(t *testing.T) {
	example := stockExample(expectAnswer("stock.shelf.items.slot_unique", -1, syntax.RuleExampleUndecided, 110))
	example.Items = example.Items[:1]
	for i := range example.Rows {
		example.Rows[i] = example.Rows[i][:1]
	}
	loaded := workspace.Result{World: exampleCheckWorld(), RuleExamples: []workspace.RuleExample{example}}
	if got := CheckRuleExamples(loaded, nil); len(got) != 0 {
		t.Fatalf("findings=%+v", got)
	}
	loaded.RuleExamples[0].Expectations[0].Expect = syntax.RuleExampleSatisfied
	got := CheckRuleExamples(loaded, nil)
	if len(got) != 1 || !strings.Contains(got[0].Diagnostic.Message, "relation-key-not-fetched") {
		t.Fatalf("findings=%+v", got)
	}
}
