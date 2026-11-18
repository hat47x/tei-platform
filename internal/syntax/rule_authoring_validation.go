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
	// The declarations are indexed before the rules are converted: a rule that
	// names an input no operator declares must be diagnosed, and that check
	// needs the declarations.
	declaredInputs := declaredOperatorInputNames(file.Operators)
	declaredMappings := declaredOperatorMappings(file.Operators)
	for _, rule := range file.Rules {
		validation, issue := ruleToValidationWithInputs(rule, declaredInputs, declaredMappings)
		if issue != nil {
			issues = append(issues, *issue)
			continue
		}
		out.Validations = append(out.Validations, validation)
	}
	for _, operator := range file.Operators {
		// A range or a refutation statement has no place in the meaning yet.
		// Converting would produce an operator that silently ignores what the
		// declaration says about how it is asked.
		if operator.OfEachRow {
			issues = append(issues, RuleConversionIssue{
				RuleID: operator.ID, Span: operator.OfEachRowSpan,
				Msg: "the meaning has no place for an input read over all rows yet; " +
					"an operator that states \"of each row\" cannot be converted until the normalization carries it",
			})
			continue
		}
		if operator.RefutesOnSomeRows {
			issues = append(issues, RuleConversionIssue{
				RuleID: operator.ID, Span: operator.RefutesOnSomeRowsSpan,
				Msg: "the meaning has no place for a refutation statement yet; " +
					"an operator that states \"refutes on some rows\" cannot be converted until the normalization carries it",
			})
			continue
		}
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
			AbsentAs:  operator.AbsentAs,
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

// declaredOperatorInputNames indexes the input names each declaration gives,
// keyed by operator id. Every declared operator gets an entry, so an operator
// that declares no name is distinguishable from one that is not declared at all.
//
// The unnamed single-input form declares no name of its own, but the reader
// gives it "value" (see OperatorAuthoring.InputName). That name has to be in
// this index too, or a rule that writes `value:` against `takes number` is
// refused as undeclared while the same contract read through
// `takes value: number` passes the name check. Both forms must still refuse
// value transport until semantic normalization can preserve it (#4362).
//
// The name stays out of the emitted semantic.Operator. This index is what the
// rule check reads; the contract continues to carry only what the author wrote,
// so a reader that needs a name for the single input still applies "value".
// declaredOperatorMappings indexes the value-absent mapping each declaration
// gives, keyed by operator id. An operator with no entry is either undeclared in
// this source or declares no mapping; the caller only checks operators it knows.
func declaredOperatorMappings(operators []OperatorAuthoring) map[core.CanonicalID][]string {
	mappings := make(map[core.CanonicalID][]string, len(operators))
	for _, operator := range operators {
		entry := mappings[operator.ID]
		mappings[operator.ID] = entry
		if operator.AbsentAs != "" {
			mappings[operator.ID] = append(entry, operator.AbsentAs)
		}
	}
	return mappings
}

func declaredOperatorInputNames(operators []OperatorAuthoring) map[core.CanonicalID][]string {
	names := make(map[core.CanonicalID][]string, len(operators))
	for _, operator := range operators {
		entry := names[operator.ID]
		names[operator.ID] = entry
		name := operator.InputName
		if name == "" {
			// The unnamed form is the single input the surface already calls
			// "value", so a rule may address it by that name.
			name = "value"
		}
		names[operator.ID] = append(entry, name)
	}
	return names
}

// ruleToValidationWithInputs refuses an assignment that names an input the
// referenced operator does not declare.
//
// An operator that is not declared in this source is left to the Workspace-wide
// resolution rather than refused here: this surface does not know whether it is
// declared elsewhere, and reporting it twice would state the same thing in two
// places.
func ruleToValidationWithInputs(
	rule RuleAuthoring,
	declaredInputs map[core.CanonicalID][]string,
	declaredMappings map[core.CanonicalID][]string,
) (semantic.Validation, *RuleConversionIssue) {
	if rule.Operator == nil || len(rule.Inputs) == 0 {
		return ruleToValidation(rule)
	}
	declared, known := declaredInputs[core.CanonicalID(rule.Operator.Text)]
	if !known {
		// The operator is not declared in this source, so its input names are
		// unknown here; only the value check applies.
		return refuseAssignedInputs(rule)
	}
	for _, assignment := range rule.Inputs {
		if !inputNameDeclared(declared, assignment.Name) {
			return semantic.Validation{}, &RuleConversionIssue{
				RuleID: rule.ID, Span: assignment.NameSpan,
				Msg: fmt.Sprintf(
					"operator %s declares no input named %q; a rule cannot assign an input the operator does not take",
					rule.Operator.Text, assignment.Name),
			}
		}
		if issue := assignmentValueTypeIssue(rule, assignment); issue != nil {
			return semantic.Validation{}, issue
		}
		if assignment.AbsentAs != "" {
			if mappings, known := declaredMappings[core.CanonicalID(rule.Operator.Text)]; known {
				if !inputNameDeclared(mappings, assignment.AbsentAs) {
					return semantic.Validation{}, &RuleConversionIssue{
						RuleID: rule.ID, Span: assignment.AbsentAsSpan,
						Msg: fmt.Sprintf(
							"operator %s declares no value-absent mapping named %q; a rule cannot read an absent value a way the operator never stated",
							rule.Operator.Text, assignment.AbsentAs),
					}
				}
			}
		}
	}
	// An assignment that survives the name check still has nowhere to go:
	// semantic.Validation carries the operator and its subject, not the input
	// values. Converting anyway would produce a rule that reads as the
	// single-input operator and silently ignores the assignment.
	return refuseAssignedInputs(rule)
}

// refuseAssignedInputs refuses a rule that assigns an operator input. The
// meaning has no field for the value yet, so converting would drop it.
func refuseAssignedInputs(rule RuleAuthoring) (semantic.Validation, *RuleConversionIssue) {
	if len(rule.Inputs) == 0 {
		return ruleToValidation(rule)
	}
	return semantic.Validation{}, &RuleConversionIssue{
		RuleID: rule.ID, Span: rule.Inputs[0].NameSpan,
		Msg: "the meaning has no place for an operator input value yet; " +
			"a rule that assigns an input cannot be converted until the normalization carries it",
	}
}

// assignmentValueTypeIssue refuses a value whose type the declared input does
// not take.
//
// Only `number` exists today, so the check is narrow and exact: a numeric input
// takes a canonical attribute id (an id names the item whose value is read) or a
// decimal constant. Anything else - a word, a quoted string, a bare `true` - is a
// type mismatch, and saying so is more useful than reporting it later as an
// unsupported transport.
func assignmentValueTypeIssue(rule RuleAuthoring, assignment RuleInputAssignment) *RuleConversionIssue {
	if !isNumberText(assignment.Value) && !isCanonicalReferenceText(assignment.Value) {
		return &RuleConversionIssue{
			RuleID: rule.ID, Span: assignment.ValueSpan,
			Msg: fmt.Sprintf(
				"input %q takes a number, but %q is neither a decimal number nor an attribute id",
				assignment.Name, assignment.Value),
		}
	}
	return nil
}

// isNumberText reports whether text reads as a decimal number under the same
// rule the operator examples use: digits, an optional leading minus, and an
// optional fraction, nothing else.
func isNumberText(text string) bool {
	if text == "" {
		return false
	}
	digits := text
	if digits[0] == '-' {
		digits = digits[1:]
	}
	seenDigit, seenPoint, fractionDigits := false, false, 0
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		switch {
		case c >= '0' && c <= '9':
			seenDigit = true
			if seenPoint {
				fractionDigits++
			}
		case c == '.' && !seenPoint && seenDigit:
			seenPoint = true
		default:
			return false
		}
	}
	if !seenDigit || seenPoint && fractionDigits == 0 {
		return false
	}
	// "007" is refused by the example reader as ambiguous, so it is refused here.
	return !(len(digits) > 1 && digits[0] == '0' && digits[1] != '.')
}

// isCanonicalReferenceText reports whether text is a multi-segment id, which is
// what an attribute reference looks like. Whether the id resolves is the
// validator's decision, not this surface's.
func isCanonicalReferenceText(text string) bool {
	return strings.Contains(text, ".") && text != "." &&
		!strings.HasPrefix(text, ".") && !strings.HasSuffix(text, ".")
}

func inputNameDeclared(names []string, name string) bool {
	for _, declared := range names {
		if declared == name {
			return true
		}
	}
	return false
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
