package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	sqlgen "tei/internal/database/sqlgen"
)

// Dialect is the PostgreSQL dialect adapter.
type Dialect struct{}

// NewDialect creates a new PostgreSQL dialect adapter.
func NewDialect() *Dialect {
	return &Dialect{}
}

// PhysicalType maps canonical TEI storage types to PostgreSQL physical types.
func (d *Dialect) PhysicalType(storage sqlgen.StorageType) (string, error) {
	switch storage {
	case sqlgen.StorageInt64:
		return "bigint", nil
	case sqlgen.StorageVarchar:
		return "varchar(255)", nil
	case sqlgen.StorageText:
		return "text", nil
	case sqlgen.StorageDateTimeTZ:
		return "timestamptz", nil
	case sqlgen.StorageDecimal:
		return "numeric(18,2)", nil
	case sqlgen.StorageJSON:
		return "jsonb", nil
	default:
		return "", fmt.Errorf("unsupported storage type %q for PostgreSQL", storage)
	}
}

// CanStore reports whether PostgreSQL can persist the canonical storage type.
func (d *Dialect) CanStore(storage sqlgen.StorageType) bool {
	switch storage {
	case sqlgen.StorageInt64, sqlgen.StorageVarchar, sqlgen.StorageText,
		sqlgen.StorageDateTimeTZ, sqlgen.StorageDecimal, sqlgen.StorageJSON:
		return true
	default:
		return false
	}
}

// PlaceholderStyle returns the PostgreSQL placeholder style ($1, $2, ...).
func (d *Dialect) PlaceholderStyle() sqlgen.PlaceholderStyle {
	return sqlgen.PlaceholderPositionalDollar
}

// Name returns the dialect identifier.
func (d *Dialect) Name() string {
	return "postgres"
}

// BeginTransaction starts a transaction on the given *sql.DB.
func (d *Dialect) BeginTransaction(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	if db == nil {
		return nil, fmt.Errorf("cannot begin transaction with nil *sql.DB")
	}
	return db.BeginTx(ctx, nil)
}

// IsUniqueViolation reports whether err is a unique constraint violation.
func (d *Dialect) IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "pq: duplicate key")
}

// IsForeignKeyViolation reports whether err is a foreign key violation.
func (d *Dialect) IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "foreign key constraint") ||
		strings.Contains(msg, "violates foreign key")
}

// IsNullViolation reports whether err is a NOT NULL violation.
func (d *Dialect) IsNullViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not null constraint") ||
		strings.Contains(msg, "null value")
}

// IsCheckViolation reports whether err is a CHECK constraint violation.
func (d *Dialect) IsCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "check constraint") ||
		strings.Contains(msg, "violates check")
}

