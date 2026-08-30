package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/query"
)

// QueryAuthoring is a parser-side result. NativeSQLPath belongs to the
// Authoring Layer and is deliberately kept outside query.EmbeddedSQL.
type QueryAuthoring struct {
	Value         query.EmbeddedSQL
	NativeSQLPath string
}

// ParseQueryAuthoring parses both the existing embedded SQL form and the M0.23
// native SQL file form. A native file is not read here; Workspace loading owns
// physical path resolution and turns its content into Value.Source.
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
		default:
			return out, p.errAt(t, "expected datasource or sql")
		}
	}
	if err := p.expectEOF(); err != nil {
		return out, err
	}
	return out, nil
}

func nativeQueryRequiresWorkspace(out QueryAuthoring) error {
	if out.NativeSQLPath == "" {
		return nil
	}
	return fmt.Errorf("query %s uses native SQL file %q; resolve it through the Workspace loader", out.Value.ID, out.NativeSQLPath)
}
