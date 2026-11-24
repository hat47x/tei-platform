package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// An operator the conversion refuses contributes no contract, so a rule naming
// it has no operator to be answered by. The rule used to be converted anyway:
// one document holding a statement the meaning could not carry produced one
// issue for the operator and
// still handed a validation naming an undeclared operator to the validator,
// which reports suspicions rather than a missing meaning. The product loader
// only notices "a declaration of this region produced no meaning" when the
// counts differ, so it saw nothing (#4359).
func TestRuleNamingARefusedOperatorIsRefusedWithIt(t *testing.T) {
	const src = `namespace sales

operator sales.all_lines_positive:
  takes value: number
  refutes on some rows

rule sales.order_line.all_positive:
  sales.order_line.line_number is sales.all_lines_positive
`
	value, issues := syntax.RuleNamespace(parseRefusedOperatorSource(t, src))
	if len(issues) != 2 {
		t.Fatalf("issues=%d, want one for the operator and one for the rule that depends on it: %+v", len(issues), issues)
	}
	if len(value.Operators) != 0 {
		t.Fatalf("a refused operator produced a contract: %+v", value.Operators)
	}
	// The rule must not survive: nothing downstream can answer it.
	if len(value.Validations) != 0 {
		t.Fatalf("a rule naming a refused operator survived as plausible meaning: %+v", value.Validations)
	}
	var sawOperator, sawRule bool
	for _, issue := range issues {
		if issue.RuleID == "sales.all_lines_positive" && strings.Contains(issue.Msg, "declares no input read over all rows") {
			sawOperator = true
		}
		if issue.RuleID == "sales.order_line.all_positive" &&
			strings.Contains(issue.Msg, "sales.all_lines_positive") &&
			strings.Contains(issue.Msg, "could not convert") {
			sawRule = true
		}
		if issue.Span == (syntax.Anchor{}) {
			t.Fatalf("issue %q carries no span", issue.Msg)
		}
	}
	if !sawOperator || !sawRule {
		t.Fatalf("the two issues are not the operator's and the rule's: %+v", issues)
	}
}

// A rule that names a different, healthy operator is unaffected: this refuses
// the dependency, not the whole source.
func TestAHealthyRuleSurvivesARefusedOperatorInTheSameSource(t *testing.T) {
	const src = `namespace sales

operator sales.all_lines_positive:
  takes value: number
  refutes on some rows

operator sales.within:
  takes value: number
  takes low: number
  takes high: number

rule sales.order_line.quantity_in_range:
  sales.order_line.quantity is sales.within low: 0 high: 100
`
	value, issues := syntax.RuleNamespace(parseRefusedOperatorSource(t, src))
	if len(issues) != 1 || !strings.Contains(issues[0].Msg, "declares no input read over all rows") {
		t.Fatalf("issues=%+v, want exactly the operator's", issues)
	}
	if len(value.Operators) != 1 || value.Operators[0].ID != "sales.within" {
		t.Fatalf("operators=%+v, want the healthy one", value.Operators)
	}
	if len(value.Validations) != 1 || value.Validations[0].ID != "sales.order_line.quantity_in_range" {
		t.Fatalf("validations=%+v, want the rule that names the healthy operator", value.Validations)
	}
}

func parseRefusedOperatorSource(t *testing.T, src string) syntax.RuleAuthoringFile {
	t.Helper()
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("cannot parse the source: %v", err)
	}
	return file
}
