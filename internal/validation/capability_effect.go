package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateCapabilityDataEffects validates explicit business intent, not a
// Provider's authority or proof of a durable write.
func validateCapabilityDataEffects(w ir.World) []Diagnostic {
	knownData := map[core.CanonicalID]struct{}{}
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			knownData[data.ID] = struct{}{}
		}
	}
	var out []Diagnostic
	for _, namespace := range w.Semantics {
		for _, capability := range namespace.Capabilities {
			if capability.Effect == nil {
				continue
			}
			effect := capability.Effect
			subj := subject.Subject{Kind: "semantic-capability", ID: string(capability.ID)}
			if !effect.Kind.Valid() {
				out = append(out, errFor(subj, "INVALID_SEMANTIC_CAPABILITY_DATA_EFFECT", "capability %s has unsupported data effect %q", capability.ID, effect.Kind))
			}
			if _, ok := knownData[effect.Data]; !ok || effect.Data == "" {
				out = append(out, errFor(subj, "UNKNOWN_SEMANTIC_CAPABILITY_EFFECT_DATA", "capability %s has data effect for unknown semantic data %s", capability.ID, effect.Data))
			}
		}
	}
	return out
}
