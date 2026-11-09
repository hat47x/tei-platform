package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
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
	inbound := map[core.CanonicalID][]semantic.Relation{}
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				inbound[relation.Target] = append(inbound[relation.Target], relation)
			}
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
			if effect.Kind == semantic.DataEffectDelete {
				for _, relation := range inbound[effect.Data] {
					if relation.OnDelete == "" {
						out = append(out, warnd("DELETE_EFFECT_RELATION_POLICY_MISSING", "capability %s deletes %s but inbound relation %s has no on-delete policy", capability.ID, effect.Data, relation.ID))
					}
				}
			}
		}
	}
	return out
}