// Select builds a SELECT statement.
func (d *Dialect) Select(ctx context.Context, opts sqlgen.SelectOptions) (sqlgen.SQL, error) {
	var sb strings.Builder

	// A CTE is a WITH clause in front of the statement, so it has to be written
	// before the SELECT keyword rather than anywhere inside the body.
	if strings.TrimSpace(opts.CTE) != "" {
		sb.WriteString("WITH ")
		sb.WriteString(opts.CTE)
		sb.WriteString(" ")
	}

	sb.WriteString("SELECT ")

	if len(opts.Columns) == 0 {
		return sqlgen.SQL{}, fmt.Errorf("SELECT requires at least one column")
	}
	sb.WriteString(strings.Join(opts.Columns, ", "))

	sb.WriteString(" FROM ")
	sb.WriteString(opts.Table)

	for _, j := range opts.Joins {
		sb.WriteString(" ")
		sb.WriteString(j.Kind)
		sb.WriteString(" JOIN ")
		sb.WriteString(j.Table)
		if j.Alias != "" {
			sb.WriteString(" AS ")
			sb.WriteString(j.Alias)
		}
		if len(j.On) > 0 {
			sb.WriteString(" ON ")
			sb.WriteString(buildConditions(j.On, ""))
		}
	}

	if len(opts.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(buildConditions(opts.Where, ""))
	}

	if len(opts.GroupBy) > 0 {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(strings.Join(opts.GroupBy, ", "))
	}

	if len(opts.Having) > 0 {
		sb.WriteString(" HAVING ")
		sb.WriteString(buildConditions(opts.Having, ""))
	}

	// A named window definition belongs to the whole query rather than to one
	// select item, so it follows GROUP BY / HAVING and precedes ORDER BY. Putting
	// it right after the select list is a syntax error in PostgreSQL.
	if strings.TrimSpace(opts.Window) != "" {
		sb.WriteString(" WINDOW ")
		sb.WriteString(opts.Window)
	}

	if len(opts.OrderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		for i, o := range opts.OrderBy {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(o.Column)
			if o.Desc {
				sb.WriteString(" DESC")
			}
			if strings.ToUpper(o.Nulls) == "FIRST" {
				sb.WriteString(" NULLS FIRST")
			} else if strings.ToUpper(o.Nulls) == "LAST" {
				sb.WriteString(" NULLS LAST")
			}
		}
	}

	if opts.Limit != nil {
		sb.WriteString(" LIMIT ")
		sb.WriteString(strconv.Itoa(*opts.Limit))
	}

	if opts.Offset != nil {
		sb.WriteString(" OFFSET ")
		sb.WriteString(strconv.Itoa(*opts.Offset))
	}

	if opts.ForUpdate {
		sb.WriteString(" FOR UPDATE")
	} else if opts.ForShare {
		sb.WriteString(" FOR SHARE")
	}

	// A set operation combines this query with the next one, so it closes the
	// statement: everything a caller appends afterwards belongs to the other
	// side of the operation, not to this one.
	if opts.SetOperation != "" {
		sb.WriteString(" ")
		sb.WriteString(setOperationSuffix(opts.SetOperation, opts.All))
	}

	args := buildArgs(opts.Parameters, opts.Where, opts.Having)
	return sqlgen.SQL{Statement: sb.String(), Args: args}, nil
}

// setOperationSuffix renders the set operation keyword, adding ALL when the
// caller keeps duplicates.
func setOperationSuffix(operation sqlgen.SetOperation, all bool) string {
	suffix := string(operation)
	if all {
		suffix += " ALL"
	}
	return suffix
}

// Insert builds an INSERT statement.
func (d *Dialect) Insert(ctx context.Context, opts sqlgen.InsertOptions) (sqlgen.SQL, error) {
	if len(opts.Columns) == 0 {
		return sqlgen.SQL{}, fmt.Errorf("INSERT requires at least one column")
	}
	if len(opts.Columns) != len(opts.Values) {
		return sqlgen.SQL{}, fmt.Errorf("INSERT columns and values count mismatch: %d != %d", len(opts.Columns), len(opts.Values))
	}

	cols := strings.Join(opts.Columns, ", ")
	placeholders := make([]string, len(opts.Columns))
	args := make([]any, 0, len(opts.Columns))
	for i := range opts.Columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args = append(args, opts.Values[opts.Columns[i]])
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(opts.Table)
	sb.WriteString(" (")
	sb.WriteString(cols)
	sb.WriteString(") VALUES (")
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	if len(opts.Returning) > 0 {
		sb.WriteString(" RETURNING ")
		sb.WriteString(strings.Join(opts.Returning, ", "))
	}

	return sqlgen.SQL{Statement: sb.String(), Args: args}, nil
}

// Update builds an UPDATE statement.
func (d *Dialect) Update(ctx context.Context, opts sqlgen.UpdateOptions) (sqlgen.SQL, error) {
	if len(opts.Assignments) == 0 {
		return sqlgen.SQL{}, fmt.Errorf("UPDATE requires at least one assignment")
	}

	assignments := make([]string, 0, len(opts.Assignments))
	args := make([]any, 0, len(opts.Assignments))
	offset := 0
	for col, val := range opts.Assignments {
		offset++
		assignments = append(assignments, fmt.Sprintf("%s = $%d", col, offset))
		args = append(args, val)
	}

	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(opts.Table)
	sb.WriteString(" SET ")
	sb.WriteString(strings.Join(assignments, ", "))

	if len(opts.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(buildConditions(opts.Where, strconv.Itoa(offset+1)))
	}

	if len(opts.Returning) > 0 {
		sb.WriteString(" RETURNING ")
		sb.WriteString(strings.Join(opts.Returning, ", "))
	}

	args = append(args, buildArgs(opts.Parameters, opts.Where, nil)...)
	return sqlgen.SQL{Statement: sb.String(), Args: args}, nil
}

