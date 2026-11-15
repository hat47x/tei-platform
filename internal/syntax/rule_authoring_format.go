package syntax

import (
	"errors"
	"fmt"
	"strings"

	"tei/internal/semantic"
)

// Rule formatting is an explicit operation. It never runs on save, and it never
// rewrites a source that carries a comment: relocating a comment is a decision
// this package cannot make from the syntax alone, so it reports that case
// instead of guessing.
var (
	// ErrRuleFormatUnsupported means the source is valid but carries a part
	// this formatter does not know how to place (comments today).
	ErrRuleFormatUnsupported = errors.New("rule source has a part the formatter cannot place")
	// ErrRuleFormatMeaningChanged means the formatted text would parse to a
	// different value. The formatter must never emit that text.
	ErrRuleFormatMeaningChanged = errors.New("rule formatting changed the parsed meaning")
)

// FormatRuleAuthoring renders a canonical layout of the rule surface.
//
// The layout is fixed: one namespace line, then one declaration per rule with
// the subject clause on its own indented line. Names and references are copied
// from the authored spans, so a spelling the caller wrote is not replaced by a
// derived spelling.
//
// The result is verified before it is returned: it is parsed again and
// compared to the input value. A formatter that would change meaning fails
// instead of returning.
func FormatRuleAuthoring(src string) (string, error) {
	file, err := ParseRuleAuthoringFile(src)
	if err != nil {
		return "", err
	}
	if hasRuleComment(src) {
		return "", ErrRuleFormatUnsupported
	}
	value, issues := RuleNamespace(file)
	if len(issues) != 0 {
		// A source whose references cannot be resolved has no canonical
		// rendering yet: changing its bytes would leave the author with a
		// tidier but still broken declaration.
		return "", fmt.Errorf("%w: %v", ErrRuleFormatUnsupported, issues[0])
	}

	var b strings.Builder
	namespaceText := string(file.Namespace)
	fmt.Fprintf(&b, "namespace %s\n\n", namespaceText)
	for _, rule := range file.Rules {
		fmt.Fprintf(&b, "rule %s:\n  %s is %s", idText(src, rule), rule.Subject.Text, rule.Predicate)
		if rule.Scope != nil {
			fmt.Fprintf(&b, " per %s", rule.Scope.Text)
		}
		b.WriteString("\n\n")
	}
	formatted := strings.TrimRight(b.String(), "\n") + "\n"

	reparsed, err := ParseRuleAuthoringFile(formatted)
	if err != nil {
		return "", fmt.Errorf("%w: formatted text does not parse: %v", ErrRuleFormatMeaningChanged, err)
	}
	reparsedValue, reparsedIssues := RuleNamespace(reparsed)
	if len(reparsedIssues) != 0 {
		return "", fmt.Errorf("%w: formatted text has conversion issues: %v", ErrRuleFormatMeaningChanged, reparsedIssues[0])
	}
	if !sameValidations(value.Validations, reparsedValue.Validations) {
		return "", ErrRuleFormatMeaningChanged
	}
	return formatted, nil
}

// idText returns the identity exactly as the author wrote it. The parsed ID is
// canonical and lower-cased for a bare symbol, so using it here would silently
// rename `OrderQuantityPositive` to `order_quantity_positive`.
func idText(src string, rule RuleAuthoring) string {
	if rule.IDSpan.Start.Offset >= rule.IDSpan.End.Offset || rule.IDSpan.End.Offset > len(src) {
		return string(rule.ID)
	}
	return src[rule.IDSpan.Start.Offset:rule.IDSpan.End.Offset]
}

func sameValidations(left, right []semantic.Validation) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// hasRuleComment reports whether the source carries a comment marker. It looks
// at the authored bytes, so a `#` inside a quoted string in another source
// shape cannot be confused with this surface's comment. This surface has no
// string literals today, so a marker anywhere outside the identifiers is a
// comment.
func hasRuleComment(src string) bool {
	return strings.Contains(src, "#") || strings.Contains(src, "//")
}

// RuleSurfaceFromValidation renders one existing validation as the rule
// surface, for comparison and migration preview only. It is not a migration:
// the caller decides whether a Source is rewritten, and no file is touched here.
//
// The rendering is checked by parsing it back and comparing the resulting
// value to the input. A validation this surface cannot express (a future kind)
// is refused instead of being approximated.
func RuleSurfaceFromValidation(namespace string, validation semantic.Validation) (string, error) {
	predicate, ok := rulePredicateForKind(validation.Kind)
	if !ok {
		return "", fmt.Errorf("%w: validation kind %q has no rule predicate", ErrRuleFormatUnsupported, validation.Kind)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "namespace %s\n\n", namespace)
	fmt.Fprintf(&b, "rule %s:\n  %s is %s", validation.ID, validation.Attribute, predicate)
	if validation.Kind == semantic.ValidationRelationUnique {
		if validation.Relation == "" {
			return "", fmt.Errorf("%w: relation-unique validation %s has no relation", ErrRuleFormatUnsupported, validation.ID)
		}
		fmt.Fprintf(&b, " per %s", validation.Relation)
	}
	b.WriteString("\n")

	file, err := ParseRuleAuthoringFile(b.String())
	if err != nil {
		return "", fmt.Errorf("%w: rendering does not parse: %v", ErrRuleFormatUnsupported, err)
	}
	value, issues := RuleNamespace(file)
	if len(issues) != 0 {
		return "", fmt.Errorf("%w: rendering has conversion issues: %v", ErrRuleFormatUnsupported, issues[0])
	}
	if len(value.Validations) != 1 {
		return "", fmt.Errorf("%w: rendering produced %d validations", ErrRuleFormatUnsupported, len(value.Validations))
	}
	got := value.Validations[0]
	// Symbol is derived from the surface spelling, so it can differ from the
	// existing declaration's symbol. Every meaning-bearing field must match.
	if got.ID != validation.ID || got.Kind != validation.Kind ||
		got.Attribute != validation.Attribute || got.Relation != validation.Relation {
		return "", fmt.Errorf("%w: rendering changed the validation meaning", ErrRuleFormatMeaningChanged)
	}
	return b.String(), nil
}

func rulePredicateForKind(kind semantic.ValidationKind) (RulePredicate, bool) {
	switch kind {
	case semantic.ValidationPositive:
		return RulePredicatePositive, true
	case semantic.ValidationRelationUnique:
		return RulePredicateUnique, true
	default:
		return "", false
	}
}
