package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// Design §12, work item 1: "宣言にない入力名 ... が診断になる". A rule must not assign
// an input the operator it names does not declare.
func TestRuleAuthoringRefusesAnAssignmentToAnUndeclaredInput(t *testing.T) {
	src := `namespace sales

operator sales.within: takes low: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 100 high: 200
`
	_, issues := syntax.RuleNamespace(mustParseFile(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want exactly the undeclared input one", issues)
	}
	if !strings.Contains(issues[0].Msg, `declares no input named "high"`) {
		t.Fatalf("message=%q", issues[0].Msg)
	}
	// The diagnostic points at the name the author wrote, not at the rule start.
	if issues[0].Span == (syntax.Anchor{}) {
		t.Fatal("the issue carries no span")
	}
}

// The declared name is accepted, and the rule converts.
func TestRuleAuthoringAcceptsAnAssignmentToADeclaredInput(t *testing.T) {
	src := `namespace sales

operator sales.within: takes low: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 100
`
	namespace, issues := syntax.RuleNamespace(mustParseFile(t, src))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want none", issues)
	}
	if len(namespace.Validations) != 1 {
		t.Fatalf("validations=%+v", namespace.Validations)
	}
	if namespace.Validations[0].Operator != "sales.within" {
		t.Fatalf("operator=%q", namespace.Validations[0].Operator)
	}
}

// The unnamed single-input form declares no name of its own, so it cannot be
// asked for a second input. Assigning one is refused rather than ignored.
func TestRuleAuthoringRefusesASecondInputOnTheUnnamedForm(t *testing.T) {
	src := `namespace sales

operator sales.non_negative: takes number

rule sales.order_line.discount_non_negative:
  sales.order_line.discount is sales.non_negative low: 100
`
	_, issues := syntax.RuleNamespace(mustParseFile(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v", issues)
	}
	if !strings.Contains(issues[0].Msg, `declares no input named "low"`) {
		t.Fatalf("message=%q", issues[0].Msg)
	}
}

func mustParseFile(t *testing.T, src string) syntax.RuleAuthoringFile {
	t.Helper()
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
