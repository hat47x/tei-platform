package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/realization"
)

// ContractQuerySQLRealizationAuthoringは、SQL realization定義を読み取る際にだけ必要な
// native SQL file pathをCanonical modelから分離して保持する。
type ContractQuerySQLRealizationAuthoring struct {
	Value         realization.ContractQuerySQLRealization
	NativeSQLPath string
}

// ParseContractQuerySQLRealizationAuthoringは、Database-owned Contract QueryのSQL realizationを読み込む。
// Database / Contract / Query、provider、Contract input→物理parameter、raw result列→Semantic attributeを
// すべて明示情報として扱い、名前やfile pathから不足した意味を補完しない。
func (p *Parser) ParseContractQuerySQLRealizationAuthoring() (ContractQuerySQLRealizationAuthoring, error) {
	var out ContractQuerySQLRealizationAuthoring
	if _, err := p.expectText("contract-query-sql-realization"); err != nil {
		return out, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return out, err
	}

	seenSingleton := map[string]bool{}
	seenInputs := map[core.LocalName]bool{}
	seenParameters := map[string]bool{}
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
		case "database", "contract", "query", "sql", "result":
			field, _ := p.next()
			if seenSingleton[field.Text] {
				return out, p.errAt(field, field.Text+" is declared more than once")
			}
			seenSingleton[field.Text] = true
			switch field.Text {
			case "database":
				v, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				out.Value.Target.Database, err = core.ParseCanonicalID(v.Text)
				if err != nil {
					return out, p.errAt(v, err.Error())
				}
			case "contract":
				v, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				out.Value.Target.Contract, err = core.ParseCanonicalID(v.Text)
				if err != nil {
					return out, p.errAt(v, err.Error())
				}
			case "query":
				v, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				out.Value.Target.Query, err = core.ParseLocalName(v.Text)
				if err != nil {
					return out, p.errAt(v, err.Error())
				}
			case "sql":
				providerTok, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				out.Value.Provider = environment.DatabaseProvider(providerTok.Text)
				form, err := p.peek()
				if err != nil {
					return out, err
				}
				switch {
				case form.Kind == TokenLBrace:
					_, _ = p.next()
					body, _, err := p.lx.ReadRawBlock()
					if err != nil {
						return out, err
					}
					p.peeked = nil
					out.Value.Source = strings.TrimSpace(body)
				case form.Kind == TokenIdent && form.Text == "file":
					_, _ = p.next()
					pathTok, err := p.expect(TokenString)
					if err != nil {
						return out, err
					}
					if strings.TrimSpace(pathTok.Text) == "" {
						return out, p.errAt(pathTok, "SQL file path must not be empty")
					}
					out.NativeSQLPath = pathTok.Text
				default:
					return out, p.errAt(form, "expected SQL block or file")
				}
			case "result":
				columns, err := p.parseQueryResultColumns()
				if err != nil {
					return out, err
				}
				out.Value.ResultColumns = columns
			}
		case "input":
			_, _ = p.next()
			inputTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			input, err := core.ParseLocalName(inputTok.Text)
			if err != nil {
				return out, p.errAt(inputTok, err.Error())
			}
			if seenInputs[input] {
				return out, p.errAt(inputTok, fmt.Sprintf("input %sのparameter mappingが重複しています", input))
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return out, err
			}
			parameterTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			if seenParameters[parameterTok.Text] {
				return out, p.errAt(parameterTok, fmt.Sprintf("physical parameter %qが重複しています", parameterTok.Text))
			}
			seenInputs[input] = true
			seenParameters[parameterTok.Text] = true
			out.Value.Inputs = append(out.Value.Inputs, realization.ContractQueryInputParameterMapping{Input: input, Parameter: parameterTok.Text})
		default:
			return out, p.errAt(t, "expected database, contract, query, sql, input, or result")
		}
	}

	if !seenSingleton["database"] || !seenSingleton["contract"] || !seenSingleton["query"] {
		return out, fmt.Errorf("contract-query-sql-realization requires database, contract, and query")
	}
	if !seenSingleton["sql"] {
		return out, fmt.Errorf("contract-query-sql-realization requires sql")
	}
	if !seenSingleton["result"] {
		return out, fmt.Errorf("contract-query-sql-realization requires result mapping")
	}
	if err := p.expectEOF(); err != nil {
		return out, err
	}
	return out, nil
}
