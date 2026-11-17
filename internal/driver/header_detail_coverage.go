package driver

import (
	"fmt"
	"strings"
	"unicode"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/environment"
	"tei/internal/realization"
)

// HeaderDetailCoverage says whether the rows a header-detail Contract Query
// returns are every row of its Relation for the header record.
//
// The binding's `covers` is the author's statement and is not enough on its
// own: a Query can keep the statement and still drop a row. Complete is
// therefore reported only when the Query that runs can be matched, without a
// general SQL prover, against the way the Workspace stores the Relation:
//
//   - the binding states `covers <the Pattern's Relation>`;
//   - the SQL selects from the table the persistence mapping of the
//     Relation's target names, and from nothing else;
//   - its only condition is that each column the relation storage of that
//     Relation uses equals a parameter, joined by AND;
//   - each such parameter receives, through the realization's input and the
//     binding's input, the very source attribute the relation storage stores
//     in that column;
//   - nothing else limits the rows: no LIMIT, OFFSET, DISTINCT, GROUP BY,
//     join, sub-query, comment, or second statement.
//
// Anything the matcher does not recognise leaves the coverage partial. A row
// count, an absent LIMIT, an empty result, or the declaration alone never
// makes it complete.
type HeaderDetailCoverage struct {
	// Complete is true only when every condition above was matched.
	Complete bool `json:"complete"`
	// Declared is what the binding states: the Relation, "some", or "".
	Declared string `json:"declared"`
	// Basis says, when Complete, which definitions the Query was matched
	// against. Reason says, when not, the first condition that failed.
	Basis  string `json:"basis,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// QualifyHeaderDetailCoverage matches the Query a header-detail Pattern runs
// on provider against the Workspace's relation storage. It never returns an
// error: a Query that cannot be matched is partial, with the reason.
func (r Resolver) QualifyHeaderDetailCoverage(resolution ResolvedHeaderDetailContractQuery, provider environment.DatabaseProvider) HeaderDetailCoverage {
	binding := resolution.Binding
	relation := resolution.Relation.Relation.ID
	out := HeaderDetailCoverage{Declared: string(binding.CoversRelation)}
	if binding.CoversSome {
		out.Declared = "some"
	}
	partial := func(format string, args ...any) HeaderDetailCoverage {
		out.Reason = fmt.Sprintf(format, args...)
		return out
	}

	switch {
	case binding.CoversSome:
		return partial("the binding states that the Query returns only some rows")
	case binding.CoversRelation == "":
		return partial("the binding does not state that the Query returns every row of %s", relation)
	case binding.CoversRelation != relation:
		return partial("the binding states that the Query returns every row of %s, but the Pattern reads %s", binding.CoversRelation, relation)
	}

	var sqlRealization *realization.ContractQuerySQLRealization
	for i := range r.world.ContractQuerySQLRealizations {
		candidate := &r.world.ContractQuerySQLRealizations[i]
		if candidate.Target != binding.Target || candidate.Provider != provider {
			continue
		}
		if sqlRealization != nil {
			return partial("Contract Query %s has more than one SQL realization for %s", binding.Target.Identity(), provider)
		}
		sqlRealization = candidate
	}
	if sqlRealization == nil {
		return partial("Contract Query %s has no SQL realization for %s", binding.Target.Identity(), provider)
	}

	target := resolution.Relation.Target.ID
	var mapping *persistence.Mapping
	for i := range r.world.Persistence {
		candidate := &r.world.Persistence[i]
		if candidate.Database == binding.Target.Database && candidate.SemanticData == target {
			if mapping != nil {
				return partial("database %s has more than one persistence mapping of %s", binding.Target.Database, target)
			}
			mapping = candidate
		}
	}
	if mapping == nil {
		return partial("database %s has no persistence mapping of %s", binding.Target.Database, target)
	}
	var storage *persistence.RelationStorageMapping
	for i := range mapping.RelationStorage {
		if mapping.RelationStorage[i].Relation == relation {
			if storage != nil {
				return partial("persistence mapping %s declares storage of %s more than once", mapping.ID, relation)
			}
			storage = &mapping.RelationStorage[i]
		}
	}
	if storage == nil || len(storage.Components) == 0 {
		return partial("persistence mapping %s does not say how %s is stored", mapping.ID, relation)
	}

	shape, err := parseRelationRead(sqlRealization.Source)
	if err != nil {
		return partial("the SQL of %s is not a plain read of one table by its parent columns: %v", sqlRealization.Identity(), err)
	}
	if !strings.EqualFold(shape.table, mapping.Table) {
		return partial("the SQL reads %s, but %s is stored in %s", shape.table, target, mapping.Table)
	}

	parameterOf := map[string]string{} // lower-case column -> parameter
	for _, condition := range shape.conditions {
		column := strings.ToLower(condition.column)
		if _, twice := parameterOf[column]; twice {
			return partial("the SQL compares column %s twice", condition.column)
		}
		parameterOf[column] = condition.parameter
	}
	if len(parameterOf) != len(storage.Components) {
		return partial("the SQL compares %d column(s), but %s is stored in %d", len(parameterOf), relation, len(storage.Components))
	}
	matched := make([]string, 0, len(storage.Components))
	for _, component := range storage.Components {
		parameter, compared := parameterOf[strings.ToLower(component.Column)]
		if !compared {
			return partial("the SQL does not compare column %s, which stores %s of %s", component.Column, component.SourceAttribute, relation)
		}
		var input core.LocalName
		found := false
		for _, mapped := range sqlRealization.Inputs {
			if mapped.Parameter != parameter {
				continue
			}
			if mapped.Nullable {
				return partial("parameter %s may be null; a null key reads no rows", parameter)
			}
			input, found = mapped.Input, true
		}
		if !found {
			return partial("parameter %s is not an input of %s", parameter, sqlRealization.Identity())
		}
		fromSource := false
		for _, bound := range binding.Inputs {
			if bound.Input == input && bound.SourceAttribute == component.SourceAttribute {
				fromSource = true
			}
		}
		if !fromSource {
			return partial("column %s stores %s, but input %s is not bound to it", component.Column, component.SourceAttribute, input)
		}
		matched = append(matched, fmt.Sprintf("%s = %s", component.Column, component.SourceAttribute))
	}

	out.Complete = true
	out.Basis = fmt.Sprintf("the SQL of %s reads %s where %s, as persistence mapping %s stores %s, with no other condition or limit",
		sqlRealization.Identity(), mapping.Table, strings.Join(matched, " and "), mapping.ID, relation)
	return out
}

// relationRead is the one shape of SQL the coverage matcher accepts:
//
//	SELECT column, ... FROM table WHERE column = :parameter [AND ...] [ORDER BY column [ASC|DESC], ...] [;]
type relationRead struct {
	table      string
	conditions []relationCondition
}

type relationCondition struct {
	column    string
	parameter string
}

// Words that change which rows a statement returns, or that this matcher does
// not read, are never taken for a column name.
var relationReadReserved = map[string]bool{
	"select": true, "from": true, "where": true, "and": true, "or": true, "not": true,
	"order": true, "by": true, "asc": true, "desc": true, "limit": true, "offset": true,
	"fetch": true, "distinct": true, "all": true, "union": true, "intersect": true, "except": true,
	"group": true, "having": true, "join": true, "on": true, "using": true, "in": true,
	"is": true, "null": true, "like": true, "between": true, "case": true, "when": true,
	"with": true, "as": true, "window": true, "for": true, "into": true, "lateral": true,
	"top": true, "only": true, "tablesample": true, "nulls": true,
}

func parseRelationRead(source string) (relationRead, error) {
	tokens, err := relationReadTokens(source)
	if err != nil {
		return relationRead{}, err
	}
	p := relationReadParser{tokens: tokens}
	var out relationRead

	if err := p.keyword("select"); err != nil {
		return out, err
	}
	for {
		if _, err := p.name(); err != nil {
			return out, fmt.Errorf("the select list: %w", err)
		}
		if !p.symbol(",") {
			break
		}
	}
	if err := p.keyword("from"); err != nil {
		return out, err
	}
	table, err := p.name()
	if err != nil {
		return out, fmt.Errorf("the table: %w", err)
	}
	if p.symbol(".") {
		second, err := p.name()
		if err != nil {
			return out, fmt.Errorf("the table: %w", err)
		}
		table += "." + second
	}
	out.table = table
	if err := p.keyword("where"); err != nil {
		return out, err
	}
	for {
		column, err := p.name()
		if err != nil {
			return out, fmt.Errorf("a condition: %w", err)
		}
		if !p.symbol("=") {
			return out, fmt.Errorf("a condition on %s is not an equality", column)
		}
		if !p.symbol(":") {
			return out, fmt.Errorf("%s is not compared with a parameter", column)
		}
		parameter, err := p.name()
		if err != nil {
			return out, fmt.Errorf("the parameter of %s: %w", column, err)
		}
		out.conditions = append(out.conditions, relationCondition{column: column, parameter: parameter})
		if !p.isKeyword("and") {
			break
		}
		p.next()
	}
	if p.isKeyword("order") {
		p.next()
		if err := p.keyword("by"); err != nil {
			return out, err
		}
		for {
			if _, err := p.name(); err != nil {
				return out, fmt.Errorf("the order: %w", err)
			}
			if p.isKeyword("asc") || p.isKeyword("desc") {
				p.next()
			}
			if !p.symbol(",") {
				break
			}
		}
	}
	p.symbol(";")
	if !p.done() {
		return out, fmt.Errorf("%q follows the statement", p.peek())
	}
	return out, nil
}

func relationReadTokens(source string) ([]string, error) {
	var tokens []string
	runes := []rune(source)
	for i := 0; i < len(runes); {
		c := runes[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '_' || unicode.IsLetter(c):
			start := i
			for i < len(runes) && (runes[i] == '_' || unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i])) {
				i++
			}
			tokens = append(tokens, string(runes[start:i]))
		case strings.ContainsRune(",.=:;", c):
			tokens = append(tokens, string(c))
			i++
		default:
			// Quotes, comments, operators, numbers, parentheses and the
			// like are not read: the statement is left partial.
			return nil, fmt.Errorf("%q is not read by the coverage matcher", string(c))
		}
	}
	return tokens, nil
}

type relationReadParser struct {
	tokens []string
	at     int
}

func (p *relationReadParser) done() bool { return p.at >= len(p.tokens) }

func (p *relationReadParser) peek() string {
	if p.done() {
		return ""
	}
	return p.tokens[p.at]
}

func (p *relationReadParser) next() { p.at++ }

func (p *relationReadParser) isKeyword(word string) bool {
	return strings.EqualFold(p.peek(), word)
}

func (p *relationReadParser) keyword(word string) error {
	if !p.isKeyword(word) {
		if p.done() {
			return fmt.Errorf("%s is missing", strings.ToUpper(word))
		}
		return fmt.Errorf("%q is where %s is expected", p.peek(), strings.ToUpper(word))
	}
	p.next()
	return nil
}

func (p *relationReadParser) symbol(s string) bool {
	if p.peek() != s {
		return false
	}
	p.next()
	return true
}

func (p *relationReadParser) name() (string, error) {
	word := p.peek()
	if word == "" {
		return "", fmt.Errorf("a name is missing")
	}
	first := []rune(word)[0]
	if !(first == '_' || unicode.IsLetter(first)) {
		return "", fmt.Errorf("%q is not a name", word)
	}
	if relationReadReserved[strings.ToLower(word)] {
		return "", fmt.Errorf("%q is not read as a name", word)
	}
	p.next()
	return word, nil
}
