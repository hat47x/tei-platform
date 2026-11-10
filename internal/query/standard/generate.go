package standard

import (
	"fmt"
	"strings"
)

// StandardToSQL converts a StandardQuery back to SQL.
func StandardToSQL(q *StandardQuery) (string, []Diagnostic) {
	if q == nil {
		return "", []Diagnostic{{Code: "EMPTY_QUERY", Message: "query is nil"}}
	}
	var diags []Diagnostic
	var sb strings.Builder

	if len(q.CTEs) > 0 {
		sb.WriteString("WITH ")
		for i, cte := range q.CTEs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(cte.ID)
			sb.WriteString(" AS (")
			sql, d := StandardToSQL(cte.Query)
			diags = append(diags, d...)
			sb.WriteString(sql)
			sb.WriteString(")")
		}
		sb.WriteString(" ")
	}

	var scans []*Scan
	var joins []*Join
	var where *Filter
	var having *Filter
	var agg *Aggregate
	var proj *Project
	var sort *Sort
	var limit *Limit
	var hasDistinct bool
	subqueries := make(map[string]*Subquery)

	for _, n := range q.Nodes {
		switch v := n.(type) {
		case *Scan:
			scans = append(scans, v)
		case *Join:
			joins = append(joins, v)
		case *Filter:
			if v.Having {
				having = v
			} else {
				where = v
			}
		case *Aggregate:
			agg = v
		case *Project:
			proj = v
		case *Sort:
			sort = v
		case *Limit:
			limit = v
		case *Distinct:
			hasDistinct = true
		case *Subquery:
			subqueries[v.ID] = v
		}
	}

	sb.WriteString("SELECT ")
	if hasDistinct {
		sb.WriteString("DISTINCT ")
	}
	if proj != nil && len(proj.Items) > 0 {
		for i, item := range proj.Items {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(exprSQL(item.Expr, subqueries))
			if item.Alias != "" {
				sb.WriteString(" AS ")
				sb.WriteString(item.Alias)
			}
		}
	} else {
		sb.WriteString("*")
	}

	sb.WriteString(" FROM ")
	if len(scans) > 0 {
		writeScan(&sb, scans[0], subqueries)
	}
	for _, join := range joins {
		sb.WriteString(" ")
		sb.WriteString(strings.ToUpper(join.Kind))
		sb.WriteString(" JOIN ")
		var right *Scan
		for _, s := range scans {
			if s.ID == join.Right {
				right = s
				break
			}
		}
		if right == nil {
			used := map[string]bool{}
			for _, s := range scans {
				used[s.ID] = true
			}
			for _, s := range scans {
				if !used[s.ID] {
					right = s
					break
				}
			}
		}
		if right != nil {
			writeScan(&sb, right, subqueries)
		} else {
			sb.WriteString("<unknown>")
		}
		if len(join.Using) > 0 {
			sb.WriteString(" USING (")
			for i, c := range join.Using {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(c)
			}
			sb.WriteString(")")
		} else if join.Condition != nil {
			sb.WriteString(" ON ")
			sb.WriteString(exprSQL(join.Condition, subqueries))
		}
	}

	if where != nil {
		sb.WriteString(" WHERE ")
		sb.WriteString(exprSQL(where.Condition, subqueries))
	}

	if agg != nil && len(agg.Grouping) > 0 {
		sb.WriteString(" GROUP BY ")
		for i, g := range agg.Grouping {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(exprSQL(g, subqueries))
		}
	}

	if having != nil {
		sb.WriteString(" HAVING ")
		sb.WriteString(exprSQL(having.Condition, subqueries))
	}

	if sort != nil && len(sort.Keys) > 0 {
		sb.WriteString(" ORDER BY ")
		for i, k := range sort.Keys {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(exprSQL(k.Expr, subqueries))
			if k.Desc {
				sb.WriteString(" DESC")
			} else {
				sb.WriteString(" ASC")
			}
		}
	}

	if limit != nil {
		sb.WriteString(" LIMIT ")
		sb.WriteString(limit.Limit)
		if limit.Offset != "" {
			sb.WriteString(" OFFSET ")
			sb.WriteString(limit.Offset)
		}
	}

	return sb.String(), diags
}

