package standard

import (
	"fmt"
	"strings"
)

// ---------- tokenizer ----------

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokNumber
	tokString
	tokNull
	tokTrue
	tokFalse
	tokParam
	tokStar
	tokComma
	tokDot
	tokLParen
	tokRParen
	tokEq
	tokNeq
	tokLt
	tokLtE
	tokGt
	tokGtE
	tokPlus
	tokMinus
	tokSlash
	tokQuestion
	tokColon
	tokOther
)

type token struct {
	kind tokenKind
	text string
	pos  int
}

type parser struct {
	src    string
	tokens []token
	pos    int
	diags  []Diagnostic
	nextID int
}

func ParseSQLToStandard(src string, dialect string) (*StandardQuery, []Diagnostic) {
	if strings.TrimSpace(src) == "" {
		return nil, []Diagnostic{{Code: "EMPTY_SQL", Message: "empty SQL source", Severity: SeverityError}}
	}
	p := &parser{src: src}
	p.tokenize()
	q, err := p.parseQuery()
	if err != nil {
		p.diags = append(p.diags, Diagnostic{Code: "PARSE_ERROR", Message: err.Error(), Severity: SeverityError})
	}
	if q == nil {
		q = &StandardQuery{Dialect: dialect}
	}
	if len(p.diags) > 0 {
		q.Diagnostics = append(q.Diagnostics, p.diags...)
	}
	return q, q.Diagnostics
}

func (p *parser) tokenize() {
	src := p.src
	i := 0
	for i < len(src) {
		if isSpace(src[i]) {
			i++
			continue
		}
		if i+1 < len(src) && src[i] == '-' && src[i+1] == '-' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		start := i
		ch := src[i]
		switch {
		case isLetter(ch) || ch == '_':
			for i < len(src) && (isLetter(src[i]) || isDigit(src[i]) || src[i] == '_') {
				i++
			}
			word := src[start:i]
			lower := strings.ToLower(word)
			switch lower {
			case "null":
				p.tokens = append(p.tokens, token{tokNull, word, start})
			case "true":
				p.tokens = append(p.tokens, token{tokTrue, word, start})
			case "false":
				p.tokens = append(p.tokens, token{tokFalse, word, start})
			default:
				p.tokens = append(p.tokens, token{tokIdent, word, start})
			}
		case isDigit(ch) || (ch == '.' && i+1 < len(src) && isDigit(src[i+1])):
			for i < len(src) && (isDigit(src[i]) || src[i] == '.') {
				i++
			}
			p.tokens = append(p.tokens, token{tokNumber, src[start:i], start})
		case ch == '\'':
			i++
			var sb strings.Builder
			for i < len(src) {
				if src[i] == '\'' {
					if i+1 < len(src) && src[i+1] == '\'' {
						sb.WriteByte('\'')
						i += 2
						continue
					}
					i++
					break
				}
				sb.WriteByte(src[i])
				i++
			}
			p.tokens = append(p.tokens, token{tokString, sb.String(), start})
		case ch == ':':
			i++
			for i < len(src) && (isLetter(src[i]) || isDigit(src[i]) || src[i] == '_') {
				i++
			}
			p.tokens = append(p.tokens, token{tokParam, src[start:i], start})
		case ch == '?':
			p.tokens = append(p.tokens, token{tokQuestion, "?", start})
			i++
		case ch == '*':
			p.tokens = append(p.tokens, token{tokStar, "*", start})
			i++
		case ch == ',':
			p.tokens = append(p.tokens, token{tokComma, ",", start})
			i++
		case ch == '.':
			p.tokens = append(p.tokens, token{tokDot, ".", start})
			i++
		case ch == '(':
			p.tokens = append(p.tokens, token{tokLParen, "(", start})
			i++
		case ch == ')':
			p.tokens = append(p.tokens, token{tokRParen, ")", start})
			i++
		case ch == '=':
			p.tokens = append(p.tokens, token{tokEq, "=", start})
			i++
		case ch == '<':
			if i+1 < len(src) {
				switch src[i+1] {
				case '=':
					p.tokens = append(p.tokens, token{tokLtE, "<=", start})
					i += 2
					continue
				case '>':
					p.tokens = append(p.tokens, token{tokNeq, "<>", start})
					i += 2
					continue
				}
			}
			p.tokens = append(p.tokens, token{tokLt, "<", start})
			i++
		case ch == '>':
			if i+1 < len(src) && src[i+1] == '=' {
				p.tokens = append(p.tokens, token{tokGtE, ">=", start})
				i += 2
				continue
			}
			p.tokens = append(p.tokens, token{tokGt, ">", start})
			i++
		case ch == '!':
			if i+1 < len(src) && src[i+1] == '=' {
				p.tokens = append(p.tokens, token{tokNeq, "!=", start})
				i += 2
				continue
			}
			p.tokens = append(p.tokens, token{tokOther, "!", start})
			i++
		case ch == '+':
			p.tokens = append(p.tokens, token{tokPlus, "+", start})
			i++
		case ch == '-':
			p.tokens = append(p.tokens, token{tokMinus, "-", start})
			i++
		case ch == '/':
			p.tokens = append(p.tokens, token{tokSlash, "/", start})
			i++
		default:
			p.tokens = append(p.tokens, token{tokOther, string(ch), start})
			i++
		}
	}
	p.tokens = append(p.tokens, token{tokEOF, "", i})
}

