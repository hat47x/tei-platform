package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

func (p *Parser) parseSemanticValidation(namespace core.CanonicalID) (semantic.Validation, error) {
	_, _ = p.next() // validation

	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Validation{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return semantic.Validation{}, p.errAt(idTok, err.Error())
	}
	if !strings.HasPrefix(string(id), string(namespace)+".") {
		return semantic.Validation{}, p.errAt(idTok, "validation canonical id must belong to the declared semantic namespace")
	}

	kindTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Validation{}, err
	}
	out := semantic.Validation{ID: id, Kind: semantic.ValidationKind(kindTok.Text)}
	if !out.Kind.Valid() {
		return out, p.errAt(kindTok, fmt.Sprintf("unsupported semantic validation kind %q", kindTok.Text))
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return out, err
	}

	var hasAttribute, hasRelation bool
	for {
		t, err := p.peek()
		if err != nil {
			return out, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "attribute":
			if hasAttribute {
				return out, p.errAt(t, "attribute is declared more than once")
			}
			hasAttribute = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Attribute, err = core.ParseCanonicalID(value.Text)
			if err != nil {
				return out, p.errAt(value, err.Error())
			}
		case "relation":
			if hasRelation {
				return out, p.errAt(t, "relation is declared more than once")
			}
			hasRelation = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Relation, err = core.ParseCanonicalID(value.Text)
			if err != nil {
				return out, p.errAt(value, err.Error())
			}
		default:
			return out, p.errAt(t, "expected attribute or relation")
		}
	}

	if out.Attribute == "" {
		return out, fmt.Errorf("validation %s requires attribute", out.ID)
	}
	switch out.Kind {
	case semantic.ValidationAttributePositive:
		if out.Relation != "" {
			return out, fmt.Errorf("attribute-positive validation %s does not accept relation", out.ID)
		}
	case semantic.ValidationRelationAttributeUnique:
		if out.Relation == "" {
			return out, fmt.Errorf("relation-attribute-unique validation %s requires relation", out.ID)
		}
	}
	return out, nil
}
