package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateCapabilityOutcomesは、Semantic Capabilityの通常outputと名前付きOutcomeの境界を検査する。
// Parserを経由しないCanonical Worldでも同じ制約を保てるよう、Workspace全体のValidationとして提供する。
func validateCapabilityOutcomes(w ir.World) []Diagnostic {
	var out []Diagnostic

	valueTypes := map[core.CanonicalID]struct{}{}
	for _, namespace := range w.Semantics {
		for _, typ := range namespace.Types {
			valueTypes[typ.ID] = struct{}{}
		}
		for _, data := range namespace.Data {
			valueTypes[data.ID] = struct{}{}
		}
	}

	for _, namespace := range w.Semantics {
		for _, capability := range namespace.Capabilities {
			capabilitySubject := subject.Subject{Kind: "semantic-capability", ID: string(capability.ID)}

			if capability.Output != "" && len(capability.Outcomes) > 0 {
				out = append(out, errFor(
					capabilitySubject,
					"SEMANTIC_CAPABILITY_OUTPUT_OUTCOME_CONFLICT",
					"capability %s cannot declare both output and named outcomes",
					capability.ID,
				))
			}

			seenNames := map[core.LocalName]struct{}{}
			for _, outcome := range capability.Outcomes {
				if _, exists := seenNames[outcome.Name]; exists {
					out = append(out, errFor(
						capabilitySubject,
						"DUPLICATE_SEMANTIC_CAPABILITY_OUTCOME",
						"capability %s declares outcome %s more than once",
						capability.ID,
						outcome.Name,
					))
				} else {
					seenNames[outcome.Name] = struct{}{}
				}

				if outcome.Payload == "" {
					out = append(out, errFor(
						capabilitySubject,
						"SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD_REQUIRED",
						"capability %s outcome %s requires a payload type",
						capability.ID,
						outcome.Name,
					))
					continue
				}
				if _, exists := valueTypes[outcome.Payload]; !exists {
					out = append(out, errFor(
						capabilitySubject,
						"UNKNOWN_SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD",
						"capability %s outcome %s refers to unknown semantic value type %s",
						capability.ID,
						outcome.Name,
						outcome.Payload,
					))
				}
			}
		}
	}

	return out
}
