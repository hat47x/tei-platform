package syntax

import (
	"fmt"

	"tei/internal/core"
)

// This file reads rule declarations (tei.rule-authoring/v0) from the definition
// regions of a TEI Markdown source. It adds no grammar: each region is parsed
// by ParseRuleAuthoringAnchored, the same entry point a standalone rule source
// uses, so a rule means the same thing in both places.
//
// Each region is its own namespace unit. A namespace written in one region
// does not reach the next one, so pasting a region into another document never
// changes which namespace a rule belongs to.

// RuleAuthoringMarkdownRegion is one region and the rules read from it. Every
// span is a byte range of the whole document.
type RuleAuthoringMarkdownRegion struct {
	Region MarkdownRegion
	Rules  RuleAuthoringAnchored
}

// ParseRuleAuthoringMarkdown reads every definition region of src. The result
// is a candidate: it is not accepted meaning and carries no authority.
//
// A rule identity declared in two regions is an error at the second
// declaration. Neither one is chosen by reading order.
func ParseRuleAuthoringMarkdown(src string) ([]RuleAuthoringMarkdownRegion, error) {
	regions, err := ScanTEIMarkdown(src)
	if err != nil {
		return nil, err
	}
	out := make([]RuleAuthoringMarkdownRegion, 0, len(regions))
	seen := map[core.CanonicalID]Anchor{}
	for _, region := range regions {
		parsed, err := ParseRuleAuthoringAnchored(MaskMarkdownRegion(src, region))
		if err != nil {
			return nil, err
		}
		for _, rule := range parsed.File.Rules {
			if !anchorWithin(rule.DeclarationSpan, region.Content) || !anchorWithin(rule.IDSpan, region.Content) {
				return nil, &ParseError{Pos: region.Open.Start, End: region.Open.End,
					Msg: fmt.Sprintf("rule %s was read outside its definition region", rule.ID)}
			}
			if prior, duplicate := seen[rule.ID]; duplicate {
				return nil, &ParseError{Pos: rule.IDSpan.Start, End: rule.IDSpan.End, Msg: fmt.Sprintf(
					"rule %s is declared more than once; the first declaration is at %d:%d",
					rule.ID, prior.Start.Line, prior.Start.Column)}
			}
			seen[rule.ID] = rule.IDSpan
		}
		out = append(out, RuleAuthoringMarkdownRegion{Region: region, Rules: parsed})
	}
	return out, nil
}

func anchorWithin(inner, outer Anchor) bool {
	return inner.Start.Offset >= outer.Start.Offset && inner.End.Offset <= outer.End.Offset
}