func isSpace(b byte) bool  { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }
func isLetter(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }
func isDigit(b byte) bool  { return b >= '0' && b <= '9' }

func (p *parser) peek() token { return p.tokens[p.pos] }
func (p *parser) next() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}
func (p *parser) backup() { p.pos-- }
func (p *parser) expect(k tokenKind) (token, error) {
	if p.pos >= len(p.tokens) {
		return token{}, fmt.Errorf("unexpected end of input")
	}
	t := p.next()
	if t.kind != k {
		return t, fmt.Errorf("expected token %d at %d, got %s", k, t.pos, t.text)
	}
	return t, nil
}
func (p *parser) match(k tokenKind) bool {
	if p.pos < len(p.tokens) && p.tokens[p.pos].kind == k {
		p.pos++
		return true
	}
	return false
}
func (p *parser) matchKeyword(words ...string) bool {
	if p.pos >= len(p.tokens) {
		return false
	}
	word := strings.ToLower(p.tokens[p.pos].text)
	for _, w := range words {
		if word == w {
			p.pos++
			return true
		}
	}
	return false
}

func (p *parser) peekKeyword(words ...string) bool {
	if p.pos >= len(p.tokens) {
		return false
	}
	word := strings.ToLower(p.tokens[p.pos].text)
	for _, w := range words {
		if word == w {
			return true
		}
	}
	return false
}
func (p *parser) matchToken(k tokenKind, text string) bool {
	if p.pos < len(p.tokens) && p.tokens[p.pos].kind == k && strings.EqualFold(p.tokens[p.pos].text, text) {
		p.pos++
		return true
	}
	return false
}
func (p *parser) error(msg string) error { return fmt.Errorf("%s at %d", msg, p.peek().pos) }
func (p *parser) addDiag(code, msg string) {
	p.diags = append(p.diags, Diagnostic{Code: code, Message: msg})
}
func (p *parser) nextNodeID() string {
	id := fmt.Sprintf("node:%d", p.nextID)
	p.nextID++
	return id
}
func (p *parser) nextSubqueryID() string {
	id := fmt.Sprintf("subq:%d", p.nextID)
	p.nextID++
	return id
}

// ---------- parse helpers ----------

type queryBuilder struct {
	nodes []Node
	cur   string
}

