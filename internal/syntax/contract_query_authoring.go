package syntax

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/database"
)

// ParseContractAuthoringは、従来のContract構文に加え、typed Query signatureを読み取る。
//
// 名前だけの`query find_by_id`は既存Contractとの互換性のためそのまま受理する。
// Queryのinput / result契約を明示する場合はQuery名の後ろにblockを置き、
// Database固有SQLやAPI等の実現方式はこの構文へ持ち込まない。
func (p *Parser) ParseContractAuthoring() (database.Contract, error) {
	if _, err := p.expectText("contract"); err != nil {
		return database.Contract{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return database.Contract{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return database.Contract{}, p.errAt(idTok, err.Error())
	}
	verTok, err := p.expect(TokenIdent)
	if err != nil {
		return database.Contract{}, err
	}
	ver, err := parseVersion(verTok.Text)
	if err != nil {
		return database.Contract{}, p.errAt(verTok, err.Error())
	}

	contract := database.Contract{ID: id, Version: ver}
	if _, err := p.expect(TokenLBrace); err != nil {
		return contract, err
	}

	for {
		t, err := p.peek()
		if err != nil {
			return contract, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}

		switch t.Text {
		case "read", "write":
			kind := t.Text
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return contract, err
			}
			ref, err := core.ParseCanonicalID(v.Text)
			if err != nil {
				return contract, p.errAt(v, err.Error())
			}
			if kind == "read" {
				contract.Reads = append(contract.Reads, ref)
			} else {
				contract.Writes = append(contract.Writes, ref)
			}

		case "query":
			queryContract, err := p.parseContractQueryAuthoring()
			if err != nil {
				return contract, err
			}
			contract.Queries = append(contract.Queries, queryContract)

		default:
			return contract, p.errAt(t, "Contractではread、write、queryのいずれかを指定してください")
		}
	}

	if err := p.expectEOF(); err != nil {
		return contract, err
	}
	return contract, nil
}

func (p *Parser) parseContractQueryAuthoring() (database.QueryContract, error) {
	if _, err := p.expectText("query"); err != nil {
		return database.QueryContract{}, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return database.QueryContract{}, err
	}
	name, err := core.ParseLocalName(nameTok.Text)
	if err != nil {
		return database.QueryContract{}, p.errAt(nameTok, err.Error())
	}

	out := database.QueryContract{Name: string(name)}
	next, err := p.peek()
	if err != nil {
		return out, err
	}
	if next.Kind != TokenLBrace {
		return out, nil
	}

	open, _ := p.next()
	hasResult := false
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
		case "input":
			_, _ = p.next()
			inputTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			inputName, err := core.ParseLocalName(inputTok.Text)
			if err != nil {
				return out, p.errAt(inputTok, err.Error())
			}
			if _, err := p.expectText("type"); err != nil {
				return out, err
			}
			typeTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			semanticType, err := core.ParseCanonicalID(typeTok.Text)
			if err != nil {
				return out, p.errAt(typeTok, err.Error())
			}
			out.Inputs = append(out.Inputs, database.QueryInput{Name: inputName, Type: semanticType})

		case "result":
			if hasResult {
				return out, p.errAt(t, "Query signatureではresultを一度だけ定義できます")
			}
			hasResult = true
			_, _ = p.next()
			dataTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			result, err := core.ParseCanonicalID(dataTok.Text)
			if err != nil {
				return out, p.errAt(dataTok, err.Error())
			}
			cardinalityTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			cardinality := database.QueryResultCardinality(cardinalityTok.Text)
			if !cardinality.Valid() {
				return out, p.errAt(cardinalityTok, fmt.Sprintf("Query result cardinality %qは定義されていません", cardinalityTok.Text))
			}
			out.Result = result
			out.ResultCardinality = cardinality

		default:
			return out, p.errAt(t, "Query signatureではinputまたはresultを指定してください")
		}
	}

	if !hasResult {
		return out, p.errAt(open, "Query signatureを定義する場合はresultが必要です")
	}
	return out, nil
}
