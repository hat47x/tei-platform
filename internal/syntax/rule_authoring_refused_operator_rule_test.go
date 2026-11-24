package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// An operator that reads every row of a relation is converted now, so a rule
// that names it has something to be answered by. What such a rule still needs
// is the relation: without `per <relation>` nothing says which rows the rule is
// about, and the rule is refused at its own span instead of being carried as a
// rule about no rows in particular.
func TestRuleReadingEveryRowRequiresAScope(t *testing.T) {
	const src = `namespace sales

operator sales.all_lines_positive:
  takes value: number of each row

rule sales.order_line.all_positive:
  sales.order_line.line_number is sales.all_lines_positive
`
	value, issues := syntax.RuleNamespace(parseRefusedOperatorSource(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want the rule's", issues)
	}
	issue := issues[0]
	if issue.RuleID != "sales.order_line.all_positive" {
		t.Fatalf("the issue is about %s, want the rule", issue.RuleID)
	}
	if !strings.Contains(issue.Msg, "reads every row of a relation, so this rule requires a scope") {
		t.Fatalf("message=%q, want the missing scope", issue.Msg)
	}
	if issue.Span == (syntax.Anchor{}) {
		t.Fatal("the issue carries no span")
	}
	// The operator itself is converted: the rule has a contract, it is the
	// scope it does not have.
	if len(value.Operators) != 1 || value.Operators[0].ID != "sales.all_lines_positive" {
		t.Fatalf("operators=%+v, want the declared one", value.Operators)
	}
	// The rule must not survive: nothing says which rows it is about.
	if len(value.Validations) != 0 {
		t.Fatalf("a rule about no relation in particular survived as plausible meaning: %+v", value.Validations)
	}
}

// With the scope, the rule is carried with the relation it names, in the same
// field the relation rules already use.
func TestRuleReadingEveryRowCarriesItsScope(t *testing.T) {
	const src = `namespace sales

operator sales.all_lines_positive:
  takes value: number of each row
  refutes on some rows

rule sales.order_line.all_positive:
  sales.order_line.line_number is sales.all_lines_positive per sales.order.lines
`
	value, issues := syntax.RuleNamespace(parseRefusedOperatorSource(t, src))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want none", issues)
	}
	if len(value.Validations) != 1 {
		t.Fatalf("validations=%+v, want the one rule", value.Validations)
	}
	rule := value.Validations[0]
	if rule.Relation != "sales.order.lines" {
		t.Fatalf("relation=%q, want sales.order.lines", rule.Relation)
	}
	if rule.Operator != "sales.all_lines_positive" {
		t.Fatalf("operator=%q, want sales.all_lines_positive", rule.Operator)
	}
	if !value.Operators[0].RefutesOnSomeRows {
		t.Fatal("the refutation statement was not carried")
	}
}

// An operator that reads one record has no relation to name, so a scope on its
// rule is refused by name rather than carried as a relation nothing reads.
func TestRuleOfAPerRecordOperatorAcceptsNoScope(t *testing.T) {
	const src = `namespace sales

operator sales.non_negative:
  takes value: number

rule sales.order_line.quantity_non_negative:
  sales.order_line.quantity is sales.non_negative per sales.order.lines
`
	_, issues := syntax.RuleNamespace(parseRefusedOperatorSource(t, src))
	if len(issues) != 1 {
		t.Fatalf("issues=%+v, want the rule's", issues)
	}
	if issues[0].RuleID != "sales.order_line.quantity_non_negative" {
		t.Fatalf("the issue is about %s, want the rule", issues[0].RuleID)
	}
	if !strings.Contains(issues[0].Msg, "reads one record, so this rule accepts no scope") {
		t.Fatalf("message=%q, want the refused scope", issues[0].Msg)
	}
	if issues[0].Span == (syntax.Anchor{}) {
		t.Fatal("the issue carries no span")
	}
}

// A healthy rule in the same source is unaffected: the scope is refused for the
// rule that states it, not for the whole source.
func TestAHealthyRuleSurvivesARowScopedOperatorInTheSameSource(t *testing.T) {
	const src = `namespace sales

operator sales.all_lines_positive:
  takes value: number of each row

operator sales.within:
  takes value: number
  takes low: number
  takes high: number

rule sales.order_line.quantity_in_range:
  sales.order_line.quantity is sales.within low: 0 high: 100
`
	value, issues := syntax.RuleNamespace(parseRefusedOperatorSource(t, src))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v, want none", issues)
	}
	if len(value.Operators) != 2 {
		t.Fatalf("operators=%+v, want both declared ones", value.Operators)
	}
	if len(value.Validations) != 1 || value.Validations[0].ID != "sales.order_line.quantity_in_range" {
		t.Fatalf("validations=%+v, want the rule that names the per-record operator", value.Validations)
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