func (b *queryBuilder) add(n Node) string {
	id := ""
	switch v := n.(type) {
	case *Scan:
		id = v.ID
	case *Join:
		id = v.NodeID
	case *Filter:
		id = v.NodeID
	case *Aggregate:
		id = v.NodeID
	case *Project:
		id = v.NodeID
	case *Sort:
		id = v.NodeID
	case *Limit:
		id = v.NodeID
	case *Distinct:
		id = v.NodeID
	case *SubqueryScan:
		id = v.NodeID
	case *Subquery:
		id = v.ID
	}
	b.nodes = append(b.nodes, n)
	b.cur = id
	return id
}

func nodeID(n Node) string {
	switch v := n.(type) {
	case *Scan:
		return v.ID
	case *Join:
		return v.NodeID
	case *Filter:
		return v.NodeID
	case *Aggregate:
		return v.NodeID
	case *Project:
		return v.NodeID
	case *Sort:
		return v.NodeID
	case *Limit:
		return v.NodeID
	case *Distinct:
		return v.NodeID
	case *SubqueryScan:
		return v.NodeID
	case *Subquery:
		return v.ID
	}
	return ""
}

func (p *parser) parseQuery() (*StandardQuery, error) {
	b := &queryBuilder{}
	var ctes []CTE
	if p.matchKeyword("with") {
		for {
			cte, err := p.parseCTE()
			if err != nil {
				return nil, err
			}
			ctes = append(ctes, *cte)
			if !p.match(tokComma) {
				break
			}
		}
	}
	root, err := p.parseQueryBody(b)
	if err != nil {
		return nil, err
	}
	q := &StandardQuery{Root: nodeID(root), Nodes: b.nodes, CTEs: ctes}
	if len(p.diags) > 0 {
		q.Diagnostics = append(q.Diagnostics, p.diags...)
	}
	return q, nil
}

func (p *parser) parseCTE() (*CTE, error) {
	nameTok, err := p.expect(tokIdent)
	if err != nil {
		return nil, err
	}
	if !p.matchKeyword("as") {
		return nil, p.error("expected AS after CTE name")
	}
	_, err = p.expect(tokLParen)
	if err != nil {
		return nil, err
	}
	sub, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	_, err = p.expect(tokRParen)
	if err != nil {
		return nil, err
	}
	return &CTE{ID: strings.ToLower(nameTok.text), Query: sub}, nil
}

func (p *parser) parseQueryBody(b *queryBuilder) (Node, error) {
	if !p.matchKeyword("select") {
		return nil, p.error("expected SELECT")
	}
	distinct := p.matchKeyword("distinct")
	items, aliases, err := p.parseSelectList()
	if err != nil {
		return nil, err
	}
	_, err = p.parseFrom(b)
	if err != nil {
		return nil, err
	}
	if distinct {
		b.add(&Distinct{NodeID: p.nextNodeID()})
	}
	proj := &Project{NodeID: p.nextNodeID(), Items: make([]ProjectItem, len(items))}
	for i, expr := range items {
		proj.Items[i] = ProjectItem{Expr: expr, Alias: aliases[i]}
	}
	b.add(proj)
	whereNode, err := p.parseWhere(b)
	if err != nil {
		return nil, err
	}
	if whereNode != nil {
		b.add(whereNode)
	}
	groupNode, err := p.parseGroupBy()
	if err != nil {
		return nil, err
	}
	if groupNode != nil {
		b.add(&Aggregate{NodeID: p.nextNodeID(), Grouping: groupNode})
	}
	havingNode, err := p.parseHaving()
	if err != nil {
		return nil, err
	}
	if havingNode != nil {
		b.add(havingNode)
	}
	sortNode, err := p.parseOrderBy()
	if err != nil {
		return nil, err
	}
	if sortNode != nil {
		b.add(sortNode)
	}
	limitNode, err := p.parseLimit()
	if err != nil {
		return nil, err
	}
	if limitNode != nil {
		b.add(limitNode)
	}
	return b.nodes[len(b.nodes)-1], nil
}

