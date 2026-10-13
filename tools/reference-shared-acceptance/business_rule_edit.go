package main

import (
	"fmt"
	"strings"

	"tei/internal/syntax"
)

func renderBusinessRuleAddition(op Operation, newline string) string {
	var b strings.Builder
	switch op.Op {
	case opAddCandidateKey:
		spec := op.AddCandidateKey
		fmt.Fprintf(&b, "    candidate-key %s {%s", spec.Symbol, newline)
		for _, attr := range spec.Attributes {
			fmt.Fprintf(&b, "        field %s%s", attr, newline)
		}
	case opAddFunctionalDependency:
		spec := op.AddFunctionalDependency
		fmt.Fprintf(&b, "    functional-dependency %s {%s", spec.Symbol, newline)
		for _, attr := range spec.Determinants {
			fmt.Fprintf(&b, "        determinant %s%s", attr, newline)
		}
		for _, attr := range spec.Dependents {
			fmt.Fprintf(&b, "        dependent %s%s", attr, newline)
		}
	default:
		return ""
	}
	fmt.Fprintf(&b, "    }%s", newline)
	return b.String()
}

// insertBusinessRuleIntoDataBlock inserts before the *outer* Data closing
// brace. An explicitly authored nested key or FD block has its own braces;
// reusing the legacy flat Relation inserter would stop at the wrong brace.
// Exact source is preserved except for the authorized insertion.
func insertBusinessRuleIntoDataBlock(content []byte, dataSymbol string, op Operation) ([]byte, string, error) {
	src := string(content)
	lx := syntax.NewLexer(src)
	var previous, beforePrevious syntax.Token
	found := false
	depth := 0
	closing := -1
	for {
		t, err := lx.Next()
		if err != nil {
			return nil, "", err
		}
		if t.Kind == syntax.TokenEOF {
			break
		}
		if !found {
			if t.Kind == syntax.TokenLBrace && previous.Text == dataSymbol && beforePrevious.Text == "data" {
				found = true
				depth = 1
			}
		} else {
			switch t.Kind {
			case syntax.TokenLBrace:
				depth++
			case syntax.TokenRBrace:
				depth--
				if depth == 0 {
					closing = t.Start.Offset
				}
			}
		}
		if closing >= 0 {
			break
		}
		beforePrevious, previous = previous, t
	}
	if closing < 0 {
		return nil, "", fmt.Errorf("Data %s outer block is missing or unterminated", dataSymbol)
	}
	lineStart := strings.LastIndex(src[:closing], "\n") + 1
	if strings.TrimSpace(src[lineStart:closing]) != "" {
		return nil, "", fmt.Errorf("Data %s closing brace is not on its own line", dataSymbol)
	}
	newline := "\n"
	if strings.Contains(src, "\r\n") {
		newline = "\r\n"
	}
	text := renderBusinessRuleAddition(op, newline)
	if text == "" {
		return nil, "", fmt.Errorf("unrecognized typed business-rule insertion %s", op.Op)
	}
	return []byte(src[:lineStart] + text + src[lineStart:]), text, nil
}
