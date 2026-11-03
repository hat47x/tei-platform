package execution

import (
	"tei/internal/core"
	"tei/internal/semantic"
)

// EffectRequirement names an effect that the invocation result cannot witness.
// An effect outside the transaction may still fail after the capability has
// returned success, so the caller has to observe it independently.
type EffectRequirement struct {
	RuleID string `json:"ruleId"`
	Stage  string `json:"stage"`
}

// EffectsOutsideTransaction lists the rules bound to a capability that run at a
// stage where the transaction no longer protects them. Comparing these with the
// invocation outcome would conflate a success code with an observation, so they
// are reported separately.
func EffectsOutsideTransaction(rules []semantic.EffectRule, capabilityID core.CanonicalID) []EffectRequirement {
	var out []EffectRequirement
	for _, rule := range rules {
		if rule.OnCapability != capabilityID {
			continue
		}
		if rule.Stage != "after-commit" && rule.Stage != "compensation" {
			continue
		}
		out = append(out, EffectRequirement{RuleID: string(rule.ID), Stage: rule.Stage})
	}
	return out
}
