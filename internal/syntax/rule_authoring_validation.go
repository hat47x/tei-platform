package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file converts the rule authoring surface into the *existing*
// semantic.Validation value. It adds no Validation kind, no Role, and no
// second meaning source: the predicates map onto the kinds that
// internal/semantic already declares and internal/semantic/validationeval
// already evaluates.
//
// The output is a candidate. It is not accepted meaning, and it carries no
// execution authority. The caller hands it to the existing semantic validation
// and, when the owner decides, to the existing acceptance boundary.

// RuleConversionIssue is a surface problem found while converting one rule.
// It keeps the exact authored span so a diagnostic points at the bytes the
// author wrote instead of at a re-lexed copy.
type RuleConversionIssue struct {
	RuleID core.CanonicalID
	Span   Anchor
	Msg    string
}

func (i RuleConversionIssue) Error() string {
	return fmt.Sprintf("%s: %s", i.RuleID, i.Msg)
}

// RuleValidationKind maps one predicate word onto the existing validation kind.
// The second return value is false for a word this surface does not define, so
// a caller never receives an approximate kind.
func RuleValidationKind(predicate RulePredicate) (semantic.ValidationKind, bool) {
	switch predicate {
	case RulePredicatePositive:
		return semantic.ValidationPositive, true
	case RulePredicateUnique:
		return semantic.ValidationRelationUnique, true
	default:
		return "", false
	}
}

// RuleNamespace converts one parsed rule authoring source into a semantic
// Namespace holding the candidate validations. It is shaped exactly like the
// namespace the existing semantic parser produces, so an existing loader can
// append it without a new path.
//
// Every issue is reported; the conversion does not stop at the first bad rule,
// because an author with three mistakes should see three. A rule with an issue
// contributes no validation, so a half-resolved rule never reaches the
// validator as a plausible meaning.
func RuleNamespace(file RuleAuthoringFile) (semantic.Namespace, []RuleConversionIssue) {
	out := semantic.Namespace{ID: file.Namespace}
	var issues []RuleConversionIssue
	for _, rule := range file.Rules {
		validation, issue := ruleToValidation(rule)
		if issue != nil {
			issues = append(issues, *issue)
			continue
		}
		out.Validations = append(out.Validations, validation)
	}
	return out, issues
}

func ruleToValidation(rule RuleAuthoring) (semantic.Validation, *RuleConversionIssue) {
	kind, ok := RuleValidationKind(rule.Predicate)
	if !ok {
		return semantic.Validation{}, &RuleConversionIssue{
			RuleID: rule.ID, Span: rule.PredicateSpan,
			Msg: fmt.Sprintf("predicate %q has no existing validation kind", rule.Predicate),
		}
	}
	attribute, issue := ruleCanonicalReference(rule, rule.Subject, "subject")
	if issue != nil {
		return semantic.Validation{}, issue
	}
	validation := semantic.Validation{
		ID:        rule.ID,
		Symbol:    rule.Symbol,
		Kind:      kind,
		Attribute: attribute,
	}
	if kind == semantic.ValidationRelationUnique {
		if rule.Scope == nil {
			return semantic.Validation{}, &RuleConversionIssue{
				RuleID: rule.ID, Span: rule.PredicateSpan,
				Msg: "relation-unique requires a scope; none was authored",
			}
		}
		relation, issue := ruleCanonicalReference(rule, *rule.Scope, "scope")
		if issue != nil {
			return semantic.Validation{}, issue
		}
		validation.Relation = relation
	} else if rule.Scope != nil {
		return semantic.Validation{}, &RuleConversionIssue{
			RuleID: rule.ID, Span: rule.ScopeSpan,
			Msg: fmt.Sprintf("%s accepts no scope, but %q was authored", kind, rule.Scope.Text),
		}
	}
	return validation, nil
}

// ruleCanonicalReference resolves a surface reference to a canonical ID.
//
// Only a fully qualified canonical ID is accepted. A bare symbol would have to
// name an owning Data, and no declaration in this source provides one: picking
// a Data by name proximity or by "the only Data in the file" is exactly the
// ambiguous reference this surface must refuse (#4359 negative case N-05).
func ruleCanonicalReference(rule RuleAuthoring, ref RuleReference, role string) (core.CanonicalID, *RuleConversionIssue) {
	text := strings.TrimSpace(ref.Text)
	if !strings.Contains(text, ".") {
		return "", &RuleConversionIssue{
			RuleID: rule.ID, Span: ref.Anchor,
			Msg: fmt.Sprintf("%s %q is not a qualified canonical id; write the owning Data, for example %s",
				role, text, rule.Symbol+"."+text),
		}
	}
	id, err := core.ParseCanonicalID(text)
	if err != nil {
		return "", &RuleConversionIssue{
			RuleID: rule.ID, Span: ref.Anchor,
			Msg: fmt.Sprintf("%s %q: %v", role, text, err),
		}
	}
	return id, nil
}
