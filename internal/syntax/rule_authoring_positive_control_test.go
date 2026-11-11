package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/validation"
)

// Positive controls for the rejection table.
//
// A rejection test alone cannot tell a strict parser from one that refuses
// everything. Each row here is the *same* source with exactly one factor
// repaired, and it must be accepted. This is the same shape #988 §13.8 used
// for the bootstrap cases: a negative input, its expected refusal, the single
// repaired delta, and the expected acceptance.
//
// This is not a mutation run. It shows the mutant "always reject" would be
// detected; it does not measure a mutation score.

func TestRuleSurfaceRejectionsHaveSingleFactorRepairs(t *testing.T) {
	cases := []struct {
		name       string
		broken     string
		brokeWhy   string
		repaired   string
		repairNote string
	}{
		{
			name:       "P-04 unknown predicate word",
			broken:     "namespace sales\nrule sales.q: a.b is positive-ish\n",
			brokeWhy:   "unknown rule predicate",
			repaired:   "namespace sales\nrule sales.q: a.b is positive\n",
			repairNote: "the predicate word is the only change",
		},
		{
			name:       "P-05 foreign namespace identity",
			broken:     "namespace sales\nrule customer.q: sales.order_line.quantity is positive\n",
			brokeWhy:   "must belong to the declared namespace",
			repaired:   "namespace sales\nrule sales.q: sales.order_line.quantity is positive\n",
			repairNote: "the declaration name is the only change",
		},
		{
			name:       "P-07 unique without scope",
			broken:     "namespace sales\nrule sales.q: a.b is unique\n",
			brokeWhy:   `requires "per <relation>"`,
			repaired:   "namespace sales\nrule sales.q: a.b is unique per a.lines\n",
			repairNote: "the scope is the only addition",
		},
		{
			name:       "P-08 positive with scope",
			broken:     "namespace sales\nrule sales.q: a.b is positive per a.lines\n",
			brokeWhy:   "accepts no scope",
			repaired:   "namespace sales\nrule sales.q: a.b is positive\n",
			repairNote: "the extra scope is the only removal",
		},
		{
			name:       "subject is a clause word",
			broken:     "namespace sales\nrule sales.q: is positive\n",
			brokeWhy:   "cannot be a rule subject",
			repaired:   "namespace sales\nrule sales.q: a.b is positive\n",
			repairNote: "the subject is the only addition",
		},
		{
			name:       "missing is",
			broken:     "namespace sales\nrule sales.q: a.b positive\n",
			brokeWhy:   `expected "is" after the subject reference`,
			repaired:   "namespace sales\nrule sales.q: a.b is positive\n",
			repairNote: `the keyword "is" is the only addition`,
		},
		{
			name:       "trailing declaration keyword",
			broken:     "namespace sales\nrule sales.q: a.b is positive\nattribute x\n",
			brokeWhy:   `unexpected "attribute" after the rule body`,
			repaired:   "namespace sales\nrule sales.q: a.b is positive\n",
			repairNote: "the stray line is the only removal",
		},
		{
			name:       "duplicate rule identity",
			broken:     "namespace sales\nrule sales.q: a.b is positive\nrule sales.q: c.d is positive\n",
			brokeWhy:   "is declared more than once",
			repaired:   "namespace sales\nrule sales.q: a.b is positive\nrule sales.r: c.d is positive\n",
			repairNote: "the second identity is the only change",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ParseRuleAuthoringFile(tc.broken)
			if err == nil {
				t.Fatalf("the broken source was accepted")
			}
			msg, ok := syntax.ErrorMessage(err)
			if !ok || !strings.Contains(msg, tc.brokeWhy) {
				t.Fatalf("refusal reason=%q want %q", msg, tc.brokeWhy)
			}

			// The repaired source must parse. If this fails, the rejection test
			// above proves nothing about the surface, only that it refuses.
			file, err := syntax.ParseRuleAuthoringFile(tc.repaired)
			if err != nil {
				t.Fatalf("the one-factor repair %q was rejected (%s): %v", tc.repairNote, tc.repairNote, err)
			}
			if len(file.Rules) == 0 {
				t.Fatalf("the repair parsed no rule")
			}
		})
	}
}

// The same shape for the conversion layer: a bare reference is refused, and
// qualifying it is the single repair that makes the rule reach the validator.
func TestRuleConversionRefusalHasSingleFactorRepair(t *testing.T) {
	const broken = "namespace sales\nrule sales.q: quantity is positive\n"
	const repaired = "namespace sales\nrule sales.q: sales.order_line.quantity is positive\n"

	file, err := syntax.ParseRuleAuthoringFile(broken)
	if err != nil {
		t.Fatalf("parse broken: %v", err)
	}
	if _, issues := syntax.RuleNamespace(file); len(issues) == 0 {
		t.Fatal("the bare reference was converted instead of refused")
	}

	repairedFile, err := syntax.ParseRuleAuthoringFile(repaired)
	if err != nil {
		t.Fatalf("parse repaired: %v", err)
	}
	ns, issues := syntax.RuleNamespace(repairedFile)
	if len(issues) != 0 {
		t.Fatalf("the qualified reference was still refused: %v", issues)
	}
	if len(ns.Validations) != 1 || ns.Validations[0].Attribute != "sales.order_line.quantity" {
		t.Fatalf("validations=%#v", ns.Validations)
	}
}

// And for the validator layer: the type mismatch is repaired by choosing a
// numeric attribute, not by weakening the rule kind.
func TestRuleValidatorRefusalHasSingleFactorRepair(t *testing.T) {
	const broken = "namespace sales\nrule sales.order_line.quantity_positive:\n  sales.order.order_id is positive\n"
	const repaired = "namespace sales\nrule sales.order_line.quantity_positive:\n  sales.order_line.quantity is positive\n"

	diagnostics := validation.Validate(ruleValidatorWorld(mustRuleNamespace(t, broken).Validations...))
	found := false
	for _, d := range diagnostics {
		if d.Code == "SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the non-numeric subject was accepted: %v", diagnostics)
	}

	for _, d := range validation.Validate(ruleValidatorWorld(mustRuleNamespace(t, repaired).Validations...)) {
		if d.Severity == validation.Error {
			t.Fatalf("the numeric subject was still refused: %s %s", d.Code, d.Message)
		}
	}
	// The kind is unchanged across the repair: the author fixed the subject, not
	// the rule. A repair that switched to a laxer kind would be a different rule.
	if mustRuleNamespace(t, repaired).Validations[0].Kind != semantic.ValidationPositive {
		t.Fatal("the repair changed the rule kind instead of the subject")
	}
}
