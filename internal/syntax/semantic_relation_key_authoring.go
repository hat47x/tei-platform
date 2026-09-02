package syntax

import (
	"tei/internal/core"
	"tei/internal/semantic"
)

// parseSemanticRelationKeysは、Relation宣言の直後に続く任意個のkey対応を読み取る。
// keyの記述順は複合keyの意味として保持し、属性名や型から対応を推測しない。
func (p *Parser) parseSemanticRelationKeys() ([]semantic.RelationKey, error) {
	var keys []semantic.RelationKey
	for {
		t, err := p.peek()
		if err != nil {
			return nil, err
		}
		if t.Text != "key" {
			break
		}
		_, _ = p.next()

		sourceTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		source, err := core.ParseCanonicalID(sourceTok.Text)
		if err != nil {
			return nil, p.errAt(sourceTok, err.Error())
		}
		if _, err := p.expect(TokenArrow); err != nil {
			return nil, err
		}
		targetTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		target, err := core.ParseCanonicalID(targetTok.Text)
		if err != nil {
			return nil, p.errAt(targetTok, err.Error())
		}

		keys = append(keys, semantic.RelationKey{
			SourceAttribute: source,
			TargetAttribute: target,
		})
	}
	return keys, nil
}
