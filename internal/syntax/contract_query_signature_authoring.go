package syntax

import (
	"tei/internal/core"
	"tei/internal/database"
)

// parseContractQuerySignatureは、Contract Query名の直後に任意で続くsignature blockを読み取る。
// blockがなければ従来どおり名前だけのQueryを返し、既存Contract sourceとの互換性を保つ。
func (p *Parser) parseContractQuerySignature(name string) (database.QueryContract, error) {
	queryContract := database.QueryContract{Name: name}

	next, err := p.peek()
	if err != nil {
		return queryContract, err
	}
	if next.Kind != TokenLBrace {
		return queryContract, nil
	}
	_, _ = p.next()

	hasResult := false
	for {
		t, err := p.peek()
		if err != nil {
			return queryContract, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}

		switch t.Text {
		case "input":
			_, _ = p.next()
			nameTok, err := p.expect(TokenIdent)
			if err != nil {
				return queryContract, err
			}
			inputName, err := core.ParseLocalName(nameTok.Text)
			if err != nil {
				return queryContract, p.errAt(nameTok, err.Error())
			}
			if _, err := p.expectText("type"); err != nil {
				return queryContract, err
			}
			typeTok, err := p.expect(TokenIdent)
			if err != nil {
				return queryContract, err
			}
			typeRef, err := core.ParseCanonicalID(typeTok.Text)
			if err != nil {
				return queryContract, p.errAt(typeTok, err.Error())
			}
			queryContract.Inputs = append(queryContract.Inputs, database.QueryInput{Name: inputName, Type: typeRef})

		case "result":
			if hasResult {
				return queryContract, p.errAt(t, "result is declared more than once")
			}
			hasResult = true
			_, _ = p.next()
			resultTok, err := p.expect(TokenIdent)
			if err != nil {
				return queryContract, err
			}
			queryContract.Result, err = core.ParseCanonicalID(resultTok.Text)
			if err != nil {
				return queryContract, p.errAt(resultTok, err.Error())
			}
			cardinalityTok, err := p.expect(TokenIdent)
			if err != nil {
				return queryContract, err
			}
			queryContract.ResultCardinality = database.QueryResultCardinality(cardinalityTok.Text)

		default:
			return queryContract, p.errAt(t, "expected input or result")
		}
	}

	return queryContract, nil
}
