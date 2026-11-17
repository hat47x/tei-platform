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
	case RulePredicateOperator:
		return semantic.ValidationOperator, true
	case RulePredicateNotAfter:
		return semantic.ValidationNotAfter, true
	case RulePredicateNotOverlapping:
		return semantic.ValidationNoOverlap, true
	case RulePredicateRequiredIf:
		return semantic.ValidationRequiredIf, true
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
	for _, operator := range file.Operators {
		input, ok := OperatorInputFor(operator.Input)
		if !ok {
			issues = append(issues, RuleConversionIssue{
				RuleID: operator.ID, Span: operator.InputSpan,
				Msg: fmt.Sprintf("operator input %q has no existing operator input", operator.Input),
			})
			continue
		}
		// InputName is what the author wrote, and nothing more. Leaving it empty
		// for the unnamed form keeps an undeclared name out of the contract; a
		// reader that needs a name for the single input applies "value", which
		// is the same reading the surface already had.
		out.Operators = append(out.Operators, semantic.Operator{
			ID: operator.ID, Symbol: operator.Symbol, Input: input,
			InputName: operator.InputName,
		})
	}
	return out, issues
}

// OperatorInputFor maps one input word onto the existing operator input. The
// second return value is false for a word this surface does not define.
func OperatorInputFor(word OperatorInputWord) (semantic.OperatorInput, bool) {
	if word == OperatorInputNumberWord {
		return semantic.OperatorInputNumber, true
	}
	return "", false
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
	if kind == semantic.ValidationOperator {
		if rule.Operator == nil {
			return semantic.Validation{}, &RuleConversionIssue{
				RuleID: rule.ID, Span: rule.PredicateSpan,
				Msg: "an operator rule requires an operator reference; none was authored",
			}
		}
		operator, issue := ruleCanonicalReference(rule, *rule.Operator, "operator")
		if issue != nil {
			return semantic.Validation{}, issue
		}
		validation.Operator = operator
	} else if rule.Operator != nil {
		return semantic.Validation{}, &RuleConversionIssue{
			RuleID: rule.ID, Span: rule.Operator.Anchor,
			Msg: fmt.Sprintf("%s accepts no operator reference, but %q was authored", kind, rule.Operator.Text),
		}
	}
	if kind == semantic.ValidationNotAfter {
		if rule.Compared == nil {
			return semantic.Validation{}, &RuleConversionIssue{
				RuleID: rule.ID, Span: rule.PredicateSpan,
				Msg: "not-after requires an attribute to compare with; none was authored",
			}
		}
		compared, issue := ruleCanonicalReference(rule, *rule.Compared, "compared attribute")
		if issue != nil {
			return semantic.Validation{}, issue
		}
		validation.Compared = compared
		validation.IfPresent = rule.IfPresent
	} else if kind == semantic.ValidationRequiredIf {
		if rule.Compared == nil {
			return semantic.Validation{}, &RuleConversionIssue{
				RuleID: rule.ID, Span: rule.PredicateSpan,
				Msg: "required-if requires the attribute whose value makes the subject required; none was authored",
			}
		}
		// The parser refuses a trailing `if present` after the trigger, so
		// there is nothing to reject here: a rule about presence has no such
		// clause to carry.
		compared, issue := ruleCanonicalReference(rule, *rule.Compared, "attribute whose value makes the subject required")
		if issue != nil {
			return semantic.Validation{}, issue
		}
		validation.Compared = compared
	} else if rule.Compared != nil || rule.IfPresent {
		return semantic.Validation{}, &RuleConversionIssue{
			RuleID: rule.ID, Span: rule.PredicateSpan,
			Msg: fmt.Sprintf("%s accepts no attribute to compare with", kind),
		}
	}
	if kind == semantic.ValidationNoOverlap {
		if rule.PeriodEnd == nil {
			return semantic.Validation{}, &RuleConversionIssue{
				RuleID: rule.ID, Span: rule.SubjectSpan,
				Msg: "no-overlap requires a period; no end of the period was authored",
			}
		}
		end, issue := ruleCanonicalReference(rule, *rule.PeriodEnd, "end of the period")
		if issue != nil {
			return semantic.Validation{}, issue
		}
		validation.Compared = end
		validation.OpenEnded = rule.OpenEnded
		if rule.GroupBy != nil {
			group, issue := ruleCanonicalReference(rule, *rule.GroupBy, "grouping attribute")
			if issue != nil {
				return semantic.Validation{}, issue
			}
			validation.GroupBy = group
		}
	} else if rule.PeriodEnd != nil || rule.OpenEnded || rule.GroupBy != nil {
		return semantic.Validation{}, &RuleConversionIssue{
			RuleID: rule.ID, Span: rule.PredicateSpan,
			Msg: fmt.Sprintf("%s accepts no period and no grouping", kind),
		}
	}
	if kind == semantic.ValidationRelationUnique || kind == semantic.ValidationNoOverlap {
		if rule.Scope == nil {
			return semantic.Validation{}, &RuleConversionIssue{
				RuleID: rule.ID, Span: rule.PredicateSpan,
				Msg: fmt.Sprintf("%s requires a scope; none was authored", kind),
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
