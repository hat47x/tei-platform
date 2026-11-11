package syntax

import (
	"tei/internal/semantic"
)

// This file projects the rule authoring parse into the shared source-anchor
// vocabulary. The anchors come from the same parse that produced the value, so
// no second lexer pass can disagree about an offset.
//
// It reuses the Subject kinds the existing semantic path already declares
// ("semantic" and "semantic-validation"). No new Subject kind is added here:
// finer spans (subject, predicate, scope) stay on the parsed declaration and
// are used by diagnostics, not by Source Index identity.

// RuleAnchors returns the anchors the rule surface declares: the namespace and
// one per rule identity.
func RuleAnchors(file RuleAuthoringFile) []SubjectAnchor {
	anchors := make([]SubjectAnchor, 0, 1+len(file.Rules))
	anchors = append(anchors, SubjectAnchor{
		Kind: "semantic", ID: string(file.Namespace), Anchor: file.NamespaceSpan,
	})
	for _, rule := range file.Rules {
		anchors = append(anchors, SubjectAnchor{
			Kind: "semantic-validation", ID: string(rule.ID), Anchor: rule.IDSpan,
		})
	}
	return anchors
}

// RuleAuthoringAnchored is one parse that yields the typed candidate, the
// conversion issues, and the source anchors together. The parts cannot
// disagree because the anchors are built from the spans the parser recorded,
// not from a re-lex.
type RuleAuthoringAnchored struct {
	File    RuleAuthoringFile
	Value   semantic.Namespace
	Issues  []RuleConversionIssue
	Anchors []SubjectAnchor
}

// ParseRuleAuthoringAnchored parses, converts, and anchors in one pass.
func ParseRuleAuthoringAnchored(src string) (RuleAuthoringAnchored, error) {
	file, err := ParseRuleAuthoringFile(src)
	if err != nil {
		return RuleAuthoringAnchored{}, err
	}
	value, issues := RuleNamespace(file)
	return RuleAuthoringAnchored{
		File:    file,
		Value:   value,
		Issues:  issues,
		Anchors: RuleAnchors(file),
	}, nil
}
