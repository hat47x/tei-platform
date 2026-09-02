package validation

import (
	"strings"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/subject"
)

type semanticValidationRelationRef struct {
	Target core.CanonicalID
}

// validateBusinessValidationReferencesは、業務Validationのrule本文と
// Applicationからのidentity参照をWorkspace全体のCanonical情報で検査する。
// 評価時の値やUI固有の表現は扱わない。
func validateBusinessValidationReferences(w ir.World) []Diagnostic {
	var out []Diagnostic

	attributes := map[core.CanonicalID]core.CanonicalID{}
	relations := map[core.CanonicalID]semanticValidationRelationRef{}
	validations := map[core.CanonicalID]semantic.Validation{}

	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = data.ID
			}
			for _, relation := range data.Relations {
				relations[relation.ID] = semanticValidationRelationRef{Target: relation.Target}
			}
		}
		for _, rule := range ns.Validations {
			ruleSubject := subject.Subject{Kind: "semantic-validation", ID: string(rule.ID)}
			if _, exists := validations[rule.ID]; exists {
				out = append(out, errFor(ruleSubject, "DUPLICATE_SEMANTIC_VALIDATION", "semantic validation %s is declared more than once", rule.ID))
				continue
			}
			validations[rule.ID] = rule

			if !rule.Kind.Valid() {
				out = append(out, errFor(ruleSubject, "INVALID_SEMANTIC_VALIDATION_KIND", "semantic validation %s has unsupported kind %q", rule.ID, rule.Kind))
				continue
			}
			owner, attributeExists := attributes[rule.Attribute]
			if rule.Attribute == "" || !attributeExists {
				out = append(out, errFor(ruleSubject, "UNKNOWN_SEMANTIC_VALIDATION_ATTRIBUTE", "semantic validation %s refers to unknown attribute %s", rule.ID, rule.Attribute))
			}

			switch rule.Kind {
			case semantic.ValidationAttributePositive:
				if rule.Relation != "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED", "attribute-positive validation %s does not accept a relation", rule.ID))
				}
			case semantic.ValidationRelationAttributeUnique:
				relation, ok := relations[rule.Relation]
				if rule.Relation == "" || !ok {
					out = append(out, errFor(ruleSubject, "UNKNOWN_SEMANTIC_VALIDATION_RELATION", "semantic validation %s refers to unknown relation %s", rule.ID, rule.Relation))
				} else if attributeExists && owner != relation.Target {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_ATTRIBUTE_RELATION_MISMATCH", "semantic validation %s evaluates attribute %s, but relation %s targets %s", rule.ID, rule.Attribute, rule.Relation, relation.Target))
				}
			}
		}
	}

	for _, app := range w.Applications {
		usedNamespaces := map[core.CanonicalID]struct{}{}
		for _, namespace := range app.SemanticUses {
			usedNamespaces[namespace] = struct{}{}
		}
		for _, validationID := range app.Validations {
			validationSubject := subject.Subject{Kind: "application", ID: string(app.ID)}
			rule, ok := validations[validationID]
			if !ok {
				out = append(out, errFor(validationSubject, "UNKNOWN_APPLICATION_VALIDATION", "application %s refers to unknown semantic validation %s", app.ID, validationID))
				continue
			}
			namespace := validationNamespace(rule.ID)
			if _, ok := usedNamespaces[namespace]; !ok {
				out = append(out, errFor(validationSubject, "APPLICATION_VALIDATION_SEMANTIC_NOT_USED", "application %s refers to semantic validation %s but does not use semantic namespace %s", app.ID, rule.ID, namespace))
			}
		}
	}

	return out
}

func validationNamespace(id core.CanonicalID) core.CanonicalID {
	text := string(id)
	if i := strings.IndexByte(text, '.'); i > 0 {
		return core.CanonicalID(text[:i])
	}
	return id
}
