package validation_test

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func effectRuleWorld(rule semantic.EffectRule) ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{
			{ID: "sales.order"},
			{ID: "sales.stock"},
		},
		Capabilities: []semantic.Capability{
			{ID: "sales.order.submit"},
			{ID: "sales.stock.decrement"},
		},
		EffectRules: []semantic.EffectRule{rule},
	}}}
}

func codesOf(diagnostics []validation.Diagnostic) map[string]bool {
	codes := map[string]bool{}
	for _, diagnostic := range diagnostics {
		codes[diagnostic.Code] = true
	}
	return codes
}

func TestEffectRuleAcceptsADeclaredRule(t *testing.T) {
	world := effectRuleWorld(semantic.EffectRule{
		ID:            "sales.order.stock",
		OnCapability:  "sales.order.submit",
		Stage:         "after-commit",
		EffectKind:    "update",
		EffectData:    "sales.stock",
		ViaCapability: "sales.stock.decrement",
		OnFailure:     "compensate",
	})
	if diagnostics := validation.Validate(world); len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics: %+v", diagnostics)
	}
}

func TestEffectRuleRejectsUnknownParts(t *testing.T) {
	world := effectRuleWorld(semantic.EffectRule{
		ID:           "sales.order.stock",
		OnCapability: "sales.order.missing",
		Stage:        "later",
		EffectKind:   "update",
		EffectData:   "sales.stock",
		OnFailure:    "ignore",
	})
	codes := codesOf(validation.Validate(world))
	for _, want := range []string{"EFFECT_RULE_UNKNOWN_CAPABILITY", "EFFECT_RULE_UNKNOWN_STAGE", "EFFECT_RULE_UNKNOWN_FAILURE"} {
		if !codes[want] {
			t.Fatalf("expected %s: %+v", want, codes)
		}
	}
}

func TestEffectRuleRequiresTheCapabilityForLateStages(t *testing.T) {
	world := effectRuleWorld(semantic.EffectRule{
		ID:           "sales.order.stock",
		OnCapability: "sales.order.submit",
		Stage:        "after-commit",
		EffectKind:   "update",
		EffectData:   "sales.stock",
	})
	if !codesOf(validation.Validate(world))["EFFECT_RULE_REQUIRES_VIA"] {
		t.Fatalf("a late stage must name the capability that performs it: %+v", validation.Validate(world))
	}
}
