package hook_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// ruleWorld declares a record pattern whose hook lists the reads the reference
// rule evaluates, so the rule runs through the Runner rather than being called
// directly.
func ruleWorld(failure string) ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Hooks: []application.Hook{{
						Phase:   "before-update",
						Reads:   []core.CanonicalID{"sales.order.quantity", "sales.order.status"},
						Failure: failure,
					}},
				},
			}},
		}},
	}
}

func ruleInvocation(reads map[core.CanonicalID]string) hook.Invocation {
	return hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
		Reads:       reads,
	}
}

func ruleRunner(failure string, rule hook.ReferenceRule) hook.Runner {
	return hook.Runner{World: ruleWorld(failure), Implementation: rule}
}

// TestReferenceRuleAcceptsObservedPositiveValue checks that a satisfied rule
// returns no error through the hook boundary.
func TestReferenceRuleAcceptsObservedPositiveValue(t *testing.T) {
	rule := hook.ReferenceRule{ID: "quantity-positive", Positive: []core.CanonicalID{"sales.order.quantity"}}
	if err := ruleRunner("reject", rule).Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{
		"sales.order.quantity": "2",
	})); err != nil {
		t.Fatalf("err=%v", err)
	}
}

// TestReferenceRuleRefusesAnUnobservedAttribute checks that silence is not a
// satisfied condition: a declared read that was not observed is refused.
func TestReferenceRuleRefusesAnUnobservedAttribute(t *testing.T) {
	rule := hook.ReferenceRule{ID: "status-required", Required: []core.CanonicalID{"sales.order.status"}}
	err := ruleRunner("reject", rule).Run(context.Background(), ruleInvocation(nil))
	var notSatisfied *hook.NotSatisfied
	if !errors.As(err, &notSatisfied) {
		t.Fatalf("err=%v", err)
	}
	if notSatisfied.Outcome != "not-satisfied" {
		t.Fatalf("outcome=%s", notSatisfied.Outcome)
	}
}

// TestReferenceRuleRefusesANonPositiveValue checks that the rule names the
// business refusal, not a generic rejection.
func TestReferenceRuleRefusesANonPositiveValue(t *testing.T) {
	rule := hook.ReferenceRule{ID: "quantity-positive", Positive: []core.CanonicalID{"sales.order.quantity"}}
	err := ruleRunner("reject", rule).Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{
		"sales.order.quantity": "0",
	}))
	var notSatisfied *hook.NotSatisfied
	if !errors.As(err, &notSatisfied) {
		t.Fatalf("err=%v", err)
	}
	if notSatisfied.Outcome != "value-rejected" {
		t.Fatalf("outcome=%s", notSatisfied.Outcome)
	}
}

// TestReferenceRuleIsDeterministic checks that the same input yields the same
// refusal, so a caller can compare runs.
func TestReferenceRuleIsDeterministic(t *testing.T) {
	rule := hook.ReferenceRule{
		ID:       "quantity-positive",
		Required: []core.CanonicalID{"sales.order.status"},
		Positive: []core.CanonicalID{"sales.order.quantity"},
	}
	reads := map[core.CanonicalID]string{"sales.order.quantity": "-1"}
	first := ruleRunner("reject", rule).Run(context.Background(), ruleInvocation(reads))
	second := ruleRunner("reject", rule).Run(context.Background(), ruleInvocation(reads))
	if first == nil || second == nil || first.Error() != second.Error() {
		t.Fatalf("first=%v second=%v", first, second)
	}
}

// TestReferenceRuleReportsTheSameOrderForSeveralRefusals checks that attribute
// order, not declaration order or map iteration, decides which refusal is
// reported first.
func TestReferenceRuleReportsTheSameOrderForSeveralRefusals(t *testing.T) {
	rule := hook.ReferenceRule{
		ID:       "two-missing",
		Required: []core.CanonicalID{"sales.order.status", "sales.order.quantity"},
	}
	err := ruleRunner("reject", rule).Run(context.Background(), ruleInvocation(nil))
	if err == nil {
		t.Fatal("two missing reads were accepted")
	}
	var notSatisfied *hook.NotSatisfied
	if !errors.As(err, &notSatisfied) {
		t.Fatalf("err=%v", err)
	}
	if notSatisfied.Reason != "attribute sales.order.quantity was not observed" {
		t.Fatalf("reason=%s", notSatisfied.Reason)
	}
}

// TestReferenceRuleIsNotAWarn checks that on-failure warn does not soften a
// business refusal from the reference rule.
func TestReferenceRuleIsNotAWarn(t *testing.T) {
	rule := hook.ReferenceRule{ID: "quantity-positive", Positive: []core.CanonicalID{"sales.order.quantity"}}
	err := ruleRunner("warn", rule).Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{
		"sales.order.quantity": "0",
	}))
	var warning *hook.Warning
	if errors.As(err, &warning) {
		t.Fatalf("a business refusal was reported as a warning: %v", warning)
	}
	var notSatisfied *hook.NotSatisfied
	if !errors.As(err, &notSatisfied) {
		t.Fatalf("err=%v", err)
	}
}

// TestReferenceRuleIgnoresUndeclaredReads checks that the rule evaluates only
// the attributes it declares, so a value the rule does not own is not judged.
func TestReferenceRuleIgnoresUndeclaredReads(t *testing.T) {
	rule := hook.ReferenceRule{ID: "quantity-positive", Positive: []core.CanonicalID{"sales.order.quantity"}}
	err := ruleRunner("reject", rule).Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{
		"sales.order.quantity": "5",
		"sales.order.status":   "unknown",
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
}

// TestReferenceRuleRefusesANonNumber checks that a value that cannot be read as
// a number is a business refusal with its cause kept.
func TestReferenceRuleRefusesANonNumber(t *testing.T) {
	rule := hook.ReferenceRule{ID: "quantity-positive", Positive: []core.CanonicalID{"sales.order.quantity"}}
	err := ruleRunner("reject", rule).Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{
		"sales.order.quantity": "many",
	}))
	var notSatisfied *hook.NotSatisfied
	if !errors.As(err, &notSatisfied) || notSatisfied.Outcome != "value-rejected" {
		t.Fatalf("err=%v", err)
	}
	if notSatisfied.Cause == nil {
		t.Fatal("the parse cause was lost")
	}
}
