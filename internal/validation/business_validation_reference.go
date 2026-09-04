package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateBusinessValidationReferences checks Application references against declared namespaces.
func validateBusinessValidationReferences(w ir.World) []Diagnostic {
	var out []Diagnostic
	validations := map[core.CanonicalID]core.CanonicalID{}
	for _, ns := range w.Semantics {
		for _, rule := range ns.Validations {
			validations[rule.ID] = ns.ID
		}
	}
	for _, app := range w.Applications {
		usedNamespaces := map[core.CanonicalID]struct{}{}
		for _, namespace := range app.SemanticUses {
			usedNamespaces[namespace] = struct{}{}
		}
		for _, validationID := range app.Validations {
			validationSubject := subject.Subject{Kind: "application", ID: string(app.ID)}
			namespace, ok := validations[validationID]
			if !ok {
				out = append(out, errFor(validationSubject, "UNKNOWN_APPLICATION_VALIDATION", "application %s refers to unknown semantic validation %s", app.ID, validationID))
				continue
			}
			if _, ok := usedNamespaces[namespace]; !ok {
				out = append(out, errFor(validationSubject, "APPLICATION_VALIDATION_SEMANTIC_NOT_USED", "application %s refers to semantic validation %s but does not use semantic namespace %s", app.ID, validationID, namespace))
			}
		}
	}

	return out
}
