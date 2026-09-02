package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/query"
)

// QueryAuthoringは、Query定義を読み取る際にだけ必要なAuthoring Layerの情報を保持する。
// NativeSQLPathは物理的なsource pathなのでCanonical Queryへは含めない。
type QueryAuthoring struct {
	Value         query.EmbeddedSQL
	NativeSQLPath string
}

// ParseQueryAuthoringは、埋め込みSQLとnative SQL fileの両方を読み取る。
// native SQL file自体の読み込みとpath解決はWorkspace Loaderが担い、ここではauthorが明示した
// Queryの意味とresult mappingだけを構文として扱う。
func (p *Parser) ParseQueryAuthoring(owner core.CanonicalID) (QueryAuthoring, error) {
	if _, err := p.expectText("query"); err != nil {
		return QueryAuthoring{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return QueryAuthoring{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return QueryAuthoring{}, p.errAt(idTok, err.Error())
	}
	out := QueryAuthoring{Value: query.EmbeddedSQL{ID: id, Application: owner}}
	if _, err := p.expect(TokenLBrace); err != nil {
		return out, err
	}

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
		case "datasource":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Value.Datasource, err = core.ParseLocalName(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "sql":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Value.Dialect = environment.DatabaseProvider(v.Text)

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
			if hasResult {
				return out, p.errAt(t, "resultは一つのQueryで一度だけ定義できます")
			}
			hasResult = true
			_, _ = p.next()
			columns, err := p.parseQueryResultColumns()
			if err != nil {
				return out, err
			}
			out.Value.ResultColumns = columns
		default:
			return out, p.errAt(t, "expected datasource, sql, or result")
		}
	}
	if err := p.expectEOF(); err != nil {
		return out, err
	}
	return out, nil
}

func (p *Parser) parseQueryResultColumns() ([]query.ResultColumnMapping, error) {
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	seenColumns := map[string]struct{}{}
	seenAttributes := map[core.CanonicalID]struct{}{}
	var out []query.ResultColumnMapping

	for {
		t, err := p.peek()
		if err != nil {
			return nil, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		if _, err := p.expectText("column"); err != nil {
			return nil, err
		}
		columnTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, exists := seenColumns[columnTok.Text]; exists {
			return nil, p.errAt(columnTok, fmt.Sprintf("result column %qが重複しています", columnTok.Text))
		}
		if _, err := p.expect(TokenArrow); err != nil {
			return nil, err
		}
		attributeTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		attribute, err := core.ParseCanonicalID(attributeTok.Text)
		if err != nil {
			return nil, p.errAt(attributeTok, err.Error())
		}
		if _, exists := seenAttributes[attribute]; exists {
			return nil, p.errAt(attributeTok, fmt.Sprintf("result attribute %sが重複しています", attribute))
		}
		seenColumns[columnTok.Text] = struct{}{}
		seenAttributes[attribute] = struct{}{}
		out = append(out, query.ResultColumnMapping{Column: columnTok.Text, Attribute: attribute})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("query resultには少なくとも一つのcolumn mappingが必要です")
	}
	return out, nil
}

func nativeQueryRequiresWorkspace(out QueryAuthoring) error {
	if out.NativeSQLPath == "" {
		return nil
	}
	return fmt.Errorf("query %s uses native SQL file %q; resolve it through the Workspace loader", out.Value.ID, out.NativeSQLPath)
}