func (p *parser) parseFrom(b *queryBuilder) (*Scan, error) {
	if !p.matchKeyword("from") {
		return nil, p.error("expected FROM")
	}
	left, leftAlias, err := p.parseTablePrimary(b)
	if err != nil {
		return nil, err
	}
	if leftAlias != "" {
		left.Alias = leftAlias
	}
	b.add(left)
	for {
		join, jerr := p.parseJoinClause(b)
		if jerr != nil {
			break
		}
		if join == nil {
			break
		}
		b.add(join)
	}
	return left, nil
}

func (p *parser) parseTablePrimary(b *queryBuilder) (*Scan, string, error) {
	tok := p.peek()
	if tok.kind == tokLParen {
		p.next()
		sub, err := p.parseQuery()
		if err != nil {
			return nil, "", err
		}
		_, err = p.expect(tokRParen)
		if err != nil {
			return nil, "", err
		}
		var alias string
		if p.matchKeyword("as") {
			at, err := p.expect(tokIdent)
			if err != nil {
				return nil, "", err
			}
			alias = at.text
		} else if p.peek().kind == tokIdent && !isReservedWord(p.peek().text) {
			at := p.next()
			alias = at.text
		}
		id := p.nextSubqueryID()
		b.add(&Subquery{ID: id, Query: sub, Alias: alias})
		return &Scan{ID: p.nextNodeID(), SourceKind: "subquery", SubqueryID: id, Alias: alias}, alias, nil
	}
	if tok.kind == tokIdent {
		p.next()
		name := tok.text
		qualifier := ""
		alias := ""
		if p.match(tokDot) {
			if p.match(tokStar) {
				return &Scan{Source: name + ".*", SourceKind: "table"}, "", nil
			}
			col, err := p.expect(tokIdent)
			if err != nil {
				return nil, "", err
			}
			qualifier = name
			name = col.text
		}
		if p.matchKeyword("as") {
			at, err := p.expect(tokIdent)
			if err != nil {
				return nil, "", err
			}
			alias = at.text
		} else if p.peek().kind == tokIdent && !isReservedWord(p.peek().text) {
			at := p.next()
			alias = at.text
		}
		src := name
		if qualifier != "" {
			src = qualifier + "." + name
		}
		return &Scan{ID: p.nextNodeID(), Source: src, SourceKind: "table"}, alias, nil
	}
	return nil, "", p.error("expected table or subquery")
}

func (p *parser) parseJoinClause(b *queryBuilder) (*Join, error) {
	kind := "inner"
	if p.matchKeyword("left") {
		kind = "left"
		p.matchKeyword("outer")
	} else if p.matchKeyword("right") {
		kind = "right"
		p.matchKeyword("outer")
	} else if p.matchKeyword("full") {
		kind = "full"
		p.matchKeyword("outer")
	} else if p.matchKeyword("inner") {
		kind = "inner"
	} else if p.matchKeyword("cross") {
		kind = "cross"
	}
	if !p.matchKeyword("join") {
		return nil, nil
	}
	right, rightAlias, err := p.parseTablePrimary(b)
	if err != nil {
		return nil, err
	}
	if rightAlias != "" {
		right.Alias = rightAlias
	}
	leftID := b.cur
	b.add(right)
	join := &Join{
		NodeID: p.nextNodeID(),
		Kind:   kind,
		Left:   leftID,
		Right:  right.ID,
	}
	if p.matchKeyword("on") {
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		join.Condition = cond
	} else if p.matchKeyword("using") {
		_, err := p.expect(tokLParen)
		if err != nil {
			return nil, err
		}
		var cols []string
		for {
			colTok, err := p.expect(tokIdent)
			if err != nil {
				return nil, err
			}
			cols = append(cols, colTok.text)
			if !p.match(tokComma) {
				break
			}
		}
		_, err = p.expect(tokRParen)
		if err != nil {
			return nil, err
		}
		join.Using = cols
	} else if kind == "cross" {
		// ok
	} else {
		return nil, p.error("expected ON or USING after JOIN")
	}
	return join, nil
}

