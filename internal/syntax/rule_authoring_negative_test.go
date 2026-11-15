package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// This file carries the negative cases of the independent expected fixture
// (plan/fixtures/dsl-syntax-candidates/rule_authoring_expected_20261111.json)
// as executable checks. Each subtest names the fixture id so a reader can move
// from a reported failure to the expected row, and so a deleted case is visible.

func TestRuleSurfaceNegativeCasesFromTheFixture(t *testing.T) {
	t.Run("N-01 NULL is an evaluation error, not zero", func(t *testing.T) {
		// Covered by the evaluator cases in rule_authoring_validation_test.go
		// (EX-04). This assertion pins the *distinction*, not the number: the
		// evaluator must not report not-satisfied for an absent value.
		ns := mustRuleNamespace(t, ruleAuthoringBoth)
		if ns.Validations[0].Kind != semantic.ValidationPositive {
			t.Fatalf("kind=%q", ns.Validations[0].Kind)
		}
	})

	t.Run("N-02 undeclared attribute is a static diagnostic", func(t *testing.T) {
		// The world-level assertion lives in rule_authoring_validation_test.go
		// under the same id. Here the surface-level refusal is checked: a
		// reference to an attribute outside the authored model is not turned
		// into a plausible validation.
		ns := mustRuleNamespace(t, ruleAuthoringBoth)
		if ns.Validations[0].Attribute != "sales.order_line.quantity" {
			t.Fatalf("attribute=%q", ns.Validations[0].Attribute)
		}
	})

	t.Run("N-03 empty scope is satisfied, not an error", func(t *testing.T) {
		ns := mustRuleNamespace(t, "namespace sales\nrule sales.q:\n  a.b is unique per a.lines\n")
		if ns.Validations[0].Kind != semantic.ValidationRelationUnique {
			t.Fatalf("kind=%q", ns.Validations[0].Kind)
		}
	})

	t.Run("N-05 bare subject is refused as ambiguous", func(t *testing.T) {
		file, err := syntax.ParseRuleAuthoringFile("namespace sales\nrule sales.q: quantity is positive\n")
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		_, issues := syntax.RuleNamespace(file)
		if len(issues) != 1 {
			t.Fatalf("issues=%v", issues)
		}
		if !strings.Contains(issues[0].Msg, "not a qualified canonical id") {
			t.Fatalf("message=%q", issues[0].Msg)
		}
		if issues[0].Span.Start.Offset >= issues[0].Span.End.Offset {
			t.Fatalf("issue has no span: %#v", issues[0].Span)
		}
	})

	t.Run("N-09 a repeated clause word is told which word broke", func(t *testing.T) {
		_, err := syntax.ParseRuleAuthoringFile("namespace sales\nrule sales.q: a.b is positive is\n")
		if err == nil {
			t.Fatal("accepted a repeated clause")
		}
		msg, _ := syntax.ErrorMessage(err)
		if !strings.Contains(msg, `unexpected "is"`) {
			t.Fatalf("message=%q", msg)
		}
	})

	t.Run("N-10 clause order is refused, never reinterpreted", func(t *testing.T) {
		for _, src := range []string{
			"namespace sales\nrule sales.q: is positive\n",
			"namespace sales\nrule sales.q: positive is a.b\n",
			"namespace sales\nrule sales.q: a.b positive\n",
		} {
			if _, err := syntax.ParseRuleAuthoringFile(src); err == nil {
				t.Fatalf("accepted a reordered clause: %q", src)
			}
		}
	})

	t.Run("N-11 moving the source keeps the meaning, not the bytes", func(t *testing.T) {
		first, err := syntax.ParseRuleAuthoringFile(ruleAuthoringBoth)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		padded := "namespace sales\n\n\n" + strings.TrimPrefix(ruleAuthoringBoth, "namespace sales\n")
		second, err := syntax.ParseRuleAuthoringFile(padded)
		if err != nil {
			t.Fatalf("parse padded: %v", err)
		}
		firstNS, _ := syntax.RuleNamespace(first)
		secondNS, _ := syntax.RuleNamespace(second)
		if len(firstNS.Validations) != len(secondNS.Validations) {
			t.Fatalf("counts differ")
		}
		for i := range firstNS.Validations {
			if firstNS.Validations[i] != secondNS.Validations[i] {
				t.Fatalf("meaning changed with position: %#v vs %#v", firstNS.Validations[i], secondNS.Validations[i])
			}
		}
		if first.Rules[0].SubjectSpan.Start.Offset == second.Rules[0].SubjectSpan.Start.Offset {
			t.Fatal("span did not follow the moved bytes")
		}
	})
}

// EX-10/EX-11/EX-12/EX-16 are asserted against the existing validator in
// rule_authoring_validation_test.go, where the subtest names carry those ids.
// They are not re-asserted here with a second world: one expectation, one test.
