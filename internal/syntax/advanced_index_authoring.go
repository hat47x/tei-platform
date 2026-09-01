package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/environment"
)

// ParseAdvancedIndex parses one standalone Advanced Index declaration. It is a
// composable parser entry point for the block form that will be embedded in a
// Persistence Mapping without changing the meaning of the existing simple
// `index NAME on ATTRIBUTE` syntax.
func ParseAdvancedIndex(src string) (persistence.IndexMapping, error) {
	return NewParser(src).ParseAdvancedIndex()
}

// ParseAdvancedIndex parses either `index NAME { ... }` or
// `unique-index NAME { ... }`. Provider-native SQL remains opaque text.
func (p *Parser) ParseAdvancedIndex() (persistence.IndexMapping, error) {
	kind, err := p.expect(TokenIdent)
	if err != nil {
		return persistence.IndexMapping{}, err
	}
	if kind.Text != "index" && kind.Text != "unique-index" {
		return persistence.IndexMapping{}, p.errAt(kind, "expected index or unique-index")
	}
	name, err := p.expect(TokenIdent)
	if err != nil {
		return persistence.IndexMapping{}, err
	}
	index := persistence.IndexMapping{Name: name.Text, Unique: kind.Text == "unique-index"}
	if _, err := p.expect(TokenLBrace); err != nil {
		return index, err
	}

	methodSet := false
	predicateSet := false
	nullTreatmentSet := false
	for {
		t, err := p.peek()
		if err != nil {
			return index, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "method":
			if methodSet {
				return index, p.errAt(t, "index method may be declared only once")
			}
			methodSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return index, err
			}
			index.Method = value.Text
		case "key":
			_, _ = p.next()
			key, err := p.parseAdvancedIndexKey()
			if err != nil {
				return index, err
			}
			index.Keys = append(index.Keys, key)
		case "include":
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return index, err
			}
			attribute, err := core.ParseCanonicalID(value.Text)
			if err != nil {
				return index, p.errAt(value, err.Error())
			}
			index.IncludedAttributes = append(index.IncludedAttributes, attribute)
		case "predicate":
			if predicateSet {
				return index, p.errAt(t, "index predicate may be declared only once")
			}
			predicateSet = true
			_, _ = p.next()
			expression, err := p.parseProviderExpressionBlock(false)
			if err != nil {
				return index, err
			}
			index.Predicate = &expression.Expression
		case "null-treatment":
			if nullTreatmentSet {
				return index, p.errAt(t, "null-treatment may be declared only once")
			}
			nullTreatmentSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return index, err
			}
			treatment := persistence.UniqueNullTreatment(value.Text)
			if !treatment.Valid() || treatment == "" {
				return index, p.errAt(value, fmt.Sprintf("unknown unique NULL treatment %q", value.Text))
			}
			if !index.Unique {
				return index, p.errAt(value, "null-treatment is available only for unique-index")
			}
			index.UniqueNullTreatment = treatment
		default:
			return index, p.errAt(t, "expected method, key, include, predicate, or null-treatment")
		}
	}
	if len(index.Keys) == 0 {
		return index, p.errAt(name, "advanced index requires at least one key")
	}
	return index, p.expectEOF()
}

func (p *Parser) parseAdvancedIndexKey() (persistence.IndexKeyMapping, error) {
	kind, err := p.expect(TokenIdent)
	if err != nil {
		return persistence.IndexKeyMapping{}, err
	}
	switch kind.Text {
	case "attribute":
		value, err := p.expect(TokenIdent)
		if err != nil {
			return persistence.IndexKeyMapping{}, err
		}
		attribute, err := core.ParseCanonicalID(value.Text)
		if err != nil {
			return persistence.IndexKeyMapping{}, p.errAt(value, err.Error())
		}
		key := persistence.IndexKeyMapping{Attribute: attribute}
		next, err := p.peek()
		if err != nil {
			return key, err
		}
		if next.Kind != TokenLBrace {
			return key, nil
		}
		_, _ = p.next()
		return p.parseIndexKeyOptions(key)
	case "expression":
		parsed, err := p.parseProviderExpressionBlock(true)
		if err != nil {
			return persistence.IndexKeyMapping{}, err
		}
		parsed.Key.Expression = &parsed.Expression
		return parsed.Key, nil
	default:
		return persistence.IndexKeyMapping{}, p.errAt(kind, "index key requires attribute or expression")
	}
}

type parsedProviderExpression struct {
	Expression persistence.ProviderExpressionMapping
	Key        persistence.IndexKeyMapping
}

