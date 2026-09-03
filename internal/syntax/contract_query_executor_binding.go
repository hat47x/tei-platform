package syntax

import (
	"tei/internal/core"
	"tei/internal/realization"
)

// ParseContractQueryExecutorBindingは、Database Contract Query executorの明示Bindingを読み込む。
//
// Environment、Database instance、Database / Contract / Query target、executor Capabilityを
// source内ですべて明示する。ファイル名や配置、Database providerから不足したidentityを補完しない。
func (p *Parser) ParseContractQueryExecutorBinding() (realization.ContractQueryExecutorBinding, error) {
	var out realization.ContractQueryExecutorBinding
	if _, err := p.expectText("contract-query-executor-binding"); err != nil {
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
		if seen[field.Text] {
			return out, p.errAt(field, field.Text+" is declared more than once")
		}
		seen[field.Text] = true

		switch field.Text {
		case "environment":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Environment, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "instance":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Instance, err = core.ParseCanonicalID(v.Text)
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
		case "executor":
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Executor, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		default:
			return out, p.errAt(field, "expected environment, instance, database, contract, query, or executor")
		}
	}

	if !seen["environment"] {
		return out, p.errAt(closeToken, "contract-query-executor-binding requires environment")
	}
	if !seen["instance"] {
		return out, p.errAt(closeToken, "contract-query-executor-binding requires instance")
	}
	if !seen["database"] || !seen["contract"] || !seen["query"] {
		return out, p.errAt(closeToken, "contract-query-executor-binding requires database, contract, and query")
	}
	if !seen["executor"] {
		return out, p.errAt(closeToken, "contract-query-executor-binding requires executor")
	}
	if err := p.expectEOF(); err != nil {
		return out, err
	}
	return out, nil
}
