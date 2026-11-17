package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// One operator signature must read the same whichever way it is written. The
// unnamed single-input form `takes number` is the contract the surface already
// calls "value", so a rule may address that input as `value:`. Before this
// check it was refused as undeclared, while the identical contract written as
// `takes value: number` accepted it (#4362).
//
// The expected verdicts below are written from the two spellings of one
// signature, not from the conversion code.
func TestRuleAuthoringReadsTheUnnamedInputAsValue(t *testing.T) {
	cases := []struct {
		name       string
		operator   string
		assignment string
		wantIssue  string
	}{
		{
			name:       "the unnamed form accepts value:",
			operator:   "operator probe.non_negative: takes number",
			assignment: "value: 0",
		},
		{
			name:       "the named form accepts value:",
			operator:   "operator probe.non_negative: takes value: number",
			assignment: "value: 0",
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
			operatorID := "probe.non_negative"
			if strings.Contains(tc.operator, "probe.within") {
				operatorID = "probe.within"
				src = "namespace probe\n\n" + tc.operator + "\n\nrule probe.r: probe.x is probe.within " + tc.assignment + "\n"
			}
			file, err := syntax.ParseRuleAuthoringFile(src)
			if err != nil {
				if tc.wantIssue != "" {
					return // refused at parse time, which is also refusing
				}
				t.Fatalf("%q did not parse: %v", tc.name, err)
			}
			namespace, issues := syntax.RuleNamespace(file)
			if tc.wantIssue != "" {
				if len(issues) != 1 || !strings.Contains(issues[0].Msg, tc.wantIssue) {
					t.Fatalf("issues=%+v, want %q", issues, tc.wantIssue)
				}
				return
			}
			if len(issues) != 0 {
				t.Fatalf("%q was refused: %+v", tc.name, issues)
			}
			if len(namespace.Validations) != 1 {
				t.Fatalf("validations=%+v", namespace.Validations)
			}
			if string(namespace.Validations[0].Operator) != operatorID {
				t.Fatalf("operator=%q", namespace.Validations[0].Operator)
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
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if len(namespace.Operators) != 1 {
		t.Fatalf("operators=%+v", namespace.Operators)
	}
	if got := namespace.Operators[0].InputName; got != "" {
		t.Fatalf("the emitted contract declares input name %q; the author wrote none", got)
	}
}
