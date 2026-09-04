package syntax

import (
	"tei/internal/core"
	"tei/internal/semantic"
)

// parseSemanticRelationKeyBlockは、Relationのcardinality直後に任意で続くkey blockを読み取る。
// blockがない既存Relationはそのまま受理し、blockを書いた場合だけ一件以上の明示的なkey対応を要求する。
func (p *Parser) parseSemanticRelationKeyBlock(relationToken Token) ([]semantic.RelationKey, error) {
	next, err := p.peek()
	if err != nil {
		return nil, err
	}
	if next.Kind != TokenLBrace {
		return nil, nil
	}
	_, _ = p.next()

	var keys []semantic.RelationKey
	for {
		keyToken, err := p.peek()
		if err != nil {
			return nil, err
		}
		if keyToken.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		if _, err := p.expectText("key"); err != nil {
			return nil, err
		}

		sourceToken, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		sourceAttribute, err := core.ParseCanonicalID(sourceToken.Text)
		if err != nil {
			return nil, p.errAt(sourceToken, err.Error())
		}

		if _, err := p.expect(TokenArrow); err != nil {
			return nil, err
		}

		targetToken, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		targetAttribute, err := core.ParseCanonicalID(targetToken.Text)
		if err != nil {
			return nil, p.errAt(targetToken, err.Error())
		}

		keys = append(keys, semantic.RelationKey{
			SourceAttribute: sourceAttribute,
			TargetAttribute: targetAttribute,
		})
	}

	if len(keys) == 0 {
		return nil, p.errAt(relationToken, "relation key block requires at least one key mapping")
	}
	return keys, nil
}
