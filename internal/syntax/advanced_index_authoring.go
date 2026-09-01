package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/environment"
)

// ParseAdvancedIndex parses one standalone Advanced Index declaration. The
// same body parser is also used from ParsePersistence so standalone and nested
// authoring cannot drift apart.
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
	index, err := p.parseAdvancedIndexBody(name, kind.Text == "unique-index")
	if err != nil {
		return index, err
	}
	return index, p.expectEOF()
}

// parseAdvancedIndexBody parses the block after an already consumed Index name.
// It intentionally does not consume EOF because Persistence embeds the block.
func (p *Parser) parseAdvancedIndexBody(name Token, unique bool) (persistence.IndexMapping, error) {
	index := persistence.IndexMapping{Name: name.Text, Unique: unique}
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
			expression, _, err := p.parseProviderExpressionBlock(false)
			if err != nil {
				return index, err
			}
			index.Predicate = &expression
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
	return index, nil
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
		expression, key, err := p.parseProviderExpressionBlock(true)
		if err != nil {
			return persistence.IndexKeyMapping{}, err
		}
		key.Expression = &expression
		return key, nil
	default:
		return persistence.IndexKeyMapping{}, p.errAt(kind, "index key requires attribute or expression")
	}
}

func (p *Parser) parseProviderExpressionBlock(withKeyOptions bool) (persistence.ProviderExpressionMapping, persistence.IndexKeyMapping, error) {
	open, err := p.expect(TokenLBrace)
	if err != nil {
		return persistence.ProviderExpressionMapping{}, persistence.IndexKeyMapping{}, err
	}
	var expression persistence.ProviderExpressionMapping
	var key persistence.IndexKeyMapping
	providerSet, dialectSet, sourceSet := false, false, false
	collationSet, operatorClassSet, orderSet, nullsSet := false, false, false, false
	for {
		t, err := p.peek()
		if err != nil {
			return expression, key, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "provider":
			if providerSet {
				return expression, key, p.errAt(t, "provider may be declared only once")
			}
			providerSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return expression, key, err
			}
			provider := environment.DatabaseProvider(value.Text)
			if !provider.Known() {
				return expression, key, p.errAt(value, fmt.Sprintf("unknown database provider %q", value.Text))
			}
			expression.Provider = provider
		case "dialect":
			if dialectSet {
				return expression, key, p.errAt(t, "dialect may be declared only once")
			}
			dialectSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return expression, key, err
			}
			expression.Dialect = strings.TrimSpace(value.Text)
		case "source":
			if sourceSet {
				return expression, key, p.errAt(t, "source may be declared only once")
			}
			sourceSet = true
			_, _ = p.next()
			value, err := p.expect(TokenString)
			if err != nil {
				return expression, key, err
			}
			expression.Source = value.Text
		case "collation":
			if !withKeyOptions {
				return expression, key, p.errAt(t, "collation is valid only inside an expression key")
			}
			if collationSet {
				return expression, key, p.errAt(t, "collation may be declared only once")
			}
			collationSet = true
			_, _ = p.next()
			value, err := p.expect(TokenString)
			if err != nil {
				return expression, key, err
			}
			key.Collation = value.Text
		case "operator-class":
			if !withKeyOptions {
				return expression, key, p.errAt(t, "operator-class is valid only inside an expression key")
			}
			if operatorClassSet {
				return expression, key, p.errAt(t, "operator-class may be declared only once")
			}
			operatorClassSet = true
			_, _ = p.next()
			value, err := p.expect(TokenString)
			if err != nil {
				return expression, key, err
			}
			key.OperatorClass = value.Text
		case "order":
			if !withKeyOptions {
				return expression, key, p.errAt(t, "order is valid only inside an expression key")
			}
			if orderSet {
				return expression, key, p.errAt(t, "order may be declared only once")
			}
			orderSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return expression, key, err
			}
			order := persistence.IndexOrder(value.Text)
			if !order.Valid() || order == "" {
				return expression, key, p.errAt(value, fmt.Sprintf("unknown index order %q", value.Text))
			}
			key.Order = order
		case "nulls":
			if !withKeyOptions {
				return expression, key, p.errAt(t, "nulls is valid only inside an expression key")
			}
			if nullsSet {
				return expression, key, p.errAt(t, "nulls may be declared only once")
			}
			nullsSet = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return expression, key, err
			}
			nulls := persistence.IndexNullsOrder(value.Text)
			if !nulls.Valid() || nulls == "" {
				return expression, key, p.errAt(value, fmt.Sprintf("unknown index NULL ordering %q", value.Text))
			}
			key.NullsOrder = nulls
		default:
			if withKeyOptions {
				return expression, key, p.errAt(t, "expected provider, dialect, source, collation, operator-class, order, or nulls")
			}
			return expression, key, p.errAt(t, "expected provider, dialect, or source")
		}
	}
	if !providerSet || !dialectSet || !sourceSet {
		return expression, key, p.errAt(open, "provider expression requires provider, dialect, and source")
	}
	if expression.Dialect == "" || strings.TrimSpace(expression.Source) == "" {
		return expression, key, p.errAt(open, "provider expression dialect and source must not be empty")
	}
	return expression, key, nil
}

func (p *Parser) parseIndexKeyOptions(key persistence.IndexKeyMapping) (persistence.IndexKeyMapping, error) {
	collationSet, operatorClassSet, orderSet, nullsSet := false, false, false, false
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

// AdvancedIndexIdentifierAnchor returns the exact Index name token for a
// standalone Advanced Index declaration.
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

// PersistenceIndexIdentifierAnchor locates a nested simple or Advanced Index
// declaration without putting source path information into Index identity.
func PersistenceIndexIdentifierAnchor(src, expectedMappingID, expectedIndexName string) (Anchor, error) {
	lx := NewLexer(src)
	keyword, err := lx.Next()
	if err != nil {
		return Anchor{}, err
	}
	if keyword.Kind != TokenIdent || keyword.Text != "persist" {
		return Anchor{}, fmt.Errorf("expected persist declaration at %d:%d", keyword.Start.Line, keyword.Start.Column)
	}
	mapping, err := lx.Next()
	if err != nil {
		return Anchor{}, err
	}
	if mapping.Kind != TokenIdent || mapping.Text != expectedMappingID {
		return Anchor{}, fmt.Errorf("persistence identifier %q does not match expected subject %q", mapping.Text, expectedMappingID)
	}
	open, err := lx.Next()
	if err != nil {
		return Anchor{}, err
	}
	if open.Kind != TokenLBrace {
		return Anchor{}, fmt.Errorf("expected persistence body at %d:%d", open.Start.Line, open.Start.Column)
	}

	depth := 1
	for depth > 0 {
		t, err := lx.Next()
		if err != nil {
			return Anchor{}, err
		}
		switch t.Kind {
		case TokenLBrace:
			depth++
		case TokenRBrace:
			depth--
		case TokenIdent:
			if depth != 1 || (t.Text != "index" && t.Text != "unique-index") {
				continue
			}
			name, err := lx.Next()
			if err != nil {
				return Anchor{}, err
			}
			if name.Kind != TokenIdent {
				return Anchor{}, fmt.Errorf("expected index identifier at %d:%d", name.Start.Line, name.Start.Column)
			}
			if name.Text == expectedIndexName {
				return Anchor{Start: name.Start, End: name.End}, nil
			}
		}
	}
	return Anchor{}, fmt.Errorf("persistence index %s/%s has no syntax anchor", expectedMappingID, expectedIndexName)
}
