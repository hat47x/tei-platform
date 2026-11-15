package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/syntax"
)

// The formatter is an explicit operation. These cases pin what it may and may
// not do. Nothing here asserts that a save path calls it: no writer exists in
// this slice, and the zero-write guarantee belongs to #4366.

func TestFormatRuleAuthoringCanonicalLayout(t *testing.T) {
	const src = "namespace sales\nrule sales.order_line.quantity_positive:   sales.order_line.quantity is positive\n\nrule   sales.order.lines.line_number_unique:\n    sales.order_line.line_number is unique per sales.order.lines\n"
	out, err := syntax.FormatRuleAuthoring(src)
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	want := `namespace sales

rule sales.order_line.quantity_positive:
  sales.order_line.quantity is positive

rule sales.order.lines.line_number_unique:
  sales.order_line.line_number is unique per sales.order.lines
`
	if out != want {
		t.Fatalf("formatted:\n%s\nwant:\n%s", out, want)
	}
}

func TestFormatRuleAuthoringIsIdempotent(t *testing.T) {
	const src = "namespace sales\nrule sales.q:\n  a.b is positive\n"
	once, err := syntax.FormatRuleAuthoring(src)
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	twice, err := syntax.FormatRuleAuthoring(once)
	if err != nil {
		t.Fatalf("format formatted: %v", err)
	}
	if once != twice {
		t.Fatalf("not idempotent:\n%q\n%q", once, twice)
	}
}

// A comment is not relocated: the formatter refuses rather than moving text the
// author placed deliberately.
func TestFormatRuleAuthoringRefusesComments(t *testing.T) {
	var target error
	for _, src := range []string{
		"namespace sales\n# explain the rule\nrule sales.q:\n  a.b is positive\n",
		"namespace sales\nrule sales.q:\n  a.b is positive // trailing\n",
	} {
		_, err := syntax.FormatRuleAuthoring(src)
		if !errors.Is(err, syntax.ErrRuleFormatUnsupported) {
			t.Fatalf("comment source was reformatted or failed differently: %v", err)
		}
		_ = target
	}
}

// Formatting must not rename: a mixed-case authored identity keeps its spelling,
// even though its canonical ID is lower-cased.
func TestFormatRuleAuthoringKeepsAuthoredSpelling(t *testing.T) {
	const src = "namespace sales\nrule OrderQuantityPositive:\n  a.b is positive\n"
	out, err := syntax.FormatRuleAuthoring(src)
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	if !strings.Contains(out, "rule OrderQuantityPositive:") {
		t.Fatalf("identity was renamed:\n%s", out)
	}
	// The canonical identity is still the derived one.
	file, err := syntax.ParseRuleAuthoringFile(out)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if file.Rules[0].ID != "sales.order_quantity_positive" {
		t.Fatalf("id=%q", file.Rules[0].ID)
	}
}

// A source whose references do not resolve has no canonical rendering: tidying
// it would leave the author with a nicer but still broken declaration.
func TestFormatRuleAuthoringRefusesUnresolvedReferences(t *testing.T) {
	_, err := syntax.FormatRuleAuthoring("namespace sales\nrule sales.q:\n  bare is positive\n")
	if !errors.Is(err, syntax.ErrRuleFormatUnsupported) {
		t.Fatalf("err=%v", err)
	}
}

// The formatter verifies its own output. A rendering that would change the
// parsed value is refused instead of emitted.
func TestFormatRuleAuthoringChecksMeaningBeforeReturning(t *testing.T) {
	out, err := syntax.FormatRuleAuthoring(ruleAuthoringBoth)
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	original, err := syntax.ParseRuleAuthoringAnchored(ruleAuthoringBoth)
	if err != nil {
		t.Fatalf("parse original: %v", err)
	}
	formatted, err := syntax.ParseRuleAuthoringAnchored(out)
	if err != nil {
		t.Fatalf("parse formatted: %v", err)
	}
	if len(original.Value.Validations) != len(formatted.Value.Validations) {
		t.Fatalf("counts differ")
	}
	for i := range original.Value.Validations {
		if original.Value.Validations[i] != formatted.Value.Validations[i] {
			t.Fatalf("meaning changed:\n%#v\n%#v", original.Value.Validations[i], formatted.Value.Validations[i])
		}
	}
}

// The comparison direction is useful for migration preview. It must express the
// existing validation, and must refuse a kind the surface cannot write.
func TestRuleSurfaceFromValidationRoundTripsExistingValues(t *testing.T) {
	cases := []semantic.Validation{
		{
			ID:        core.MustCanonicalID("sales.order_line.quantity_positive"),
			Kind:      semantic.ValidationPositive,
			Attribute: core.MustCanonicalID("sales.order_line.quantity"),
		},
		{
			ID:        core.MustCanonicalID("sales.order.lines.line_number_unique"),
			Kind:      semantic.ValidationRelationUnique,
			Attribute: core.MustCanonicalID("sales.order_line.line_number"),
			Relation:  core.MustCanonicalID("sales.order.lines"),
		},
	}
	for _, validation := range cases {
		out, err := syntax.RuleSurfaceFromValidation("sales", validation)
		if err != nil {
			t.Fatalf("%s: %v", validation.ID, err)
		}
		file, err := syntax.ParseRuleAuthoringFile(out)
		if err != nil {
			t.Fatalf("%s: rendering does not parse: %v", validation.ID, err)
		}
		value, issues := syntax.RuleNamespace(file)
		if len(issues) != 0 {
			t.Fatalf("%s: issues=%v", validation.ID, issues)
		}
		got := value.Validations[0]
		if got.ID != validation.ID || got.Kind != validation.Kind ||
			got.Attribute != validation.Attribute || got.Relation != validation.Relation {
			t.Fatalf("%s: rendering changed the meaning: %#v", validation.ID, got)
		}
	}
}

func TestRuleSurfaceFromValidationRefusesUnknownKind(t *testing.T) {
	unknown := semantic.Validation{ID: "sales.q", Kind: semantic.ValidationKind("expression"), Attribute: "sales.a.b"}
	_, err := syntax.RuleSurfaceFromValidation("sales", unknown)
	if !errors.Is(err, syntax.ErrRuleFormatUnsupported) {
		t.Fatalf("err=%v", err)
	}
}
