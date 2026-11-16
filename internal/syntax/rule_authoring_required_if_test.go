package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// The conditional requirement is written as words a domain expert reads:
//
//	<attribute> is required if <attribute>
//
// This file pins the words, the conversion to the existing validation kind,
// and the refusals. The expected values are written by hand.

const requiredIfSource = `namespace sales
rule sales.order_line.discount_reason_required:
  sales.order_line.discount_reason is required if sales.order_line.discount
`

func TestRequiredIfParsesAndConvertsToTheExistingKind(t *testing.T) {
	file, err := syntax.ParseRuleAuthoringFile(requiredIfSource)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(file.Rules) != 1 {
		t.Fatalf("rules=%d", len(file.Rules))
	}
	rule := file.Rules[0]
	if rule.Predicate != syntax.RulePredicateRequiredIf ||
		rule.Subject.Text != "sales.order_line.discount_reason" ||
		rule.Compared == nil || rule.Compared.Text != "sales.order_line.discount" ||
		rule.IfPresent || rule.Scope != nil || rule.Operator != nil || rule.PeriodEnd != nil {
		t.Fatalf("parsed rule=%+v", rule)
	}
	// The subject span points at the attribute the author wrote.
	if got := requiredIfSource[rule.SubjectSpan.Start.Offset:rule.SubjectSpan.End.Offset]; got != "sales.order_line.discount_reason" {
		t.Fatalf("subject span=%q", got)
	}

	namespace, issues := syntax.RuleNamespace(file)
	if len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
	if len(namespace.Validations) != 1 {
		t.Fatalf("validations=%d", len(namespace.Validations))
	}
	got := namespace.Validations[0]
	want := semantic.Validation{
		ID:        "sales.order_line.discount_reason_required",
		Symbol:    "discount_reason_required",
		Kind:      semantic.ValidationRequiredIf,
		Attribute: "sales.order_line.discount_reason",
		Compared:  "sales.order_line.discount",
	}
	if got != want {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
	if !got.Kind.Valid() {
		t.Fatal("the kind is not accepted by the model")
	}
}

// The words taught the viewer earlier must keep their meaning. This pins the
// three spellings that are close but mean something else.
func TestRequiredIfRefusesTheNearMissSpellings(t *testing.T) {
	cases := []struct{ name, source, mention string }{
		{"a requirement with no condition",
			`namespace sales
rule sales.order_line.reason_required:
  sales.order_line.discount_reason is required
`, "expected \"if\" after \"required\""},
		{"a condition with no attribute",
			`namespace sales
rule sales.order_line.reason_required:
  sales.order_line.discount_reason is required if
`, "expected the attribute whose value makes the subject required"},
		{"the word order reversed",
			`namespace sales
rule sales.order_line.reason_required:
  sales.order_line.discount_reason is if required sales.order_line.discount
`, "unknown rule predicate \"if\""},
		{"a scope on a rule about one record",
			`namespace sales
rule sales.order_line.reason_required:
  sales.order_line.discount_reason is required if sales.order_line.discount per sales.order.lines
`, "accepts no scope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ParseRuleAuthoringFile(tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.mention) {
				t.Fatalf("err=%v, want a mention of %q", err, tc.mention)
			}
		})
	}
}

// `if present` belongs to the two-date rule. After a rule about presence the
// words mean nothing the rule says, so they are refused where they are written
// rather than carried into the meaning model and dropped there.
func TestRequiredIfRefusesIfPresent(t *testing.T) {
	const source = `namespace sales
rule sales.order_line.discount_reason_required:
  sales.order_line.discount_reason is required if sales.order_line.discount if present
`
	_, err := syntax.ParseRuleAuthoringFile(source)
	if err == nil || !strings.Contains(err.Error(), "unexpected \"if\" after the rule body") {
		t.Fatalf("err=%v, want a refusal of the trailing if present", err)
	}
	// The same rule without the words is accepted, so the refusal is about the
	// words and not about the rest of the declaration.
	file, err := syntax.ParseRuleAuthoringFile(strings.Replace(source, " if present", "", 1))
	if err != nil {
		t.Fatalf("the rule without the words was refused: %v", err)
	}
	if _, issues := syntax.RuleNamespace(file); len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
}

// The predicate word is not the kind word. Pinning both prevents a later edit
// from making one spelling quietly mean the other.
func TestRequiredIfWordsAreFixed(t *testing.T) {
	if string(syntax.RulePredicateRequiredIf) != "required-if" {
		t.Fatalf("preposition word changed: %q", syntax.RulePredicateRequiredIf)
	}
	if string(semantic.ValidationRequiredIf) != "required-if" {
		t.Fatalf("kind word changed: %q", semantic.ValidationRequiredIf)
	}
	kind, ok := syntax.RuleValidationKind(syntax.RulePredicateRequiredIf)
	if !ok || kind != semantic.ValidationRequiredIf {
		t.Fatalf("predicate maps to %q (%v)", kind, ok)
	}
}
