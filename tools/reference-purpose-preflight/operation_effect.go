package main

// This derived observation classifies explicitly declared Canonical Capability
// write intent. The absence of an effect is not proof of a read-only execution.
// It never predicts, authorizes, executes, or fences a provider-side mutation.

import (
	"tei/internal/application"
	"tei/internal/ir"
)

type OperationWriteIntent struct {
	Status     string `json:"status"`
	Capability string `json:"capability,omitempty"`
	EffectKind string `json:"effectKind,omitempty"`
	EffectData string `json:"effectData,omitempty"`
	Reason     string `json:"reason"`
}

func ResolveOperationWriteIntent(world ir.World, applicationID, patternID string) OperationWriteIntent {
	result := OperationWriteIntent{Status: "unresolved", Reason: "application-or-pattern-not-unique"}
	var pattern application.Pattern
	matches := 0
	for _, app := range world.Applications {
		if string(app.ID) != applicationID {
			continue
		}
		for _, candidate := range app.Patterns {
			if string(candidate.ID) == patternID {
				pattern = candidate
				matches++
			}
		}
	}
	if matches != 1 {
		return result
	}
	if pattern.Kind != application.PatternCommand {
		result.Status, result.Reason = "not-command", "no-command-write-intent-evaluated"
		return result
	}
	result.Capability = string(pattern.Capability)
	if result.Capability == "" {
		result.Reason = "command-capability-reference-missing"
		return result
	}
	var found int
	var effectKind, effectData string
	var hasEffect bool
	for _, ns := range world.Semantics {
		for _, capability := range ns.Capabilities {
			if string(capability.ID) != result.Capability {
				continue
			}
			found++
			if capability.Effect != nil {
				hasEffect = true
				effectKind = string(capability.Effect.Kind)
				effectData = string(capability.Effect.Data)
			}
		}
	}
	if found != 1 {
		result.Reason = "command-capability-not-unique"
		return result
	}
	if !hasEffect {
		result.Status, result.Reason = "effect-not-declared", "no-explicit-data-effect-is-not-read-only-proof"
		return result
	}
	if (effectKind != "create" && effectKind != "update") || effectData == "" {
		result.Reason = "command-effect-incomplete-or-unsupported"
		return result
	}
	result.Status = "declared-write"
	result.Reason = "canonical-capability-data-effect-declaration"
	result.EffectKind = effectKind
	result.EffectData = effectData
	return result
}
