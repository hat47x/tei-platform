package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// effectRuleStages are the points around a capability where an effect may run.
var effectRuleStages = map[string]bool{
	"derivation":     true,
	"validation":     true,
	"pre-commit":     true,
	"in-transaction": true,
	"after-commit":   true,
	"compensation":   true,
}

// effectRuleFailures are the ways a failed effect may be handled.
var effectRuleFailures = map[string]bool{
	"reject":     true,
	"compensate": true,
}

// validateSemanticEffectRules validates the declared meaning of an effect rule:
// the capability it runs on, the stage, the data it changes, the capability it
// delegates to, and the failure handling. It does not check that the effect ran.
func validateSemanticEffectRules(w ir.World) []Diagnostic {
	capabilities := map[core.CanonicalID]bool{}
	data := map[core.CanonicalID]bool{}
	for _, namespace := range w.Semantics {
		for _, capability := range namespace.Capabilities {
			capabilities[capability.ID] = true
		}
		for _, element := range namespace.Data {
			data[element.ID] = true
		}
	}

	var out []Diagnostic
	for _, namespace := range w.Semantics {
		for _, rule := range namespace.EffectRules {
			subj := subject.Subject{Kind: "semantic-effect-rule", ID: string(rule.ID)}

			if !capabilities[rule.OnCapability] {
				out = append(out, errFor(subj, "EFFECT_RULE_UNKNOWN_CAPABILITY", "effect rule %s runs on unknown capability %s", rule.ID, rule.OnCapability))
			}
			if !effectRuleStages[rule.Stage] {
				out = append(out, errFor(subj, "EFFECT_RULE_UNKNOWN_STAGE", "effect rule %s declares unknown stage %q", rule.ID, rule.Stage))
			}
			if !data[rule.EffectData] {
				out = append(out, errFor(subj, "EFFECT_RULE_UNKNOWN_DATA", "effect rule %s changes unknown data %s", rule.ID, rule.EffectData))
			}
			if rule.ViaCapability != "" && !capabilities[rule.ViaCapability] {
				out = append(out, errFor(subj, "EFFECT_RULE_UNKNOWN_VIA", "effect rule %s delegates to unknown capability %s", rule.ID, rule.ViaCapability))
			}
			if rule.OnFailure != "" && !effectRuleFailures[rule.OnFailure] {
				out = append(out, errFor(subj, "EFFECT_RULE_UNKNOWN_FAILURE", "effect rule %s declares unknown failure handling %q", rule.ID, rule.OnFailure))
			}
			// An effect outside the transaction cannot be rolled back with it, so
			// it must name the capability that performs and reports it.
			if (rule.Stage == "after-commit" || rule.Stage == "compensation") && rule.ViaCapability == "" {
				out = append(out, errFor(subj, "EFFECT_RULE_REQUIRES_VIA", "effect rule %s runs at %s, so it must name the capability that performs it", rule.ID, rule.Stage))
			}
		}
	}
	return out
}
