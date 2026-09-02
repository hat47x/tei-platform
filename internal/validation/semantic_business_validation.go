package validation

import (
	"strings"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/subject"
)

type semanticValidationRelationRef struct {
	Target      core.CanonicalID
	Cardinality semantic.Cardinality
}

// validateSemanticBusinessValidationsは、Semanticに置かれた業務Validationが
// 既存のData / Attribute / Relationと矛盾していないかを検査する。
// 業務ruleの評価そのものはここでは行わず、authoringされた意味の参照整合性だけを扱う。
func validateSemanticBusinessValidations(w ir.World) []Diagnostic {
	var out []Diagnostic

	attributes := map[core.CanonicalID]semantic.Attribute{}
	attributeOwners := map[core.CanonicalID]core.CanonicalID{}
	relations := map[core.CanonicalID]semanticValidationRelationRef{}
	types := map[core.CanonicalID]semantic.Type{}
	otherSemanticElements := map[core.CanonicalID]string{}

	for _, ns := range w.Semantics {
		for _, typ := range ns.Types {
			types[typ.ID] = typ
			otherSemanticElements[typ.ID] = "type"
		}
		for _, data := range ns.Data {
			otherSemanticElements[data.ID] = "data"
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = attribute
				attributeOwners[attribute.ID] = data.ID
				otherSemanticElements[attribute.ID] = "attribute"
			}
			for _, relation := range data.Relations {
				relations[relation.ID] = semanticValidationRelationRef{
					Target:      relation.Target,
					Cardinality: relation.Cardinality,
				}
				otherSemanticElements[relation.ID] = "relation"
			}
		}
		for _, capability := range ns.Capabilities {
			otherSemanticElements[capability.ID] = "capability"
		}
	}

	isNumericType := func(id core.CanonicalID) bool {
		seen := map[core.CanonicalID]struct{}{}
		for id != "" {
			if id == "std.integer" || id == "std.decimal" || id == "std.money" {
				return true
			}
			if _, cycle := seen[id]; cycle {
				return false
			}
			seen[id] = struct{}{}
			typ, ok := types[id]
			if !ok || typ.Kind != semantic.TypeAlias {
				return false
			}
			id = typ.Underlying
		}
		return false
	}

	seenValidationIDs := map[core.CanonicalID]core.CanonicalID{}
	for _, ns := range w.Semantics {
		for _, rule := range ns.Validations {
			ruleSubject := subject.Subject{Kind: "semantic-validation", ID: string(rule.ID)}

			if owner, exists := seenValidationIDs[rule.ID]; exists {
				out = append(out, errFor(
					ruleSubject,
					"DUPLICATE_SEMANTIC_VALIDATION",
					"semantic validation %s is declared more than once, in namespaces %s and %s",
					rule.ID,
					owner,
					ns.ID,
				))
			} else {
				seenValidationIDs[rule.ID] = ns.ID
			}

			if prior, exists := otherSemanticElements[rule.ID]; exists {
				out = append(out, errFor(
					ruleSubject,
					"DUPLICATE_SEMANTIC_ELEMENT",
					"semantic element %s is declared as both %s and validation",
					rule.ID,
					prior,
				))
			}

			if !strings.HasPrefix(string(rule.ID), string(ns.ID)+".") {
				out = append(out, errFor(
					ruleSubject,
					"SEMANTIC_VALIDATION_NAMESPACE_MISMATCH",
					"semantic validation %s is declared in semantic namespace %s",
					rule.ID,
					ns.ID,
				))
			}
			if !rule.Kind.Valid() {
				out = append(out, errFor(
					ruleSubject,
					"INVALID_SEMANTIC_VALIDATION_KIND",
					"semantic validation %s has invalid kind %q",
					rule.ID,
					rule.Kind,
				))
				continue
			}

			attribute, attributeExists := attributes[rule.Attribute]
			if rule.Attribute == "" {
				out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_ATTRIBUTE_REQUIRED", "semantic validation %s requires an attribute", rule.ID))
			} else if !attributeExists {
				out = append(out, errFor(
					ruleSubject,
					"UNKNOWN_SEMANTIC_VALIDATION_ATTRIBUTE",
					"semantic validation %s refers to unknown semantic attribute %s",
					rule.ID,
					rule.Attribute,
				))
			}

			switch rule.Kind {
			case semantic.ValidationPositive:
				if rule.Relation != "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED", "positive semantic validation %s does not accept a relation", rule.ID))
				}
				if attributeExists && !isNumericType(attribute.TypeRef) {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE",
						"positive semantic validation %s requires a numeric attribute but %s has type %s",
						rule.ID,
						rule.Attribute,
						attribute.TypeRef,
					))
				}

			case semantic.ValidationRelationUnique:
				if rule.Relation == "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_RELATION_REQUIRED", "relation-unique semantic validation %s requires a relation", rule.ID))
					continue
				}
				relation, relationExists := relations[rule.Relation]
				if !relationExists {
					out = append(out, errFor(
						ruleSubject,
						"UNKNOWN_SEMANTIC_VALIDATION_RELATION",
						"semantic validation %s refers to unknown semantic relation %s",
						rule.ID,
						rule.Relation,
					))
					continue
				}
				if relation.Cardinality != semantic.CardinalityMany {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_RELATION_NOT_MANY",
						"relation-unique semantic validation %s requires a many relation but %s has cardinality %s",
						rule.ID,
						rule.Relation,
						relation.Cardinality,
					))
				}
				if attributeExists && attributeOwners[rule.Attribute] != relation.Target {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_ATTRIBUTE_OUTSIDE_RELATION_TARGET",
						"relation-unique semantic validation %s uses attribute %s outside relation target %s",
						rule.ID,
						rule.Attribute,
						relation.Target,
					))
				}
			}
		}
	}

	return out
}