// Delete builds a DELETE statement.
func (d *Dialect) Delete(ctx context.Context, opts sqlgen.DeleteOptions) (sqlgen.SQL, error) {
	var sb strings.Builder
	sb.WriteString("DELETE FROM ")
	sb.WriteString(opts.Table)

	if len(opts.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(buildConditions(opts.Where, "1"))
	}

	if len(opts.Returning) > 0 {
		sb.WriteString(" RETURNING ")
		sb.WriteString(strings.Join(opts.Returning, ", "))
	}

	args := buildArgs(opts.Parameters, opts.Where, nil)
	return sqlgen.SQL{Statement: sb.String(), Args: args}, nil
}

func buildConditions(conditions []sqlgen.Condition, start string) string {
	if len(conditions) == 0 {
		return ""
	}
	var sb strings.Builder
	index, _ := strconv.Atoi(start)
	if index == 0 {
		index = 1
	}
	for i, c := range conditions {
		if strings.TrimSpace(c.Expression) != "" {
			if i > 0 {
				if strings.ToUpper(c.LogicalOp) == "OR" {
					sb.WriteString(" OR ")
				} else {
					sb.WriteString(" AND ")
				}
			}
			sb.WriteString(c.Expression)
			continue
		}
		if i > 0 {
			if strings.ToUpper(c.LogicalOp) == "OR" {
				sb.WriteString(" OR ")
			} else {
				sb.WriteString(" AND ")
			}
		}
		if c.Not {
			sb.WriteString("NOT ")
		}
		switch strings.ToUpper(c.Operator) {
		case "IS NULL":
			sb.WriteString(fmt.Sprintf("%s IS NULL", c.Column))
		case "IS NOT NULL":
			sb.WriteString(fmt.Sprintf("%s IS NOT NULL", c.Column))
		case "IN":
			sb.WriteString(fmt.Sprintf("%s IN (", c.Column))
			sb.WriteString(buildInList(c.Value, &index))
			sb.WriteString(")")
		case "NOT IN":
			sb.WriteString(fmt.Sprintf("%s NOT IN (", c.Column))
			sb.WriteString(buildInList(c.Value, &index))
			sb.WriteString(")")
		default:
			sb.WriteString(fmt.Sprintf("%s %s $%d", c.Column, c.Operator, index))
			index++
		}
	}
	return sb.String()
}

func buildInList(value any, index *int) string {
	if *index == 0 {
		*index = 1
	}
	switch v := value.(type) {
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmt.Sprintf("$%d", *index)
			*index++
			_ = item
		}
		return strings.Join(parts, ", ")
	default:
		result := fmt.Sprintf("$%d", *index)
		*index++
		return result
	}
}

func buildArgs(parameters map[string]any, where []sqlgen.Condition, having []sqlgen.Condition) []any {
	args := make([]any, 0)
	for _, v := range parameters {
		args = append(args, v)
	}
	for _, c := range where {
		if strings.TrimSpace(c.Expression) != "" {
			continue
		}
		if isComparison(c.Operator) {
			args = append(args, c.Value)
		}
		if strings.ToUpper(c.Operator) == "IN" || strings.ToUpper(c.Operator) == "NOT IN" {
			switch v := c.Value.(type) {
			case []any:
				args = append(args, v...)
			default:
				args = append(args, c.Value)
			}
		}
	}
	for _, c := range having {
		if strings.TrimSpace(c.Expression) != "" {
			continue
		}
		if isComparison(c.Operator) {
			args = append(args, c.Value)
		}
	}
	return args
}

func isComparison(op string) bool {
	switch strings.ToUpper(op) {
	case "=", "!=", "<", ">", "<=", ">=", "LIKE", "NOT LIKE":
		return true
	default:
		return false
	}
}
