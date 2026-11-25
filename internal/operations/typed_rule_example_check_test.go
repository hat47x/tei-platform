package operations

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// A rule whose operator reads a value that is not a number is asked about the
// example through the reader of that type. The example's cells are read as the
// values they state, so a text cell is a string and a boolean cell is a
// boolean, and the implementation is handed those and not a number.
func TestCheckRuleExamplesAsksATypedOperatorRuleAboutTheExample(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "stock",
		Validations: []semantic.Validation{{
			ID: "stock.item.code_prefix_a", Symbol: "code_prefix_a", Kind: semantic.ValidationOperator,
			Attribute: "stock.item.code", Operator: "stock.starts_with",
			Inputs: []semantic.ValidationInputAssignment{{Name: "prefix", Value: `"A-"`}},
		}},
		Operators: []semantic.Operator{{
			ID: "stock.starts_with", Symbol: "starts_with", Input: semantic.OperatorInputText,
			Inputs: []semantic.OperatorInputDeclaration{
				{Name: "value", Type: semantic.OperatorInputText},
				{Name: "prefix", Type: semantic.OperatorInputText},
			},
		}},
	}}}
	example := workspace.RuleExample{
		File: "semantic/stock/examples.tei.md", Data: "stock.item",
		Items: []workspace.RuleExampleItem{
			{Attribute: "stock.item.code", Symbol: "code"},
		},
		Rows: [][]workspace.RuleExampleCell{
			{exampleCell(`"A-1"`, 10)},
			{exampleCell(`"B-2"`, 20)},
		},
		Expectations: []workspace.RuleExampleExpectation{
			expectAnswer("stock.item.code_prefix_a", 0, syntax.RuleExampleSatisfied, 100),
			expectAnswer("stock.item.code_prefix_a", 1, syntax.RuleExampleNotSatisfied, 110),
		},
	}
	loaded := workspace.Result{
		World:                   world,
		OperatorImplementations: []workspace.OperatorImplementationSelection{{Operator: "stock.starts_with", Implementation: "test.starts-with"}},
		RuleExamples:            []workspace.RuleExample{example},
	}
	var asked []string
	available := []validationeval.NamedImplementation{{
		ID: "test.starts-with", Input: semantic.OperatorInputText,
		Values: func(values map[string]validationeval.OperatorValue) (bool, error) {
			value, prefix := values["value"].Text, values["prefix"].Text
			if value == nil || prefix == nil {
				return false, errors.New("an input was not handed as text")
			}
			asked = append(asked, *value)
			return strings.HasPrefix(*value, *prefix), nil
		},
	}}
	if got := CheckRuleExamples(loaded, available); len(got) != 0 {
		t.Fatalf("findings=%+v", got)
	}
	if len(asked) != 2 || asked[0] != "A-1" || asked[1] != "B-2" {
		t.Fatalf("the implementation was asked %v, want the two texts as written", asked)
	}
}

// An implementation that decides from numbers is not selected for an operator
// that reads text, so the rule example is reported as not checked rather than
// answered from a number.
func TestCheckRuleExamplesRefusesANumericImplementationForATypedRule(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "stock",
		Validations: []semantic.Validation{{
			ID: "stock.item.code_positive", Symbol: "code_positive", Kind: semantic.ValidationOperator,
			Attribute: "stock.item.code", Operator: "stock.non_negative",
		}},
		Operators: []semantic.Operator{{
			ID: "stock.non_negative", Symbol: "non_negative", Input: semantic.OperatorInputText,
			Inputs: []semantic.OperatorInputDeclaration{{Name: "value", Type: semantic.OperatorInputText}},
		}},
	}}}
	loaded := workspace.Result{
		World: world,
		OperatorImplementations: []workspace.OperatorImplementationSelection{
			{Operator: "stock.non_negative", Implementation: "test.number-only"},
		},
		RuleExamples: []workspace.RuleExample{{
			File: "semantic/stock/examples.tei.md", Data: "stock.item",
			Items: []workspace.RuleExampleItem{{Attribute: "stock.item.code", Symbol: "code"}},
			Rows:  [][]workspace.RuleExampleCell{{exampleCell(`"A-1"`, 10)}},
			Expectations: []workspace.RuleExampleExpectation{
				expectAnswer("stock.item.code_positive", 0, syntax.RuleExampleSatisfied, 100),
			},
		}},
	}
	available := []validationeval.NamedImplementation{{
		ID: "test.number-only", Input: semantic.OperatorInputNumber,
		Number: func(*big.Rat) (bool, error) { return true, nil },
	}}
	got := CheckRuleExamples(loaded, available)
	if len(got) != 1 {
		t.Fatalf("findings=%+v, want one", got)
	}
	if got[0].Diagnostic.Code != RuleExampleNotCheckedCode {
		t.Fatalf("code=%q, want %q", got[0].Diagnostic.Code, RuleExampleNotCheckedCode)
	}
	if !strings.Contains(got[0].Diagnostic.Message, "text") {
		t.Fatalf("the finding does not name the type: %q", got[0].Diagnostic.Message)
	}
}
