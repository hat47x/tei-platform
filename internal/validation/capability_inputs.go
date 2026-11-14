package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateCapabilityInputs keeps a parsed Workspace from accepting duplicate
// input names that capabilityinvoke already refuses before Adapter dispatch.
// Value type resolution remains owned by the existing global semantic checks.
// The diagnostic identifies the Capability; byte spans are Source-owned.
func validateCapabilityInputs(w ir.World) []Diagnostic {
	var out []Diagnostic
	for _, namespace := range w.Semantics {
		for _, capability := range namespace.Capabilities {
			subj := subject.Subject{Kind: "semantic-capability", ID: string(capability.ID)}
			seen := make(map[core.LocalName]struct{}, len(capability.Inputs))
			for _, input := range capability.Inputs {
				if input.Name == "" {
					out = append(out, errFor(subj, "SEMANTIC_CAPABILITY_INPUT_NAME_REQUIRED",
						"capability %s declares an input without a name", capability.ID))
					continue
				}
				if _, duplicate := seen[input.Name]; duplicate {
					out = append(out, errFor(subj, "DUPLICATE_SEMANTIC_CAPABILITY_INPUT",
						"capability %s declares input %s more than once", capability.ID, input.Name))
					continue
				}
				seen[input.Name] = struct{}{}
			}
		}
	}
	return out
}
