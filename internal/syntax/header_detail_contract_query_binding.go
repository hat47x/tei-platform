package syntax

import (
	"tei/internal/core"
	"tei/internal/realization"
)

// ParseHeaderDetailContractQueryBindingは、Relation-onlyのheader-detail Patternを
// collection取得用Database Contract Queryへ接続する独立realization sourceを読み込む。
//
// Application / Pattern / logical Datasource / Database Contract Query targetを本文で明示し、
// ファイル名や配置からidentityを補完しない。inputはRelation source DataのSemantic attributeから
// Query inputへの対応だけを保持し、Relation target keyや物理Foreign Keyを要求しない。
func (p *Parser) ParseHeaderDetailContractQueryBinding() (realization.HeaderDetailContractQueryBinding, error) {
	var out realization.HeaderDetailContractQueryBinding
	if _, err := p.expectText("header-detail-contract-query-binding"); err != nil {
		return out, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return out, err
	}

	seen := map[string]bool{}
	var closeToken Token
	for {
		t, err := p.peek()
		if err != nil {
			return out, err
		}
		if t.Kind == TokenRBrace {
			closeToken, _ = p.next()
			break
		}

		field, _ := p.next()
		if field.Text != "input" {
			if seen[field.Text] {
				return out, p.errAt(field, field.Text+" is declared more than once")
			}
			seen[field.Text] = true
		}

		switch field.Text {
		case "application":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Application, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "pattern":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Pattern, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "datasource":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Datasource, err = core.ParseLocalName(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "database":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Target.Database, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "contract":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Target.Contract, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "query":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Target.Query, err = core.ParseLocalName(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "input":
			source, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			sourceAttribute, err := core.ParseCanonicalID(source.Text)
			if err != nil {
				return out, p.errAt(source, err.Error())
			}
			if _, err := p.expectText("to"); err != nil {
				return out, err
			}
			input, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			inputName, err := core.ParseLocalName(input.Text)
			if err != nil {
				return out, p.errAt(input, err.Error())
			}
			out.Inputs = append(out.Inputs, realization.HeaderDetailContractQueryInputBinding{
				SourceAttribute: sourceAttribute,
				Input:           inputName,
			})
		default:
			return out, p.errAt(field, "expected application, pattern, datasource, database, contract, query, or input")
		}
	}

	for _, required := range []string{"application", "pattern", "datasource", "database", "contract", "query"} {
		if !seen[required] {
			return out, p.errAt(closeToken, "header-detail-contract-query-binding requires "+required)
		}
	}
	if err := p.expectEOF(); err != nil {
		return out, err
	}
	return out, nil
}