func (p *parser) parseSelectList() ([]Expr, []string, error) {
	var exprs []Expr
	var aliases []string
	for {
		expr, alias, err := p.parseSelectItem()
		if err != nil {
			return nil, nil, err
		}
		exprs = append(exprs, expr)
		aliases = append(aliases, alias)
		if !p.match(tokComma) {
			break
		}
	}
	return exprs, aliases, nil
}

var reserved = map[string]bool{
	"from": true, "where": true, "group": true, "having": true,
	"order": true, "limit": true, "offset": true, "join": true,
	"on": true, "using": true, "inner": true, "left": true,
	"right": true, "full": true, "cross": true, "as": true,
	"select": true, "distinct": true, "case": true, "when": true,
	"then": true, "else": true, "end": true, "and": true, "or": true,
	"not": true, "null": true, "true": true, "false": true,
	"in": true, "like": true, "between": true, "exists": true,
	"is": true, "asc": true, "desc": true,
}

func isReservedWord(s string) bool {
	return reserved[strings.ToLower(s)]
}

func (p *parser) parseSelectItem() (Expr, string, error) {
	if p.match(tokStar) {
		return &Literal{Kind: "star", Value: "*"}, "", nil
	}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, "", err
	}
	var alias string
	if p.matchKeyword("as") {
		tok, err := p.expect(tokIdent)
		if err != nil {
			return nil, "", err
		}
		alias = tok.text
	} else if p.peek().kind == tokIdent && !isReservedWord(p.peek().text) {
		tok := p.next()
		alias = tok.text
	}
	return expr, alias, nil
}

// ---------- expressions ----------

func (p *parser) parseExpr() (Expr, error) {
	return p.parseOr()
}

