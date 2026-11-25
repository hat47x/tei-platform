package validation

import (
	"sort"
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

// isCanonicalReference reports whether an input assignment names an
// attribute. A decimal literal or a bare word is a constant or a mapping
// name, not a column, so it is not checked against the relation's rows.
func isCanonicalReference(value string) bool {
	if value == "" || !strings.Contains(value, ".") {
		return false
	}
	_, err := core.ParseCanonicalID(value)
	return err == nil
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
				// An example states either one value or one value per declared
				// input. Each stated value is checked the same way: it must be a
				// decimal number, and no two examples may state the same values.
				stated := []string{example.Value}
				if example.Inputs != nil {
					stated = stated[:0]
					for name, value := range example.Inputs {
						if _, err := semantic.ParseOperatorCaseValue(value); err != nil {
							out = append(out, errFor(
								operatorSubject,
								"INVALID_SEMANTIC_OPERATOR_CASE",
								"semantic operator %s has an example whose value for input %q, %q, is not a decimal number",
								operator.ID, name, value,
							))
							stated = nil
							break
						}
						stated = append(stated, value)
					}
					if stated == nil {
						continue
					}
					sort.Strings(stated)
				} else if _, err := semantic.ParseOperatorCaseValue(example.Value); err != nil {
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
					if semantic.SameOperatorCaseStates(prior, example) {
						out = append(out, errFor(
							operatorSubject,
							"DUPLICATE_SEMANTIC_OPERATOR_CASE",
							"semantic operator %s states the same example more than once (%s)",
							operator.ID,
							strings.Join(stated, ", "),
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

			if rule.Kind != semantic.ValidationNoOverlap && (rule.GroupBy != "" || rule.OpenEnded) {
				out = append(out, errFor(
					ruleSubject,
					"SEMANTIC_VALIDATION_PERIOD_GROUPING_NOT_ALLOWED",
					"%s semantic validation %s is not about periods and accepts no grouping and no open end",
					rule.Kind,
					rule.ID,
				))
			}
			if rule.Kind == semantic.ValidationNoOverlap && rule.IfPresent {
				out = append(out, errFor(
					ruleSubject,
					"SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_NOT_ALLOWED",
					"no-overlap semantic validation %s states an open end with openEnded, not with ifPresent",
					rule.ID,
				))
			}
			if rule.Kind != semantic.ValidationNotAfter && rule.Kind != semantic.ValidationNoOverlap && rule.Kind != semantic.ValidationRequiredIf && rule.Compared != "" {
				out = append(out, errFor(
					ruleSubject,
					"SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_NOT_ALLOWED",
					"%s semantic validation %s does not compare its attribute with another one",
					rule.Kind,
					rule.ID,
				))
			}
			// no-overlap has its own message for ifPresent, above.
			if rule.Kind != semantic.ValidationNotAfter && rule.Kind != semantic.ValidationNoOverlap && rule.IfPresent {
				out = append(out, errFor(
					ruleSubject,
					"SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_NOT_ALLOWED",
					"%s semantic validation %s does not accept ifPresent",
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
				operator, declared := operators[rule.Operator]
				// An operator whose declaration reads a column of the rows is
				// asked of a relation, so that rule names one; an operator that
				// reads one record's values is not, and a relation written for it
				// would ask a question the declaration cannot answer (decision
				// section 15.4).
				if declared && operatorReadsEachRow(operator) {
					if rule.Relation == "" {
						out = append(out, errFor(
							ruleSubject,
							"SEMANTIC_VALIDATION_OPERATOR_RELATION_REQUIRED",
							"operator semantic validation %s uses operator %s, which reads every row of a relation, but names no relation; write \"per <relation>\"",
							rule.ID, rule.Operator,
						))
					} else if relation, known := relations[rule.Relation]; !known {
						// A relation that no Data declares cannot be read, and
						// the read is the only place this used to be found. A
						// reader who mistyped it would see "semantic validation
						// cannot read ... over relation <typo>" only when they
						// happened to open a screen; validate said ok.
						out = append(out, errFor(
							ruleSubject,
							"SEMANTIC_VALIDATION_OPERATOR_RELATION_UNKNOWN",
							"operator semantic validation %s names relation %s, which no Data declares",
							rule.ID, rule.Relation,
						))
					} else if attributeExists && attributeOwners[rule.Attribute] != relation.Target {
						// The rule is read over the rows of one relation, and the
						// subject attribute has to be a column of those rows. A
						// relation to another Data gives rows of that Data, so
						// the read refuses with "these are the rows of relation
						// ..., not of ..." — but only when someone opens a
						// screen. Validate refuses it for the same reason
						// relation-unique already does.
						out = append(out, errFor(
							ruleSubject,
							"SEMANTIC_VALIDATION_OPERATOR_ATTRIBUTE_OUTSIDE_RELATION_TARGET",
							"operator semantic validation %s reads attribute %s over relation %s, whose rows belong to %s",
							rule.ID, rule.Attribute, rule.Relation, relation.Target,
						))
					}
				} else if rule.Relation != "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED", "operator semantic validation %s does not accept a relation", rule.ID))
				}
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
				// A rule that reads the rows of a relation reads the columns of
				// those rows. Every assigned input must be a column of the
				// relation's target Data; a column of another Data is not there
				// to read, and the read refuses it with "is not an item of ...".
				// Validate refuses it first so the same Workspace does not pass
				// here and fail at the first screen.
				if declared && operatorReadsEachRow(operator) && rule.Relation != "" {
					if relation, known := relations[rule.Relation]; known {
						for _, assignment := range rule.Inputs {
							if !isCanonicalReference(assignment.Value) {
								continue
							}
							column, err := core.ParseCanonicalID(assignment.Value)
							if err != nil {
								continue
							}
							// Two shapes reach here: a column no Data declares,
							// and a column of another Data. Either way it is not
							// a column of the rows this rule reads, and the read
							// refuses it with "is not an item of ...".
							if attributeOwners[column] == relation.Target {
								continue
							}
							out = append(out, errFor(
								ruleSubject,
								"SEMANTIC_VALIDATION_OPERATOR_INPUT_OUTSIDE_RELATION_TARGET",
								"operator semantic validation %s reads input %q from %s, which is not a column of relation %s's rows (%s)",
								rule.ID, assignment.Name, column, rule.Relation, relation.Target,
							))
						}
					}
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

			case semantic.ValidationNoOverlap:
				if rule.Relation == "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_RELATION_REQUIRED", "no-overlap semantic validation %s requires a relation", rule.ID))
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
						"no-overlap semantic validation %s requires a many relation but %s has cardinality %s",
						rule.ID,
						rule.Relation,
						relation.Cardinality,
					))
				}
				if rule.Compared == "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_REQUIRED", "no-overlap semantic validation %s requires the attribute that ends the period", rule.ID))
					continue
				}
				end, endExists := attributes[rule.Compared]
				if !endExists {
					out = append(out, errFor(
						ruleSubject,
						"UNKNOWN_SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE",
						"semantic validation %s ends its period with unknown semantic attribute %s",
						rule.ID,
						rule.Compared,
					))
					continue
				}
				if rule.GroupBy != "" {
					if _, groupExists := attributes[rule.GroupBy]; !groupExists {
						out = append(out, errFor(
							ruleSubject,
							"UNKNOWN_SEMANTIC_VALIDATION_GROUP_ATTRIBUTE",
							"semantic validation %s groups rows by unknown semantic attribute %s",
							rule.ID,
							rule.GroupBy,
						))
						continue
					}
				}
				if !attributeExists {
					continue
				}
				if rule.Compared == rule.Attribute {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_NO_OVERLAP_SAME_ATTRIBUTE",
						"no-overlap semantic validation %s starts and ends its period with the same attribute %s",
						rule.ID,
						rule.Attribute,
					))
					continue
				}
				for _, read := range []core.CanonicalID{rule.Attribute, rule.Compared, rule.GroupBy} {
					if read != "" && attributeOwners[read] != relation.Target {
						out = append(out, errFor(
							ruleSubject,
							"SEMANTIC_VALIDATION_ATTRIBUTE_OUTSIDE_RELATION_TARGET",
							"no-overlap semantic validation %s uses attribute %s outside relation target %s",
							rule.ID,
							read,
							relation.Target,
						))
					}
				}
				if !isDateType(attribute.TypeRef) || !isDateType(end.TypeRef) {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_NO_OVERLAP_REQUIRES_DATE_ATTRIBUTES",
						"no-overlap semantic validation %s requires a period of two calendar dates but %s has type %s and %s has type %s",
						rule.ID,
						rule.Attribute,
						attribute.TypeRef,
						rule.Compared,
						end.TypeRef,
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

			case semantic.ValidationRequiredIf:
				if rule.Relation != "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED", "required-if semantic validation %s does not accept a relation", rule.ID))
				}
				if rule.Compared == "" {
					out = append(out, errFor(ruleSubject, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_REQUIRED", "required-if semantic validation %s requires an attribute whose value makes %s required", rule.ID, rule.Attribute))
					continue
				}
				if _, triggerExists := attributes[rule.Compared]; !triggerExists {
					out = append(out, errFor(
						ruleSubject,
						"UNKNOWN_SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE",
						"semantic validation %s makes %s required when unknown semantic attribute %s has a value",
						rule.ID,
						rule.Attribute,
						rule.Compared,
					))
					continue
				}
				if !attributeExists {
					continue
				}
				if rule.Compared == rule.Attribute {
					// "A is required when A has a value" is always satisfied and
					// says nothing; writing it hides that the author meant
					// another attribute.
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_REQUIRED_IF_SAME_ATTRIBUTE",
						"required-if semantic validation %s makes attribute %s required when it has a value",
						rule.ID,
						rule.Attribute,
					))
					continue
				}
				if attributeOwners[rule.Attribute] != attributeOwners[rule.Compared] {
					out = append(out, errFor(
						ruleSubject,
						"SEMANTIC_VALIDATION_REQUIRED_IF_REQUIRES_SAME_DATA",
						"required-if semantic validation %s makes %s required by %s, which belongs to another data",
						rule.ID,
						rule.Attribute,
						rule.Compared,
					))
				}
				// A rule about presence reads only whether a value is there,
				// so any attribute type is expressible. That is why no type
				// check exists here: refusing a type would state a constraint
				// the rule does not have.
			}
		}
	}

	return out
}

// operatorReadsEachRow reports whether the operator declares an input read over
// every row of a relation.
func operatorReadsEachRow(operator semantic.Operator) bool {
	if len(operator.Inputs) == 0 {
		return false
	}
	for _, input := range operator.Inputs {
		if input.OfEachRow {
			return true
		}
	}
	return false
}
