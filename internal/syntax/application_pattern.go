package syntax

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// ParseApplicationPatternは、Application配下の独立したPattern定義を読み取る。
// Pattern Kindごとの必須条件は、実Referenceで確認した意味に限ってここで構文契約として固定する。
func (p *Parser) ParseApplicationPattern() (application.Pattern, error) {
	if _, err := p.expectText("pattern"); err != nil {
		return application.Pattern{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return application.Pattern{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return application.Pattern{}, p.errAt(idTok, err.Error())
	}
	kindTok, err := p.expect(TokenIdent)
	if err != nil {
		return application.Pattern{}, err
	}
	out := application.Pattern{ID: id, Kind: application.PatternKind(kindTok.Text)}
	if !out.Kind.Valid() {
		return out, p.errAt(kindTok, fmt.Sprintf("unsupported application pattern kind %q", kindTok.Text))
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return out, err
	}

	var hasData, hasRelation, hasQuery, hasDatasource bool
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
		case "data":
			if hasData {
				return out, p.errAt(t, "data is declared more than once")
			}
			hasData = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Data, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "relation":
			if hasRelation {
				return out, p.errAt(t, "relation is declared more than once")
			}
			hasRelation = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Relation, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "query":
			if hasQuery {
				return out, p.errAt(t, "query is declared more than once")
			}
			hasQuery = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Query, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "datasource":
			if hasDatasource {
				return out, p.errAt(t, "datasource is declared more than once")
			}
			hasDatasource = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Datasource, err = core.ParseLocalName(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		default:
			return out, p.errAt(t, "expected data, relation, query, or datasource")
		}
	}

	switch out.Kind {
	case application.PatternBrowse:
		if out.Data == "" {
			return out, fmt.Errorf("browse pattern %s requires data", out.ID)
		}
		if out.Query == "" {
			return out, fmt.Errorf("browse pattern %s requires query", out.ID)
		}
		if out.Relation != "" {
			return out, fmt.Errorf("browse pattern %s does not accept relation", out.ID)
		}
	case application.PatternRecord:
		if out.Data == "" {
			return out, fmt.Errorf("record pattern %s requires data", out.ID)
		}
		if out.Relation != "" {
			return out, fmt.Errorf("record pattern %s does not accept relation", out.ID)
		}
		if out.Query != "" {
			return out, fmt.Errorf("record pattern %s does not accept query", out.ID)
		}
		if out.Datasource != "" {
			return out, fmt.Errorf("record pattern %s does not accept datasource", out.ID)
		}
	case application.PatternReference:
		if out.Relation == "" {
			return out, fmt.Errorf("reference pattern %s requires relation", out.ID)
		}
		if out.Datasource == "" {
			return out, fmt.Errorf("reference pattern %s requires datasource", out.ID)
		}
		if out.Data != "" {
			return out, fmt.Errorf("reference pattern %s does not accept data", out.ID)
		}
		if out.Query != "" {
			return out, fmt.Errorf("reference pattern %s does not accept query", out.ID)
		}
	}
	if err := p.expectEOF(); err != nil {
		return out, err
	}
	return out, nil
}
