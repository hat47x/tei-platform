package validation

import (
	"strings"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateSemanticBusinessRules validates explicitly authored logical
// assertions, not whether the business claim is true for arbitrary records.
// No candidate key is inferred from physical PKs, relation keys, or data.
func validateSemanticBusinessRules(w ir.World) []Diagnostic {
	var out []Diagnostic
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			attributes := make(map[core.CanonicalID]bool, len(data.Attributes))
			identities := make(map[core.CanonicalID]bool, len(data.Attributes)+len(data.Relations))
			for _, attr := range data.Attributes {
				attributes[attr.ID] = true
				identities[attr.ID] = true
			}
			for _, relation := range data.Relations {
				identities[relation.ID] = true
			}
			check := func(kind string, ruleID core.CanonicalID, fields []core.CanonicalID, role string) {
				sub := subject.Subject{Kind: kind, ID: string(ruleID)}
				if len(fields) == 0 {
					out = append(out, errFor(sub, "EMPTY_SEMANTIC_RULE_FIELDS", "rule %s has no %s", ruleID, role))
				}
				seen := make(map[core.CanonicalID]bool, len(fields))
				for _, field := range fields {
					if !strings.HasPrefix(string(field), string(data.ID)+".") || !attributes[field] {
						out = append(out, errFor(sub, "UNKNOWN_SEMANTIC_RULE_ATTRIBUTE", "rule %s %s %s is not an attribute declared by Data %s", ruleID, role, field, data.ID))
					}
					if seen[field] {
						out = append(out, errFor(sub, "DUPLICATE_SEMANTIC_RULE_ATTRIBUTE", "rule %s repeats %s %s", ruleID, role, field))
					}
					seen[field] = true
				}
			}
			seenRules := make(map[core.CanonicalID]bool, len(data.CandidateKeys)+len(data.FunctionalDependencies))
			identity := func(kind string, id core.CanonicalID) {
				sub := subject.Subject{Kind: kind, ID: string(id)}
				if !strings.HasPrefix(string(id), string(data.ID)+".") {
					out = append(out, errFor(sub, "SEMANTIC_RULE_PARENT_MISMATCH", "rule %s does not belong to Data %s", id, data.ID))
				}
				if seenRules[id] || identities[id] {
					out = append(out, errFor(sub, "DUPLICATE_SEMANTIC_RULE_IDENTITY", "semantic Data %s has a repeated member identity %s", data.ID, id))
				}
				seenRules[id] = true
			}
			for _, key := range data.CandidateKeys {
				identity("semantic-candidate-key", key.ID)
				check("semantic-candidate-key", key.ID, key.Attributes, "field")
			}
			for _, fd := range data.FunctionalDependencies {
				identity("semantic-functional-dependency", fd.ID)
				check("semantic-functional-dependency", fd.ID, fd.Determinants, "determinant")
				check("semantic-functional-dependency", fd.ID, fd.Dependents, "dependent")
			}
		}
	}
	return out
}
