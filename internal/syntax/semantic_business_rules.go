package syntax

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// Explicit Semantic Data business-rule declarations have nested blocks to
// delimit their attribute sets without relying on line boundaries:
//
//	candidate-key BusinessIdentity { field sales.order_line.order_id ... }
//	functional-dependency ItemOfLine {
//	  determinant sales.order_line.order_id
//	  dependent sales.order_line.item_id
//	}
//
// They do not infer physical PKs, relation keys, or runtime truth.
func (p *Parser) parseSemanticRuleIdentity(dataID core.CanonicalID) (core.CanonicalID, string, error) {
	tok, err := p.expect(TokenIdent)
	if err != nil {
		return "", "", err
	}
	segment, err := core.SymbolToSegment(tok.Text)
	if err != nil {
		return "", "", p.errAt(tok, err.Error())
	}
	id, err := core.ParseCanonicalID(string(dataID) + "." + segment)
	if err != nil {
		return "", "", p.errAt(tok, err.Error())
	}
	return id, tok.Text, nil
}

func (p *Parser) parseSemanticRuleAttribute(dataID core.CanonicalID, seen map[core.CanonicalID]bool) (core.CanonicalID, error) {
	tok, err := p.expect(TokenIdent)
	if err != nil {
		return "", err
	}
	id, err := core.ParseCanonicalID(tok.Text)
	if err != nil {
		return "", p.errAt(tok, err.Error())
	}
	if len(string(id)) <= len(string(dataID))+1 || string(id)[:len(string(dataID))+1] != string(dataID)+"." {
		return "", p.errAt(tok, fmt.Sprintf("rule attribute %s must belong to owning Data %s", id, dataID))
	}
	if seen[id] {
		return "", p.errAt(tok, fmt.Sprintf("duplicate rule attribute %s", id))
	}
	seen[id] = true
	return id, nil
}

func (p *Parser) parseSemanticCandidateKey(dataID core.CanonicalID) (semantic.CandidateKey, error) {
	_, _ = p.next() // candidate-key
	id, symbol, err := p.parseSemanticRuleIdentity(dataID)
	if err != nil {
		return semantic.CandidateKey{}, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return semantic.CandidateKey{}, err
	}
	key := semantic.CandidateKey{ID: id, Symbol: symbol}
	seen := map[core.CanonicalID]bool{}
	for {
		t, err := p.peek()
		if err != nil {
			return semantic.CandidateKey{}, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		if t.Kind == TokenEOF {
			return semantic.CandidateKey{}, p.errAt(t, "unterminated candidate-key declaration")
		}
		if _, err := p.expectText("field"); err != nil {
			return semantic.CandidateKey{}, err
		}
		attr, err := p.parseSemanticRuleAttribute(dataID, seen)
		if err != nil {
			return semantic.CandidateKey{}, err
		}
		key.Attributes = append(key.Attributes, attr)
	}
	if len(key.Attributes) == 0 {
		return semantic.CandidateKey{}, fmt.Errorf("candidate-key %s must have at least one field", id)
	}
	return key, nil
}

func (p *Parser) parseSemanticFunctionalDependency(dataID core.CanonicalID) (semantic.FunctionalDependency, error) {
	_, _ = p.next() // functional-dependency
	id, symbol, err := p.parseSemanticRuleIdentity(dataID)
	if err != nil {
		return semantic.FunctionalDependency{}, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return semantic.FunctionalDependency{}, err
	}
	fd := semantic.FunctionalDependency{ID: id, Symbol: symbol}
	determinants, dependents := map[core.CanonicalID]bool{}, map[core.CanonicalID]bool{}
	seenDependent := false
	for {
		t, err := p.peek()
		if err != nil {
			return semantic.FunctionalDependency{}, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		if t.Kind == TokenEOF {
			return semantic.FunctionalDependency{}, p.errAt(t, "unterminated functional-dependency declaration")
		}
		switch t.Text {
		case "determinant":
			if seenDependent {
				return semantic.FunctionalDependency{}, p.errAt(t, "determinants must precede dependents")
			}
			_, _ = p.next()
			attr, err := p.parseSemanticRuleAttribute(dataID, determinants)
			if err != nil {
				return semantic.FunctionalDependency{}, err
			}
			fd.Determinants = append(fd.Determinants, attr)
		case "dependent":
			seenDependent = true
			_, _ = p.next()
			attr, err := p.parseSemanticRuleAttribute(dataID, dependents)
			if err != nil {
				return semantic.FunctionalDependency{}, err
			}
			fd.Dependents = append(fd.Dependents, attr)
		default:
			return semantic.FunctionalDependency{}, p.errAt(t, "expected determinant or dependent")
		}
	}
	if len(fd.Determinants) == 0 || len(fd.Dependents) == 0 {
		return semantic.FunctionalDependency{}, fmt.Errorf("functional-dependency %s requires determinants and dependents", id)
	}
	return fd, nil
}

// skipSemanticBusinessRuleBody walks the same bounded nested block grammar
// for source anchors. The canonical parser has already checked attribute
// identity, duplicates, and the mandatory determinant/dependent sets.
func skipSemanticBusinessRuleBody(lx *Lexer, kind string) error {
	for {
		t, err := lx.Next()
		if err != nil {
			return err
		}
		if t.Kind == TokenRBrace {
			return nil
		}
		if t.Kind == TokenEOF {
			return fmt.Errorf("unterminated %s source anchor block", kind)
		}
		switch kind {
		case "candidate-key":
			if t.Kind != TokenIdent || t.Text != "field" {
				return fmt.Errorf("expected candidate-key field at %d:%d", t.Start.Line, t.Start.Column)
			}
		case "functional-dependency":
			if t.Kind != TokenIdent || (t.Text != "determinant" && t.Text != "dependent") {
				return fmt.Errorf("expected functional-dependency determinant or dependent at %d:%d", t.Start.Line, t.Start.Column)
			}
		default:
			return fmt.Errorf("unsupported rule kind %s", kind)
		}
		if _, err := expectLexerIdent(lx, "business rule attribute identity"); err != nil {
			return err
		}
	}
}
