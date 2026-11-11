package syntax

import (
	"tei/internal/application"
	"tei/internal/core"
)

// parseReferenceContractQueryは、reference Patternが利用するDatabase Contract Queryと、
// Relation target側のSemantic attributeからQuery inputへの明示的な対応を読み取る。
// Contract Queryのsignatureや実現方式はここでは解釈しない。
func (p *Parser) parseReferenceContractQuery() (*application.ContractQueryInvocation, error) {
	if _, err := p.expectText("contract-query"); err != nil {
		return nil, err
	}
	open, err := p.expect(TokenLBrace)
	if err != nil {
		return nil, err
	}

	out := application.ContractQueryInvocation{}
	hasContract := false
	hasQuery := false

	for {
		t, err := p.peek()
		if err != nil {
			return nil, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}

		switch t.Text {
		case "contract":
			if hasContract {
				return nil, p.errAt(t, "contract-query can define contract once")
			}
			hasContract = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			out.Contract, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return nil, p.errAt(v, err.Error())
			}

		case "query":
			if hasQuery {
				return nil, p.errAt(t, "contract-query can define query once")
			}
			hasQuery = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			out.Query, err = core.ParseLocalName(v.Text)
			if err != nil {
				return nil, p.errAt(v, err.Error())
			}

		case "input":
			_, _ = p.next()
			targetTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			target, err := core.ParseCanonicalID(targetTok.Text)
			if err != nil {
				return nil, p.errAt(targetTok, err.Error())
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return nil, err
			}
			inputTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			input, err := core.ParseLocalName(inputTok.Text)
			if err != nil {
				return nil, p.errAt(inputTok, err.Error())
			}
			out.Inputs = append(out.Inputs, application.ContractQueryInputBinding{
				TargetAttribute: target,
				Input:           input,
			})

		default:
			return nil, p.errAt(t, "contract-query needs one of contract, query or input")
		}
	}

	if !hasContract {
		return nil, p.errAt(open, "contract-query requires contract")
	}
	if !hasQuery {
		return nil, p.errAt(open, "contract-query requires query")
	}
	return &out, nil
}
