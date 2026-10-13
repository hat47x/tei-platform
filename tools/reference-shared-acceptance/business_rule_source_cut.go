package main

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// BusinessRuleSourceCut is a proposal-side byte-span plan, not a Shared
// Acceptance operation. Constructing it neither edits a file nor authorizes
// the retraction of a business assertion.
type BusinessRuleSourceCut struct {
	Kind              string            `json:"kind"`
	DataID            string            `json:"dataId"`
	RuleID            string            `json:"ruleId"`
	DefinitionRange   sourceindex.Range `json:"definitionRange"`
	RemovedRange      sourceindex.Range `json:"removedRange"`
	BeforeFingerprint string            `json:"beforeFingerprint"`
	AfterFingerprint  string            `json:"afterFingerprint"`
	RemovedText       string            `json:"removedText"`
}

// planBusinessRuleSourceCut locates exactly one already-authored canonical
// candidate-key / FD in the parsed owner Data and identifies an entire
// standalone declaration block. It deliberately does not write the resulting
// bytes; revision checks, review and an explicit Accept are separate gates.
func planBusinessRuleSourceCut(content []byte, kind, dataID, ruleID string) (BusinessRuleSourceCut, []byte, error) {
	var empty BusinessRuleSourceCut
	if kind != "candidate-key" && kind != "functional-dependency" {
		return empty, nil, fmt.Errorf("unsupported logical rule kind %q", kind)
	}
	data, err := core.ParseCanonicalID(dataID)
	if err != nil {
		return empty, nil, fmt.Errorf("invalid owning Data %q: %w", dataID, err)
	}
	rule, err := core.ParseCanonicalID(ruleID)
	if err != nil {
		return empty, nil, fmt.Errorf("invalid business rule %q: %w", ruleID, err)
	}
	if !strings.HasPrefix(string(rule), string(data)+".") {
		return empty, nil, fmt.Errorf("rule %s is not owned by Data %s", rule, data)
	}
	parsed, err := syntax.ParseSemanticFileAnchored(string(content))
	if err != nil {
		return empty, nil, fmt.Errorf("semantic source must parse before editing: %w", err)
	}
	ownerCount, ruleCount := 0, 0
	for _, d := range parsed.Value.Data {
		if d.ID != data {
			continue
		}
		ownerCount++
		switch kind {
		case "candidate-key":
			for _, key := range d.CandidateKeys {
				if key.ID == rule {
					ruleCount++
				}
			}
		case "functional-dependency":
			for _, fd := range d.FunctionalDependencies {
				if fd.ID == rule {
					ruleCount++
				}
			}
		}
	}
	if ownerCount != 1 || ruleCount != 1 {
		return empty, nil, fmt.Errorf("rule %s in Data %s must occur exactly once (owners=%d rules=%d)", rule, data, ownerCount, ruleCount)
	}
	var ownerAt, ruleAt int
	ownerAnchors, ruleAnchors := 0, 0
	for _, a := range parsed.Anchors {
		if a.Kind == "semantic-data" && a.ID == dataID {
			ownerAt = a.Anchor.Start.Offset
			ownerAnchors++
		}
		if a.Kind == "semantic-"+kind && a.ID == ruleID {
			ruleAt = a.Anchor.Start.Offset
			ruleAnchors++
		}
	}
	if ownerAnchors != 1 || ruleAnchors != 1 {
		return empty, nil, fmt.Errorf("source anchors for Data %s and rule %s are ambiguous or missing", data, rule)
	}
	tokens, err := businessRuleSourceTokens(content)
	if err != nil {
		return empty, nil, err
	}
	ownerSymbolIndex, ruleSymbolIndex := -1, -1
	for i, t := range tokens {
		if t.Start.Offset == ownerAt {
			ownerSymbolIndex = i
		}
		if t.Start.Offset == ruleAt {
			ruleSymbolIndex = i
		}
	}
	if ownerSymbolIndex < 1 || ruleSymbolIndex < 1 ||
		tokens[ownerSymbolIndex-1].Text != "data" ||
		tokens[ruleSymbolIndex-1].Text != kind ||
		ownerSymbolIndex+1 >= len(tokens) || tokens[ownerSymbolIndex+1].Kind != syntax.TokenLBrace ||
		ruleSymbolIndex+1 >= len(tokens) || tokens[ruleSymbolIndex+1].Kind != syntax.TokenLBrace {
		return empty, nil, fmt.Errorf("rule %s declaration is not bounded by canonical Data and rule blocks", rule)
	}
	ownerOpen := ownerSymbolIndex + 1
	ownerEnd, err := closingBusinessRuleBrace(tokens, ownerOpen)
	if err != nil {
		return empty, nil, err
	}
	ruleOpen := ruleSymbolIndex + 1
	ruleEnd, err := closingBusinessRuleBrace(tokens, ruleOpen)
	if err != nil {
		return empty, nil, err
	}
	if ruleOpen <= ownerOpen || ruleEnd >= ownerEnd {
		return empty, nil, fmt.Errorf("rule %s block does not belong to owning Data %s", rule, data)
	}
	// The rule keyword must be at depth one in its Data. A similarly named
	// token in a nested block cannot be used to cut an unrelated declaration.
	depth := 1
	for i := ownerOpen + 1; i < ruleSymbolIndex-1; i++ {
		switch tokens[i].Kind {
		case syntax.TokenLBrace:
			depth++
		case syntax.TokenRBrace:
			depth--
		}
		if depth < 1 {
			return empty, nil, fmt.Errorf("invalid enclosing Data scope for %s", rule)
		}
	}
	if depth != 1 {
		return empty, nil, fmt.Errorf("rule %s is not a direct Data declaration", rule)
	}
	keywordAt := tokens[ruleSymbolIndex-1].Start.Offset
	start := strings.LastIndex(string(content[:keywordAt]), "\n") + 1
	if strings.TrimSpace(string(content[start:keywordAt])) != "" {
		return empty, nil, fmt.Errorf("rule %s must start on its own line", rule)
	}
	closeOffset := tokens[ruleEnd].End.Offset
	end := closeOffset
	for end < len(content) && (content[end] == ' ' || content[end] == '\t' || content[end] == '\r') {
		end++
	}
	if end < len(content) {
		if content[end] != '\n' {
			return empty, nil, fmt.Errorf("rule %s must end on its own line", rule)
		}
		end++
	}
	// Deleting the last block with no terminal newline is allowed only when
	// it has no sibling text. Everything outside [start,end) remains exact.
	if start >= end || end > len(content) {
		return empty, nil, fmt.Errorf("invalid authored source range for rule %s", rule)
	}
	after := make([]byte, 0, len(content)-(end-start))
	after = append(after, content[:start]...)
	after = append(after, content[end:]...)
	if _, err := syntax.ParseSemanticFileAnchored(string(after)); err != nil {
		return empty, nil, fmt.Errorf("rule cut would leave unparsable semantic source: %w", err)
	}
	idRange, err := sourceindex.RangeFromOffsets(content, tokens[ruleSymbolIndex].Start.Offset, tokens[ruleSymbolIndex].End.Offset)
	if err != nil {
		return empty, nil, err
	}
	cutRange, err := sourceindex.RangeFromOffsets(content, start, end)
	if err != nil {
		return empty, nil, err
	}
	cut := BusinessRuleSourceCut{
		Kind:              kind,
		DataID:            dataID,
		RuleID:            ruleID,
		DefinitionRange:   idRange,
		RemovedRange:      cutRange,
		BeforeFingerprint: sourceindex.Fingerprint(content),
		AfterFingerprint:  sourceindex.Fingerprint(after),
		RemovedText:       string(content[start:end]),
	}
	return cut, after, nil
}

func businessRuleSourceTokens(content []byte) ([]syntax.Token, error) {
	lx := syntax.NewLexer(string(content))
	var tokens []syntax.Token
	for {
		tok, err := lx.Next()
		if err != nil {
			return nil, err
		}
		if tok.Kind == syntax.TokenEOF {
			return tokens, nil
		}
		tokens = append(tokens, tok)
	}
}

func closingBusinessRuleBrace(tokens []syntax.Token, open int) (int, error) {
	if open < 0 || open >= len(tokens) || tokens[open].Kind != syntax.TokenLBrace {
		return 0, fmt.Errorf("expected a rule or Data opening brace")
	}
	depth := 0
	for i := open; i < len(tokens); i++ {
		switch tokens[i].Kind {
		case syntax.TokenLBrace:
			depth++
		case syntax.TokenRBrace:
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated semantic business-rule block")
}
