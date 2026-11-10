package driver

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/sqlgen"
	"tei/internal/query"
)

// RelationQuerySQL builds dialect-neutral SQL from a resolved relation query.
type RelationQuerySQL struct {
	Dialect    sqlgen.Dialect
	Resolved   query.ResolvedRelationQuery
	Parameters map[string]any
	Mappings   []persistence.Mapping
}

// BuildSelect builds a SELECT statement for the relation query.
func (b *RelationQuerySQL) BuildSelect() (sqlgen.SQL, error) {
	if len(b.Resolved.Steps) == 0 {
		return sqlgen.SQL{}, fmt.Errorf("relation query %s has no relation steps", b.Resolved.Query.ID)
	}
	root := b.Resolved.RootData
	rootTable := persistenceTable(root.ID, b.Mappings)
	if rootTable == "" {
		return sqlgen.SQL{}, fmt.Errorf("no persistence mapping for root data %s", root.ID)
	}
	columns, err := b.buildColumns(b.Mappings)
	if err != nil {
		return sqlgen.SQL{}, err
	}
	joins, err := b.buildJoins(b.Mappings)
	if err != nil {
		return sqlgen.SQL{}, err
	}
	where, err := b.buildWhere()
	if err != nil {
		return sqlgen.SQL{}, err
	}
	orderBy, err := b.buildOrderBy()
	if err != nil {
		return sqlgen.SQL{}, err
	}
	opts := sqlgen.SelectOptions{
		Table:      rootTable,
		Columns:    columns,
		Joins:      joins,
		Where:      where,
		OrderBy:    orderBy,
		Limit:      intPtr(b.Resolved.Query.PageSize),
		Offset:     intPtr(b.Resolved.Query.PageOffset),
		Parameters: b.Parameters,
	}
	return b.Dialect.Select(nil, opts)
}

func (b *RelationQuerySQL) buildColumns(mappings []persistence.Mapping) ([]string, error) {
	seen := map[string]bool{}
	var cols []string
	rootTable := persistenceTable(b.Resolved.RootData.ID, mappings)
	for _, col := range b.Resolved.Query.ResultColumns {
		table, column, err := lookupColumn(col.Attribute, mappings)
		if err != nil {
			return nil, err
		}
		if table == "" {
			table = rootTable
		}
		ref := fmt.Sprintf("%s.%s", table, column)
		if seen[ref] {
			continue
		}
		seen[ref] = true
		cols = append(cols, ref)
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("relation query %s has no resolvable result columns", b.Resolved.Query.ID)
	}
	return cols, nil
}

func (b *RelationQuerySQL) buildJoins(mappings []persistence.Mapping) ([]sqlgen.JoinClause, error) {
	var joins []sqlgen.JoinClause
	for i, step := range b.Resolved.Steps {
		targetTable := persistenceTable(step.Target.ID, mappings)
		if targetTable == "" {
			return nil, fmt.Errorf("no persistence mapping for target data %s in step %d", step.Target.ID, i)
		}
		on, err := b.buildJoinCondition(step, mappings)
		if err != nil {
			return nil, err
		}
		kind := "LEFT"
		if step.Step.JoinKind != "" {
			kind = step.Step.JoinKind
		}
		alias := fmt.Sprintf("rel_%d", i)
		if step.Step.Alias != "" {
			alias = step.Step.Alias
		}
		joins = append(joins, sqlgen.JoinClause{
			Kind:  kind,
			Table: targetTable,
			Alias: alias,
			On:    on,
		})
	}
	return joins, nil
}

func (b *RelationQuerySQL) buildJoinCondition(step query.ResolvedRelationStep, mappings []persistence.Mapping) ([]sqlgen.Condition, error) {
	var conditions []sqlgen.Condition
	for _, key := range step.Relation.Keys {
		srcTable, srcCol, err := lookupColumn(key.SourceAttribute, mappings)
		if err != nil {
			return nil, err
		}
		tgtTable, tgtCol, err := lookupColumn(key.TargetAttribute, mappings)
		if err != nil {
			return nil, err
		}
		if srcTable == "" || tgtTable == "" {
			return nil, fmt.Errorf("missing persistence mapping for join between %s and %s", step.Source.ID, step.Target.ID)
		}
		srcRef := fmt.Sprintf("%s.%s", srcTable, srcCol)
		tgtRef := fmt.Sprintf("%s.%s", tgtTable, tgtCol)
		conditions = append(conditions, sqlgen.Condition{
			Column:    srcRef,
			Operator:  "=",
			Value:     tgtRef,
			LogicalOp: "AND",
		})
	}
	if len(conditions) == 0 {
		return nil, fmt.Errorf("relation %s has no keys for join condition", step.Relation.ID)
	}
	conditions[0].LogicalOp = ""
	return conditions, nil
}

func (b *RelationQuerySQL) buildWhere() ([]sqlgen.Condition, error) {
	var where []sqlgen.Condition
	for _, f := range b.Resolved.Query.Filters {
		where = append(where, sqlgen.Condition{
			Expression: f.Expression,
		})
	}
	return where, nil
}

func (b *RelationQuerySQL) buildOrderBy() ([]sqlgen.OrderClause, error) {
	var orderBy []sqlgen.OrderClause
	for _, s := range b.Resolved.Query.Sorts {
		desc := false
		expr := s.Expression
		if stringsHasSuffix(expr, " desc") {
			desc = true
			expr = stringsTrimSuffix(expr, " desc")
		} else if stringsHasSuffix(expr, " DESC") {
			desc = true
			expr = stringsTrimSuffix(expr, " DESC")
		}
		orderBy = append(orderBy, sqlgen.OrderClause{
			Column: expr,
			Desc:   desc,
		})
	}
	return orderBy, nil
}

func persistenceTable(dataID core.CanonicalID, mappings []persistence.Mapping) string {
	for _, m := range mappings {
		if m.SemanticData == dataID {
			return m.Table
		}
	}
	return ""
}

func lookupColumn(attr core.CanonicalID, mappings []persistence.Mapping) (table, column string, err error) {
	for _, m := range mappings {
		for _, col := range m.Columns {
			if col.Attribute == attr {
				return m.Table, col.Column, nil
			}
		}
	}
	return "", "", fmt.Errorf("no persistence mapping for attribute %s", attr)
}

func intPtr(n int) *int {
	if n <= 0 {
		return nil
	}
	return &n
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func stringsTrimSuffix(s, suffix string) string {
	if stringsHasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}
