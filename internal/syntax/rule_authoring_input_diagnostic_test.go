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

// A declared name passes the name check, but the value still has nowhere to go:
// semantic.Validation carries the operator and its subject, not the input
// values. Converting would produce a rule that reads as the single-input
// operator, so the conversion refuses the assignment instead of dropping it.
// Work item 2 (the normalization) is what makes this convert.
func TestRuleAuthoringRefusesADeclaredInputAssignmentUntilTheValueCanBeCarried(t *testing.T) {
	src := `namespace sales

operator sales.within: takes low: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 100
`
	namespace, issues := syntax.RuleNamespace(mustParseFile(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want the value-not-carried one", issues)
	}
	if !strings.Contains(issues[0].Msg, "no place for an operator input value") {
		t.Fatalf("message=%q", issues[0].Msg)
	}
	// No half-resolved rule reaches the validator.
	if len(namespace.Validations) != 0 {
		t.Fatalf("a rule with an unrepresentable assignment converted anyway: %+v", namespace.Validations)
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
