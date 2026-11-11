package syntax

import (
	"fmt"
	"strconv"
	"strings"

	"tei/internal/core"
	"tei/internal/query"
)

// RelationQueryAuthoringは、Relation Query定義を読み取る際に必要なAuthoring Layerの情報を保持する。
type RelationQueryAuthoring struct {
	Value query.RelationQuery
}

// ParseRelationQueryは、relation-queryブロックを読み取る。
//
// 期待する構文:
//
//	relation-query <id> {
//	    datasource <alias>
//	    from <data_id>
//	    relation <relation_id>
//	    filter { <raw expression> }
//	    sort { <raw expression> }
//	    limit <number>
//	    offset <number>
//	    depth <number>
//	    mode <tree|graph|path>
//	    result { column <column> -> <attribute> ... }
//	}
func (p *Parser) ParseRelationQueryAuthoring(owner core.CanonicalID) (RelationQueryAuthoring, error) {
	if _, err := p.expectText("relation-query"); err != nil {
		return RelationQueryAuthoring{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return RelationQueryAuthoring{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return RelationQueryAuthoring{}, p.errAt(idTok, err.Error())
	}
	out := RelationQueryAuthoring{Value: query.RelationQuery{ID: id, Application: owner}}
	if _, err := p.expect(TokenLBrace); err != nil {
		return RelationQueryAuthoring{}, err
	}

	hasResult := false
	for {
		t, err := p.peek()
		if err != nil {
			return RelationQueryAuthoring{}, err
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
				return RelationQueryAuthoring{}, err
			}
			out.Value.Datasource = v.Text
		case "from":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			out.Value.Root, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return RelationQueryAuthoring{}, p.errAt(v, err.Error())
			}
		case "relation":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			relID, err := core.ParseCanonicalID(v.Text)
			if err != nil {
				return RelationQueryAuthoring{}, p.errAt(v, err.Error())
			}
			out.Value.Relations = append(out.Value.Relations, query.RelationStep{Relation: relID})
		case "filter":
			_, _ = p.next()
			if _, err := p.expect(TokenLBrace); err != nil {
				return RelationQueryAuthoring{}, err
			}
			body, _, err := p.lx.ReadRawBlock()
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			filter := query.Filter{Expression: strings.TrimSpace(body)}
			if strings.Contains(body, "AS OF") || strings.Contains(body, "BETWEEN") || strings.Contains(body, "validAt") {
				filter.Temporal = &query.TemporalFilter{}
				if idx := strings.Index(body, "AS OF"); idx >= 0 {
					filter.Temporal.ValidAt = strings.TrimSpace(body[idx+5:])
				}
				if idx := strings.Index(body, "BETWEEN"); idx >= 0 {
					parts := strings.Split(body[idx+7:], "AND")
					if len(parts) == 2 {
						filter.Temporal.From = strings.TrimSpace(parts[0])
						filter.Temporal.To = strings.TrimSpace(parts[1])
					}
				}
				filter.Temporal.ValidAt = strings.Trim(filter.Temporal.ValidAt, "'\"")
				filter.Temporal.From = strings.Trim(filter.Temporal.From, "'\"")
				filter.Temporal.To = strings.Trim(filter.Temporal.To, "'\"")
			}
			out.Value.Filters = append(out.Value.Filters, filter)
		case "sort":
			_, _ = p.next()
			body, _, err := p.lx.ReadRawBlock()
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			out.Value.Sorts = append(out.Value.Sorts, query.Sort{Expression: strings.TrimSpace(body)})
		case "limit":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			n, err := strconv.Atoi(v.Text)
			if err != nil {
				return RelationQueryAuthoring{}, p.errAt(v, "limit must be a number")
			}
			out.Value.PageSize = n
		case "offset":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			n, err := strconv.Atoi(v.Text)
			if err != nil {
				return RelationQueryAuthoring{}, p.errAt(v, "offset must be a number")
			}
			out.Value.PageOffset = n
		case "depth":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			n, err := strconv.Atoi(v.Text)
			if err != nil {
				return RelationQueryAuthoring{}, p.errAt(v, "depth must be a number")
			}
			out.Value.Depth = n
		case "mode":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			out.Value.Mode = query.RelationQueryMode(v.Text)
			if !out.Value.Mode.Valid() {
				return RelationQueryAuthoring{}, p.errAt(v, fmt.Sprintf("invalid mode %q; expected tree, graph, or path", v.Text))
			}
		case "fetch":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			out.Value.FetchMode = query.RelationFetchMode(v.Text)
			if !out.Value.FetchMode.Valid() {
				return RelationQueryAuthoring{}, p.errAt(v, fmt.Sprintf("invalid fetch %q; expected eager, lazy, or batch", v.Text))
			}
		case "batch":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			n, err := strconv.Atoi(v.Text)
			if err != nil {
				return RelationQueryAuthoring{}, p.errAt(v, "batch must be a number")
			}
			out.Value.BatchSize = n
		case "result":
			if hasResult {
				return RelationQueryAuthoring{}, p.errAt(t, "resultは一つのQueryで一度だけ定義できます")
			}
			hasResult = true
			_, _ = p.next()
			columns, err := p.parseQueryResultColumns()
			if err != nil {
				return RelationQueryAuthoring{}, err
			}
			out.Value.ResultColumns = columns
		default:
			return RelationQueryAuthoring{}, p.errAt(t, "expected datasource, from, relation, filter, sort, limit, offset, depth, mode, fetch, batch, or result")
		}
	}
	if out.Value.Root == "" {
		return RelationQueryAuthoring{}, fmt.Errorf("relation query %s requires a from clause", id)
	}
	if len(out.Value.Relations) == 0 {
		return RelationQueryAuthoring{}, fmt.Errorf("relation query %s requires at least one relation", id)
	}
	if !hasResult {
		return RelationQueryAuthoring{}, fmt.Errorf("relation query %s requires a result block", id)
	}
	return out, nil
}
