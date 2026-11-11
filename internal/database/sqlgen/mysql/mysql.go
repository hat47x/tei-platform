package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sqlgen "tei/internal/database/sqlgen"
)

// Dialect is the MySQL dialect adapter.
type Dialect struct{}

// NewDialect creates a new MySQL dialect adapter.
func NewDialect() *Dialect {
	return &Dialect{}
}

// PhysicalType maps canonical TEI storage types to MySQL physical types.
func (d *Dialect) PhysicalType(storage sqlgen.StorageType) (string, error) {
	switch storage {
	case sqlgen.StorageInt64:
		return "BIGINT", nil
	case sqlgen.StorageVarchar:
		return "VARCHAR(255)", nil
	case sqlgen.StorageText:
		return "TEXT", nil
	case sqlgen.StorageDateTimeTZ:
		return "DATETIME(6)", nil
	case sqlgen.StorageDecimal:
		return "DECIMAL(18,2)", nil
	case sqlgen.StorageJSON:
		return "JSON", nil
	default:
		return "", fmt.Errorf("unsupported storage type %q for MySQL", storage)
	}
}

// CanStore reports whether MySQL can persist the canonical storage type.
func (d *Dialect) CanStore(storage sqlgen.StorageType) bool {
	switch storage {
	case sqlgen.StorageInt64, sqlgen.StorageVarchar, sqlgen.StorageText,
		sqlgen.StorageDateTimeTZ, sqlgen.StorageDecimal, sqlgen.StorageJSON:
		return true
	default:
		return false
	}
}

// PlaceholderStyle returns the MySQL placeholder style (?).
func (d *Dialect) PlaceholderStyle() sqlgen.PlaceholderStyle {
	return sqlgen.PlaceholderPositionalQuestion
}

// Name returns the dialect identifier.
func (d *Dialect) Name() string {
	return "mysql"
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
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "unique constraint")
}

// IsForeignKeyViolation reports whether err is a foreign key violation.
func (d *Dialect) IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "foreign key constraint fails") ||
		strings.Contains(msg, "foreign key constraint")
}

// IsNullViolation reports whether err is a NOT NULL violation.
func (d *Dialect) IsNullViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cannot be null") ||
		strings.Contains(msg, "field doesn't have a default value")
}

// IsCheckViolation reports whether err is a CHECK constraint violation.
func (d *Dialect) IsCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "check constraint") ||
		strings.Contains(msg, "check constraint is violated")
}

// Select builds a SELECT statement with ? placeholders.
func (d *Dialect) Select(ctx context.Context, opts sqlgen.SelectOptions) (sqlgen.SQL, error) {
	var sb strings.Builder

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

	if strings.TrimSpace(opts.Window) != "" {
		sb.WriteString(" WINDOW ")
		sb.WriteString(opts.Window)
	}

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
		}
	}

	if opts.Limit != nil {
		sb.WriteString(" LIMIT ")
		sb.WriteString(fmt.Sprintf("%d", *opts.Limit))
	}

	if opts.Offset != nil {
		sb.WriteString(" OFFSET ")
		sb.WriteString(fmt.Sprintf("%d", *opts.Offset))
	}

	if opts.SetOperation != "" {
		suffix := string(opts.SetOperation)
		if opts.All {
			suffix += " ALL"
		}
		sb.WriteString(" ")
		sb.WriteString(suffix)
	}

	args := buildArgs(opts.Parameters, opts.Where, opts.Having)
	return sqlgen.SQL{Statement: sb.String(), Args: args}, nil
}

// Insert builds an INSERT statement with ? placeholders.
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
		placeholders[i] = "?"
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

	// MySQL does not support RETURNING.
	_ = opts.Returning

	return sqlgen.SQL{Statement: sb.String(), Args: args}, nil
}

// Update builds an UPDATE statement with ? placeholders.
func (d *Dialect) Update(ctx context.Context, opts sqlgen.UpdateOptions) (sqlgen.SQL, error) {
	if len(opts.Assignments) == 0 {
		return sqlgen.SQL{}, fmt.Errorf("UPDATE requires at least one assignment")
	}

	assignments := make([]string, 0, len(opts.Assignments))
	args := make([]any, 0, len(opts.Assignments))
	for col, val := range opts.Assignments {
		assignments = append(assignments, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(opts.Table)
	sb.WriteString(" SET ")
	sb.WriteString(strings.Join(assignments, ", "))

	if len(opts.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(buildConditions(opts.Where, ""))
	}

	// MySQL does not support RETURNING.
	_ = opts.Returning

	args = append(args, buildArgs(opts.Parameters, opts.Where, nil)...)
	return sqlgen.SQL{Statement: sb.String(), Args: args}, nil
}

// Delete builds a DELETE statement with ? placeholders.
func (d *Dialect) Delete(ctx context.Context, opts sqlgen.DeleteOptions) (sqlgen.SQL, error) {
	var sb strings.Builder
	sb.WriteString("DELETE FROM ")
	sb.WriteString(opts.Table)

	if len(opts.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(buildConditions(opts.Where, ""))
	}

	// MySQL does not support RETURNING.
	_ = opts.Returning

	args := buildArgs(opts.Parameters, opts.Where, nil)
	return sqlgen.SQL{Statement: sb.String(), Args: args}, nil
}

func buildConditions(conditions []sqlgen.Condition, _ string) string {
	if len(conditions) == 0 {
		return ""
	}
	var sb strings.Builder
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
			sb.WriteString(buildInList(c.Value))
			sb.WriteString(")")
		case "NOT IN":
			sb.WriteString(fmt.Sprintf("%s NOT IN (", c.Column))
			sb.WriteString(buildInList(c.Value))
			sb.WriteString(")")
		default:
			sb.WriteString(fmt.Sprintf("%s %s ?", c.Column, c.Operator))
		}
	}
	return sb.String()
}

func buildInList(value any) string {
	switch v := value.(type) {
	case []any:
		parts := make([]string, len(v))
		for i := range v {
			parts[i] = "?"
		}
		return strings.Join(parts, ", ")
	default:
		return "?"
	}
}

func buildArgs(parameters map[string]any, where []sqlgen.Condition, _ []sqlgen.Condition) []any {
	args := make([]any, 0)
	for _, v := range parameters {
		args = append(args, v)
	}
	for _, c := range where {
		if strings.TrimSpace(c.Expression) != "" {
			continue
		}
		if sqlgen.IsComparison(c.Operator) {
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
	return args
}
