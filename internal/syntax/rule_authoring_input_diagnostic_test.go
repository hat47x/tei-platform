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

// The subject is already the first Operator input. Even a declared name
// cannot rebind it: the authored assignment would give it two values.
func TestRuleAuthoringRefusesRebindingOfSubjectInput(t *testing.T) {
	src := `namespace sales

operator sales.within: takes low: number

rule sales.order_line.discount_in_range:
  sales.order_line.discount is sales.within low: 100
`
	namespace, issues := syntax.RuleNamespace(mustParseFile(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want the subject-input rebinding issue", issues)
	}
	if !strings.Contains(issues[0].Msg, "already supplied by the rule subject") {
		t.Fatalf("message=%q", issues[0].Msg)
	}
	// No half-resolved rule reaches the validator.
	if len(namespace.Validations) != 0 {
		t.Fatalf("a rule with an unrepresentable assignment converted anyway: %+v", namespace.Validations)
	}
}

// Both the implicit and explicit names of the first input are already bound
// by the subject. The error belongs to the added name, not to the subject or
// operator declaration. Its byte position must stay stable across a CRLF file.
func TestRuleAuthoringRejectsSubjectInputRebindingAtAuthoredName(t *testing.T) {
	cases := []struct {
		name, signature, input string
	}{
		{"unnamed input", "takes number", "value"},
		{"named value", "takes value: number", "value"},
		{"named low", "takes low: number", "low"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace probe\r\noperator probe.op: " + tc.signature + "\r\n" +
				"rule probe.r: probe.x is probe.op " + tc.input + ": 10\r\n"
			ns, issues := syntax.RuleNamespace(mustParseFile(t, src))
			if len(issues) != 1 || !strings.Contains(issues[0].Msg, "already supplied by the rule subject") {
				t.Fatalf("issues=%+v, want a subject-input rebinding refusal", issues)
			}
			if len(ns.Validations) != 0 {
				t.Fatalf("the twice-bound rule was converted: %+v", ns.Validations)
			}
			start := strings.LastIndex(src, tc.input+":")
			if start < 0 || issues[0].Span.Start.Offset != start ||
				issues[0].Span.End.Offset != start+len(tc.input) {
				t.Fatalf("issue span=%+v, want input %q at byte %d",
					issues[0].Span, tc.input, start)
			}
		})
	}
}

// A rule with no explicit reassignment keeps its existing single-input meaning.
func TestRuleAuthoringKeepsSubjectAsTheOnlyOperatorInput(t *testing.T) {
	src := "namespace probe\noperator probe.op: takes value: number\n" +
		"rule probe.r: probe.x is probe.op\n"
	ns, issues := syntax.RuleNamespace(mustParseFile(t, src))
	if len(issues) != 0 || len(ns.Validations) != 1 {
		t.Fatalf("issues=%+v validations=%+v, want a single rule", issues, ns.Validations)
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