func (p *Parser) parseProviderExpressionBlock(withKeyOptions bool) (parsedProviderExpression, error) {
	if _, err := p.expect(TokenLBrace); err != nil {
		return parsedProviderExpression{}, err
	}
	var out parsedProviderExpression
	providerSet := false
	dialectSet := false
	sourceSet := false
	collationSet := false
	operatorClassSet := false
	orderSet := false
	nullsSet := false
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
		case "provider":
			if providerSet {
				return out, p.errAt(t, "provider may be declared only once")
			}
			providerSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			provider := environment.DatabaseProvider(value.Text)
			if !provider.Known() {
				return out, p.errAt(value, fmt.Sprintf("unknown database provider %q", value.Text))
			}
			out.Expression.Provider = provider
		case "dialect":
			if dialectSet {
				return out, p.errAt(t, "dialect may be declared only once")
			}
			dialectSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Expression.Dialect = strings.TrimSpace(value.Text)
		case "source":
			if sourceSet {
				return out, p.errAt(t, "source may be declared only once")
			}
			sourceSet = true
			_, _ = p.next()
			value, err := p.expect(TokenString)
			if err != nil {
				return out, err
			}
			out.Expression.Source = value.Text
		case "collation":
			if !withKeyOptions {
				return out, p.errAt(t, "collation is valid only inside an expression key")
			}
			if collationSet {
				return out, p.errAt(t, "collation may be declared only once")
			}
			collationSet = true
			_, _ = p.next()
			value, err := p.expect(TokenString)
			if err != nil {
				return out, err
			}
			out.Key.Collation = value.Text
		case "operator-class":
			if !withKeyOptions {
				return out, p.errAt(t, "operator-class is valid only inside an expression key")
			}
			if operatorClassSet {
				return out, p.errAt(t, "operator-class may be declared only once")
			}
			operatorClassSet = true
			_, _ = p.next()
			value, err := p.expect(TokenString)
			if err != nil {
				return out, err
			}
			out.Key.OperatorClass = value.Text
		case "order":
			if !withKeyOptions {
				return out, p.errAt(t, "order is valid only inside an expression key")
			}
			if orderSet {
				return out, p.errAt(t, "order may be declared only once")
			}
			orderSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			order := persistence.IndexOrder(value.Text)
			if !order.Valid() || order == "" {
				return out, p.errAt(value, fmt.Sprintf("unknown index order %q", value.Text))
			}
			out.Key.Order = order
		case "nulls":
			if !withKeyOptions {
				return out, p.errAt(t, "nulls is valid only inside an expression key")
			}
			if nullsSet {
				return out, p.errAt(t, "nulls may be declared only once")
			}
			nullsSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			nulls := persistence.IndexNullsOrder(value.Text)
			if !nulls.Valid() || nulls == "" {
				return out, p.errAt(value, fmt.Sprintf("unknown index NULL ordering %q", value.Text))
			}
			out.Key.NullsOrder = nulls
		default:
			if withKeyOptions {
				return out, p.errAt(t, "expected provider, dialect, source, collation, operator-class, order, or nulls")
			}
			return out, p.errAt(t, "expected provider, dialect, or source")
		}
	}
	if !providerSet || !dialectSet || !sourceSet {
		return out, &ParseError{Pos: p.lx.pos(), End: p.lx.pos(), Msg: "provider expression requires provider, dialect, and source"}
	}
	if strings.TrimSpace(out.Expression.Dialect) == "" || strings.TrimSpace(out.Expression.Source) == "" {
		return out, &ParseError{Pos: p.lx.pos(), End: p.lx.pos(), Msg: "provider expression dialect and source must not be empty"}
	}
	return out, nil
}

func (p *Parser) parseIndexKeyOptions(key persistence.IndexKeyMapping) (persistence.IndexKeyMapping, error) {
	collationSet := false
	operatorClassSet := false
	orderSet := false
	nullsSet := false
	for {
		t, err := p.peek()
		if err != nil {
			return key, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "collation":
			if collationSet {
				return key, p.errAt(t, "collation may be declared only once")
			}
			collationSet = true
			_, _ = p.next()
			value, err := p.expect(TokenString)
			if err != nil {
				return key, err
			}
			key.Collation = value.Text
		case "operator-class":
			if operatorClassSet {
				return key, p.errAt(t, "operator-class may be declared only once")
			}
			operatorClassSet = true
			_, _ = p.next()
			value, err := p.expect(TokenString)
			if err != nil {
				return key, err
			}
			key.OperatorClass = value.Text
		case "order":
			if orderSet {
				return key, p.errAt(t, "order may be declared only once")
			}
			orderSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return key, err
			}
			order := persistence.IndexOrder(value.Text)
			if !order.Valid() || order == "" {
				return key, p.errAt(value, fmt.Sprintf("unknown index order %q", value.Text))
			}
			key.Order = order
		case "nulls":
			if nullsSet {
				return key, p.errAt(t, "nulls may be declared only once")
			}
			nullsSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return key, err
			}
			nulls := persistence.IndexNullsOrder(value.Text)
			if !nulls.Valid() || nulls == "" {
				return key, p.errAt(value, fmt.Sprintf("unknown index NULL ordering %q", value.Text))
			}
			key.NullsOrder = nulls
		default:
			return key, p.errAt(t, "expected collation, operator-class, order, or nulls")
		}
	}
	return key, nil
}

// AdvancedIndexIdentifierAnchor returns the exact Index name token for Source
// Index integration. The physical path remains outside the semantic identity.
func AdvancedIndexIdentifierAnchor(src, expectedName string) (Anchor, error) {
	lx := NewLexer(src)
	kind, err := lx.Next()
	if err != nil {
		return Anchor{}, err
	}
	if kind.Kind != TokenIdent || (kind.Text != "index" && kind.Text != "unique-index") {
		return Anchor{}, fmt.Errorf("expected index or unique-index declaration at %d:%d", kind.Start.Line, kind.Start.Column)
	}
	name, err := lx.Next()
	if err != nil {
		return Anchor{}, err
	}
	if name.Kind != TokenIdent {
		return Anchor{}, fmt.Errorf("expected index identifier at %d:%d", name.Start.Line, name.Start.Column)
	}
	if name.Text != expectedName {
		return Anchor{}, fmt.Errorf("index identifier %q does not match expected subject %q", name.Text, expectedName)
	}
	return Anchor{Start: name.Start, End: name.End}, nil
}
