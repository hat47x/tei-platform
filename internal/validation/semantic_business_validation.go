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

	// A calendar date is std.date or an alias of it. A moment in time
	// (std.datetime) is not: ordering moments needs a time zone.
	isDateType := func(id core.CanonicalID) bool {
		seen := map[core.CanonicalID]struct{}{}
		for id != "" {
			if id == "std.date" {
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

	// Operator contracts are data. A rule may refer only to a declared one, and
	// a declaration decides what the rule reads.
	operators := map[core.CanonicalID]semantic.Operator{}
	for _, ns := range w.Semantics {
		for _, operator := range ns.Operators {
			operatorSubject := subject.Subject{Kind: "semantic-operator", ID: string(operator.ID)}
			if _, exists := operators[operator.ID]; exists {
				out = append(out, errFor(
					operatorSubject,
					"DUPLICATE_SEMANTIC_OPERATOR",
					"semantic operator %s is declared more than once",
					operator.ID,
				))
				continue
			}
			operators[operator.ID] = operator
			if prior, exists := otherSemanticElements[operator.ID]; exists {
				// A rule writes an attribute before `is` and an operator after
				// it. One identity that is both would make a reordered rule
				// resolve instead of failing.
				out = append(out, errFor(
					operatorSubject,
					"DUPLICATE_SEMANTIC_ELEMENT",
					"semantic element %s is declared as both %s and operator",
					operator.ID,
					prior,
				))
			}
			if !strings.HasPrefix(string(operator.ID), string(ns.ID)+".") {
				out = append(out, errFor(
					operatorSubject,
					"SEMANTIC_OPERATOR_NAMESPACE_MISMATCH",
					"semantic operator %s is declared in semantic namespace %s",
					operator.ID,
					ns.ID,
				))
			}
			for i, example := range operator.Cases {
				if _, err := semantic.ParseOperatorCaseValue(example.Value); err != nil {
					out = append(out, errFor(
						operatorSubject,
						"INVALID_SEMANTIC_OPERATOR_CASE",
						"semantic operator %s has an example whose value %q is not a decimal number",
						operator.ID,
						example.Value,
					))
					continue
				}
				for _, prior := range operator.Cases[:i] {
					if semantic.SameOperatorCaseValue(prior.Value, example.Value) {
						out = append(out, errFor(
							operatorSubject,
							"DUPLICATE_SEMANTIC_OPERATOR_CASE",
							"semantic operator %s states the example value %s more than once",
							operator.ID,
							example.Value,
						))
						break
					}
				}
			}
			if !operator.Input.Valid() {
				out = append(out, errFor(
					operatorSubject,
					"INVALID_SEMANTIC_OPERATOR_INPUT",
					"semantic operator %s has invalid input %q",
					operator.ID,
					operator.Input,
				))
			}
		}
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

			if rule.Kind != semantic.ValidationNotAfter && (rule.Compared != "" || rule.IfPresent) {
				out = append(out, errFor(
					ruleSubject,
					"SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_NOT_ALLOWED",
					"%s semantic validation %s does not compare its attribute with another one",
					rule.Kind,
					rule.ID,
				))
			}
			if rule.Kind != semantic.ValidationOperator && rule.Operator != "" {
				out = append(out, errFor(
					ruleSubject,
					"SEMANTIC_VALIDATION_OPERATOR_NOT_ALLOWED",
					"%s semantic validation %s does not accept an operator",
					rule.Kind,
					rule.ID,
				))
			}

			switch rule.Kind {
			case semantic.ValidationOperator:
				if rule.Relation != "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED", "operator semantic validation %s does not accept a relation", rule.ID))
				}
				operator, declared := operators[rule.Operator]
				if !declared {
					out = append(out, errFor(
						ruleSubject,
						"UNKNOWN_SEMANTIC_VALIDATION_OPERATOR",
						"semantic validation %s refers to undeclared semantic operator %q",
						rule.ID,
						rule.Operator,
					))
					continue
				}
				if operator.Input == semantic.OperatorInputNumber && attributeExists && !isNumericType(attribute.TypeRef) {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_OPERATOR_REQUIRES_NUMERIC_ATTRIBUTE",
						"semantic validation %s uses operator %s, which takes a number, but %s has type %s",
						rule.ID,
						rule.Operator,
						rule.Attribute,
						attribute.TypeRef,
					))
				}

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

			case semantic.ValidationNotAfter:
				if rule.Relation != "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED", "not-after semantic validation %s does not accept a relation", rule.ID))
				}
				if rule.Compared == "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_REQUIRED", "not-after semantic validation %s requires an attribute to compare with", rule.ID))
					continue
				}
				compared, comparedExists := attributes[rule.Compared]
				if !comparedExists {
					out = append(out, errFor(
						ruleSubject,
						"UNKNOWN_SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE",
						"semantic validation %s compares with unknown semantic attribute %s",
						rule.ID,
						rule.Compared,
					))
					continue
				}
				if !attributeExists {
					continue
				}
				if rule.Compared == rule.Attribute {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_NOT_AFTER_SAME_ATTRIBUTE",
						"not-after semantic validation %s compares attribute %s with itself",
						rule.ID,
						rule.Attribute,
					))
					continue
				}
				if attributeOwners[rule.Attribute] != attributeOwners[rule.Compared] {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_NOT_AFTER_REQUIRES_SAME_DATA",
						"not-after semantic validation %s compares %s with %s, which belongs to another data",
						rule.ID,
						rule.Attribute,
						rule.Compared,
					))
				}
				if !isDateType(attribute.TypeRef) || !isDateType(compared.TypeRef) {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_NOT_AFTER_REQUIRES_DATE_ATTRIBUTES",
						"not-after semantic validation %s requires two calendar dates but %s has type %s and %s has type %s",
						rule.ID,
						rule.Attribute,
						attribute.TypeRef,
						rule.Compared,
						compared.TypeRef,
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