func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.matchKeyword("or") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryOp{Op: "or", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.matchKeyword("and") {
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &BinaryOp{Op: "and", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseNot() (Expr, error) {
	if p.matchKeyword("not") {
		operand, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &UnaryOp{Op: "not", Operand: operand}, nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (Expr, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	if p.matchKeyword("is") {
		if p.matchKeyword("not") {
			if !p.matchKeyword("null") {
				return nil, p.error("expected NULL")
			}
			return &IsNull{Negated: true, Expr: left}, nil
		}
		if !p.matchKeyword("null") {
			return nil, p.error("expected NULL")
		}
		return &IsNull{Expr: left}, nil
	}
	if p.matchKeyword("between") {
		low, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		if !p.matchKeyword("and") {
			return nil, p.error("expected AND")
		}
		high, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		return &Between{Expr: left, Low: low, High: high}, nil
	}
	if p.matchKeyword("in") {
		if p.match(tokLParen) {
			if p.peekKeyword("select") {
				sub, err := p.parseQuery()
				if err != nil {
					return nil, err
				}
				_, err = p.expect(tokRParen)
				if err != nil {
					return nil, err
				}
				return &InList{Expr: left, Subquery: sub}, nil
			}
			var list []Expr
			for {
				item, err := p.parseAdditive()
				if err != nil {
					return nil, err
				}
				list = append(list, item)
				if !p.match(tokComma) {
					break
				}
			}
			_, err = p.expect(tokRParen)
			if err != nil {
				return nil, err
			}
			return &InList{Expr: left, List: list}, nil
		}
		return nil, p.error("expected '(' after IN")
	}
	if p.matchKeyword("like") {
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		return &BinaryOp{Op: "like", Left: left, Right: right}, nil
	}
	if p.matchKeyword("not") {
		if p.matchKeyword("in") {
			if p.match(tokLParen) {
				if p.peekKeyword("select") {
					sub, err := p.parseQuery()
					if err != nil {
						return nil, err
					}
					_, err = p.expect(tokRParen)
					if err != nil {
						return nil, err
					}
					return &InList{Negated: true, Expr: left, Subquery: sub}, nil
				}
				var list []Expr
				for {
					item, err := p.parseAdditive()
					if err != nil {
						return nil, err
					}
					list = append(list, item)
					if !p.match(tokComma) {
						break
					}
				}
				_, err = p.expect(tokRParen)
				if err != nil {
					return nil, err
				}
				return &InList{Negated: true, Expr: left, List: list}, nil
			}
			return nil, p.error("expected '(' after IN")
		}
		if p.matchKeyword("like") {
			right, err := p.parseAdditive()
			if err != nil {
				return nil, err
			}
			return &BinaryOp{Op: "not like", Left: left, Right: right}, nil
		}
		p.backup()
	}
	if p.match(tokEq) {
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		return &BinaryOp{Op: "=", Left: left, Right: right}, nil
	}
	if p.match(tokNeq) {
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		return &BinaryOp{Op: "!=", Left: left, Right: right}, nil
	}
	if p.match(tokLtE) {
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		return &BinaryOp{Op: "<=", Left: left, Right: right}, nil
	}
	if p.match(tokGtE) {
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		return &BinaryOp{Op: ">=", Left: left, Right: right}, nil
	}
	if p.match(tokLt) {
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		return &BinaryOp{Op: "<", Left: left, Right: right}, nil
	}
	if p.match(tokGt) {
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		return &BinaryOp{Op: ">", Left: left, Right: right}, nil
	}
	return left, nil
}

func (p *parser) parseAdditive() (Expr, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for {
		if p.match(tokPlus) {
			right, err := p.parseMultiplicative()
			if err != nil {
				return nil, err
			}
			left = &BinaryOp{Op: "+", Left: left, Right: right}
		} else if p.match(tokMinus) {
			right, err := p.parseMultiplicative()
			if err != nil {
				return nil, err
			}
			left = &BinaryOp{Op: "-", Left: left, Right: right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *parser) parseMultiplicative() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		if p.match(tokStar) {
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = &BinaryOp{Op: "*", Left: left, Right: right}
		} else if p.match(tokSlash) {
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = &BinaryOp{Op: "/", Left: left, Right: right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *parser) parseUnary() (Expr, error) {
	if p.match(tokPlus) {
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryOp{Op: "+", Operand: operand}, nil
	}
	if p.match(tokMinus) {
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryOp{Op: "-", Operand: operand}, nil
	}
	if p.matchKeyword("not") {
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryOp{Op: "not", Operand: operand}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Expr, error) {
	tok := p.peek()
	if tok.kind == tokLParen {
		p.next()
		if p.matchKeyword("select") {
			sub, err := p.parseQuery()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(tokRParen)
			if err != nil {
				return nil, err
			}
			return &SubqueryExpr{Query: sub}, nil
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(tokRParen)
		if err != nil {
			return nil, err
		}
		return expr, nil
	}
	if tok.kind == tokNull {
		p.next()
		return &Literal{Kind: "null", Value: "null"}, nil
	}
	if tok.kind == tokTrue {
		p.next()
		return &Literal{Kind: "bool", Value: "true"}, nil
	}
	if tok.kind == tokFalse {
		p.next()
		return &Literal{Kind: "bool", Value: "false"}, nil
	}
	if tok.kind == tokNumber {
		p.next()
		return &Literal{Kind: "number", Value: tok.text}, nil
	}
	if tok.kind == tokString {
		p.next()
		return &Literal{Kind: "string", Value: tok.text}, nil
	}
	if tok.kind == tokParam || tok.kind == tokQuestion {
		p.next()
		name := tok.text
		if tok.kind == tokQuestion {
			name = "?"
		}
		return &Parameter{Name: name}, nil
	}
	if tok.kind == tokIdent {
		p.next()
		name := tok.text
		qualifier := ""
		if p.match(tokDot) {
			if p.match(tokStar) {
				return &Literal{Kind: "star", Value: name + ".*"}, nil
			}
			col, err := p.expect(tokIdent)
			if err != nil {
				return nil, err
			}
			qualifier = name
			name = col.text
		}
		if strings.ToLower(name) == "case" {
			cse := &Case{}
			if !p.matchKeyword("when") {
				return nil, p.error("expected WHEN after CASE")
			}
			for {
				cond, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				if !p.matchKeyword("then") {
					return nil, p.error("expected THEN")
				}
				then, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				cse.Whens = append(cse.Whens, When{Cond: cond, Then: then})
				if p.matchKeyword("else") {
					elseExpr, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					cse.Else = elseExpr
					break
				}
				if !p.matchKeyword("when") {
					break
				}
			}
			if !p.matchKeyword("end") {
				return nil, p.error("expected END")
			}
			return cse, nil
		}
		if p.match(tokLParen) {
			fn := &FunctionCall{Name: strings.ToLower(name)}
			if p.match(tokStar) {
				fn.Args = []Expr{&Literal{Kind: "star", Value: "*"}}
				_, err := p.expect(tokRParen)
				if err != nil {
					return nil, err
				}
				return fn, nil
			}
			if p.matchKeyword("distinct") {
				fn.Distinct = true
			}
			for {
				arg, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				fn.Args = append(fn.Args, arg)
				if !p.match(tokComma) {
					break
				}
			}
			_, err := p.expect(tokRParen)
			if err != nil {
				return nil, err
			}
			return fn, nil
		}
		if qualifier != "" {
			return &Column{Qualifier: qualifier, Name: name}, nil
		}
		return &Column{Name: name}, nil
	}
	return nil, fmt.Errorf("unexpected token %s at %d", tok.text, tok.pos)
}

func (p *parser) parseWhere(b *queryBuilder) (Node, error) {
	if !p.matchKeyword("where") {
		return nil, nil
	}
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &Filter{NodeID: p.nextNodeID(), Condition: cond}, nil
}

func (p *parser) parseGroupBy() ([]Expr, error) {
	if !p.matchKeyword("group") {
		return nil, nil
	}
	if !p.matchKeyword("by") {
		return nil, nil
	}
	var items []Expr
	for {
		expr, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		items = append(items, expr)
		if !p.match(tokComma) {
			break
		}
	}
	return items, nil
}

func (p *parser) parseHaving() (Node, error) {
	if !p.matchKeyword("having") {
		return nil, nil
	}
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &Filter{NodeID: p.nextNodeID(), Condition: cond, Having: true}, nil
}

func (p *parser) parseOrderBy() (Node, error) {
	if !p.matchKeyword("order") {
		return nil, nil
	}
	if !p.matchKeyword("by") {
		return nil, nil
	}
	var keys []SortKey
	for {
		expr, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		desc := false
		if p.matchKeyword("desc") {
			desc = true
		} else if p.matchKeyword("asc") {
			desc = false
		}
		keys = append(keys, SortKey{Expr: expr, Desc: desc})
		if !p.match(tokComma) {
			break
		}
	}
	return &Sort{NodeID: p.nextNodeID(), Keys: keys}, nil
}

func (p *parser) parseLimit() (Node, error) {
	if !p.matchKeyword("limit") {
		return nil, nil
	}
	limitTok := p.peek()
	limitVal := ""
	if limitTok.kind == tokNumber {
		limitVal = limitTok.text
		p.next()
	} else if p.match(tokQuestion) {
		limitVal = "?"
	} else {
		return nil, p.error("expected LIMIT value")
	}
	offsetVal := ""
	if p.matchKeyword("offset") {
		otok := p.peek()
		if otok.kind == tokNumber {
			offsetVal = otok.text
			p.next()
		} else if p.match(tokQuestion) {
			offsetVal = "?"
		}
	}
	return &Limit{NodeID: p.nextNodeID(), Limit: limitVal, Offset: offsetVal}, nil
}