func writeScan(sb *strings.Builder, s *Scan, subqueries map[string]*Subquery) {
	if s.SourceKind == "subquery" && s.SubqueryID != "" {
		if sub, ok := subqueries[s.SubqueryID]; ok && sub.Query != nil {
			sql, _ := StandardToSQL(sub.Query)
			sb.WriteString("(")
			sb.WriteString(sql)
			sb.WriteString(")")
			if s.Alias != "" {
				sb.WriteString(" AS ")
				sb.WriteString(s.Alias)
			}
			return
		}
	}
	sb.WriteString(s.Source)
	if s.Alias != "" {
		sb.WriteString(" AS ")
		sb.WriteString(s.Alias)
	}
}

func exprSQL(e Expr, subqueries map[string]*Subquery) string {
	if e == nil {
		return "<nil>"
	}
	switch v := e.(type) {
	case *Column:
		if v.Qualifier != "" {
			return v.Qualifier + "." + v.Name
		}
		return v.Name
	case *Literal:
		switch v.Kind {
		case "star":
			return "*"
		case "null":
			return "NULL"
		case "bool":
			return strings.ToUpper(v.Value)
		case "string":
			return "'" + strings.ReplaceAll(v.Value, "'", "''") + "'"
		case "number":
			return v.Value
		}
	case *Parameter:
		return v.Name
	case *BinaryOp:
		return "(" + exprSQL(v.Left, subqueries) + " " + strings.ToUpper(v.Op) + " " + exprSQL(v.Right, subqueries) + ")"
	case *UnaryOp:
		return "(" + strings.ToUpper(v.Op) + " " + exprSQL(v.Operand, subqueries) + ")"
	case *FunctionCall:
		args := make([]string, len(v.Args))
		for i, a := range v.Args {
			args[i] = exprSQL(a, subqueries)
		}
		prefix := strings.ToUpper(v.Name)
		if v.Distinct {
			prefix += " DISTINCT"
		}
		return prefix + "(" + strings.Join(args, ", ") + ")"
	case *IsNull:
		if v.Negated {
			return "(" + exprSQL(v.Expr, subqueries) + " IS NOT NULL)"
		}
		return "(" + exprSQL(v.Expr, subqueries) + " IS NULL)"
	case *InList:
		if v.Subquery != nil {
			sql, _ := StandardToSQL(v.Subquery)
			if v.Negated {
				return "(" + exprSQL(v.Expr, subqueries) + " NOT IN (" + sql + "))"
			}
			return "(" + exprSQL(v.Expr, subqueries) + " IN (" + sql + "))"
		}
		items := make([]string, len(v.List))
		for i, it := range v.List {
			items[i] = exprSQL(it, subqueries)
		}
		if v.Negated {
			return "(" + exprSQL(v.Expr, subqueries) + " NOT IN (" + strings.Join(items, ", ") + "))"
		}
		return "(" + exprSQL(v.Expr, subqueries) + " IN (" + strings.Join(items, ", ") + "))"
	case *Between:
		return "(" + exprSQL(v.Expr, subqueries) + " BETWEEN " + exprSQL(v.Low, subqueries) + " AND " + exprSQL(v.High, subqueries) + ")"
	case *Exists:
		if v.Query != nil {
			sql, _ := StandardToSQL(v.Query)
			if v.Negated {
				return "(NOT EXISTS (" + sql + "))"
			}
			return "(EXISTS (" + sql + "))"
		}
		return "(EXISTS (<subquery>))"
	case *Cast:
		return "CAST(" + exprSQL(v.Expr, subqueries) + " AS " + v.Type + ")"
	case *Case:
		var sb strings.Builder
		sb.WriteString("CASE")
		if v.Value != nil {
			sb.WriteString(" ")
			sb.WriteString(exprSQL(v.Value, subqueries))
		}
		for _, w := range v.Whens {
			sb.WriteString(" WHEN ")
			sb.WriteString(exprSQL(w.Cond, subqueries))
			sb.WriteString(" THEN ")
			sb.WriteString(exprSQL(w.Then, subqueries))
		}
		if v.Else != nil {
			sb.WriteString(" ELSE ")
			sb.WriteString(exprSQL(v.Else, subqueries))
		}
		sb.WriteString(" END")
		return sb.String()
	case *SubqueryExpr:
		if v.Query != nil {
			sql, _ := StandardToSQL(v.Query)
			prefix := "(" + sql + ")"
			if v.Lateral {
				prefix = "LATERAL " + prefix
			}
			return prefix
		}
		return "(<subquery>)"
	}
	return fmt.Sprintf("<%T>", e)
}
