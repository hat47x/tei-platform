package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// The unnamed `takes number` and named `takes value: number` forms both
// supply their first input from the rule subject. `value:` recognizes that
// input name, but cannot assign a second value to the same input (#4362).
//
// The expected diagnostic is different for a known input name and an
// undeclared one. Neither can emit a validation until value transport exists.
func TestRuleAuthoringNormalizesUnnamedInputBeforeRefusal(t *testing.T) {
	cases := []struct {
		name       string
		operator   string
		assignment string
		wantIssue  string
	}{
		{
			name:       "the unnamed form recognizes value before rejecting a second subject-input binding",
			operator:   "operator probe.non_negative: takes number",
			assignment: "value: 0",
			wantIssue:  "the meaning has already supplied by the rule subject yet",
		},
		{
			name:       "the named form recognizes value before rejecting a second subject-input binding",
			operator:   "operator probe.non_negative: takes value: number",
			assignment: "value: 0",
			wantIssue:  "the meaning has already supplied by the rule subject yet",
		},
		{
			name:       "the unnamed form still refuses another name",
			operator:   "operator probe.non_negative: takes number",
			assignment: "low: 0",
			wantIssue:  `declares no input named "low"`,
		},
		{
			name:       "a named form refuses its own name's sibling",
			operator:   "operator probe.within: takes low: number",
			assignment: "high: 0",
			wantIssue:  `declares no input named "high"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace probe\n\n" + tc.operator + "\n\nrule probe.r: probe.x is probe.non_negative " + tc.assignment + "\n"
			// The operator a rule names must match the one declared, so the
			// named-form cases rename the predicate's operator reference.
			if strings.Contains(tc.operator, "probe.within") {
				src = "namespace probe\n\n" + tc.operator + "\n\nrule probe.r: probe.x is probe.within " + tc.assignment + "\n"
			}
			file, err := syntax.ParseRuleAuthoringFile(src)
			if err != nil {
				t.Fatalf("%q did not parse: %v", tc.name, err)
			}
			namespace, issues := syntax.RuleNamespace(file)
			if tc.wantIssue != "" {
				if len(issues) != 1 || !strings.Contains(issues[0].Msg, tc.wantIssue) {
					t.Fatalf("issues=%+v, want %q", issues, tc.wantIssue)
				}
				if len(namespace.Validations) != 0 {
					t.Fatalf("a rejected assignment produced validations: %+v", namespace.Validations)
				}
				return
			}
			if len(issues) == 0 {
				t.Fatal("assignment was silently accepted despite an already-bound subject input")
			}
		})
	}
}

// The contract still carries only what the author wrote. Applying "value" to
// the unnamed form is a reading the rule check performs; it must not be written
// into the emitted Operator, or the contract would claim a name nobody declared.
func TestRuleAuthoringDoesNotDeclareValueForTheUnnamedForm(t *testing.T) {
	src := "namespace probe\n\noperator probe.non_negative: takes number\n\nrule probe.r: probe.x is probe.non_negative value: 0\n"
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatal(err)
	}
	namespace, issues := syntax.RuleNamespace(file)
	if len(issues) != 1 || !strings.Contains(issues[0].Msg, "the meaning has already supplied by the rule subject yet") {
		t.Fatalf("expected the subject-input rebinding refusal, issues=%+v", issues)
	}
	if len(namespace.Validations) != 0 {
		t.Fatalf("rejected assignment unexpectedly produced validations: %+v", namespace.Validations)
	}
	if len(namespace.Operators) != 1 {
		t.Fatalf("operators=%+v", namespace.Operators)
	}
	if got := namespace.Operators[0].InputName; got != "" {
		t.Fatalf("the emitted contract declares input name %q; the author wrote none", got)
	}
}
