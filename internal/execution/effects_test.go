package execution_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/semantic"
)

func TestEffectsOutsideTransactionSelectsLateStages(t *testing.T) {
	rules := []semantic.EffectRule{
		{ID: "s.order.audit", OnCapability: "s.order.submit", Stage: "in-transaction", EffectKind: "create", EffectData: "s.order"},
		{ID: "s.order.stock", OnCapability: "s.order.submit", Stage: "after-commit", EffectKind: "update", EffectData: "s.stock", ViaCapability: "s.stock.decrement"},
		{ID: "s.order.undo", OnCapability: "s.order.submit", Stage: "compensation", EffectKind: "update", EffectData: "s.stock", ViaCapability: "s.stock.restore"},
		{ID: "s.other.stock", OnCapability: "s.other.submit", Stage: "after-commit", EffectKind: "update", EffectData: "s.stock", ViaCapability: "s.stock.decrement"},
	}

	requirements := execution.EffectsOutsideTransaction(rules, core.MustCanonicalID("s.order.submit"))
	if len(requirements) != 2 {
		t.Fatalf("only the late stages of the bound capability need observation: %+v", requirements)
	}
	if requirements[0].RuleID != "s.order.stock" || requirements[0].Stage != "after-commit" {
		t.Fatalf("unexpected first requirement: %+v", requirements[0])
	}
	if requirements[1].RuleID != "s.order.undo" || requirements[1].Stage != "compensation" {
		t.Fatalf("unexpected second requirement: %+v", requirements[1])
	}
}
