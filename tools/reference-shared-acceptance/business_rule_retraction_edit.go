package main

import (
	"fmt"
	"strings"

	"tei/internal/syntax"
)

// removeExactBusinessRuleBlock removes only the top-level rule body identified
// by the current canonical semantic parser and its source identifier anchor.
// No regex deletion, physical PK/FK inference, or silent adjacent text removal.
func removeExactBusinessRuleBlock(content []byte, op Operation) ([]byte, string, error) {
	spec := op.RemoveBusinessRule
	kind := businessRuleRetractionKind(op)
	src := string(content)
	parsed, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		return nil, "", fmt.Errorf("cannot identify current semantic declaration: %w", err)
	}
	var anchorStart, anchorEnd int
	found := false
	for _, anchor := range parsed.Anchors {
		if anchor.Kind == "semantic-"+kind && anchor.ID == spec.ID {
			if found {
				return nil, "", fmt.Errorf("rule %s has multiple source anchors", spec.ID)
			}
			found = true
			anchorStart, anchorEnd = anchor.Anchor.Start.Offset, anchor.Anchor.End.Offset
		}
	}
	if !found || anchorStart < 0 || anchorEnd > len(src) || src[anchorStart:anchorEnd] != spec.Symbol {
		return nil, "", fmt.Errorf("rule %s has no exact current source identifier anchor", spec.ID)
	}
	// The anchored parser has already recognized the full rule and its owning
	// Data. Match only the keyword immediately preceding this exact token.
	lx := syntax.NewLexer(src)
	var prev syntax.Token
	var start, end int
	matched := false
	for {
		tok, err := lx.Next()
		if err != nil {
			return nil, "", err
		}
		if tok.Kind == syntax.TokenEOF {
			break
		}
		if tok.Start.Offset != anchorStart || tok.End.Offset != anchorEnd {
			prev = tok
			continue
		}
		if prev.Kind != syntax.TokenIdent || prev.Text != kind {
			return nil, "", fmt.Errorf("rule %s anchor is not preceded by %q keyword", spec.ID, kind)
		}
		start = strings.LastIndex(src[:prev.Start.Offset], "\n") + 1
		if strings.TrimSpace(src[start:prev.Start.Offset]) != "" {
			return nil, "", fmt.Errorf("rule %s is not declared at the start of its own line", spec.ID)
		}
		open, err := lx.Next()
		if err != nil || open.Kind != syntax.TokenLBrace {
			return nil, "", fmt.Errorf("rule %s has no standalone braced body", spec.ID)
		}
		depth := 1
		for depth > 0 {
			next, err := lx.Next()
			if err != nil || next.Kind == syntax.TokenEOF {
				return nil, "", fmt.Errorf("rule %s body is not closed", spec.ID)
			}
			if next.Kind == syntax.TokenLBrace {
				depth++
			} else if next.Kind == syntax.TokenRBrace {
				depth--
			}
			if depth == 0 {
				end = next.End.Offset
			}
		}
		lineEnd := strings.IndexByte(src[end:], '\n')
		if lineEnd < 0 {
			if strings.TrimSpace(src[end:]) != "" {
				return nil, "", fmt.Errorf("rule %s has trailing content after its closing brace", spec.ID)
			}
			end = len(src)
		} else {
			if strings.TrimSpace(src[end:end+lineEnd]) != "" {
				return nil, "", fmt.Errorf("rule %s has trailing content after its closing brace", spec.ID)
			}
			end += lineEnd + 1
		}
		matched = true
		break
	}
	if !matched || start >= end {
		return nil, "", fmt.Errorf("rule %s source range could not be established", spec.ID)
	}
	removed := src[start:end]
	return []byte(src[:start] + src[end:]), removed, nil
}
