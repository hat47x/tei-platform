package sqlgen

import (
	"context"
	"database/sql"
)

// PlaceholderStyle describes how a dialect binds runtime parameters.
type PlaceholderStyle string

const (
	// PlaceholderPositionalDollar uses $1, $2, ... (PostgreSQL).
	PlaceholderPositionalDollar PlaceholderStyle = "positional-dollar"
	// PlaceholderPositionalQuestion uses ? (SQLite, MySQL).
	PlaceholderPositionalQuestion PlaceholderStyle = "positional-question"
	// PlaceholderNamedColon uses :name (Oracle named binding via database/sql).
	PlaceholderNamedColon PlaceholderStyle = "named-colon"
)

// StorageType is a canonical TEI storage type.
type StorageType string

const (
	// StorageInt64 represents a 64-bit integer.
	StorageInt64 StorageType = "int64"
	// StorageVarchar represents a variable-length string.
	StorageVarchar StorageType = "varchar"
	// StorageText represents a text string.
	StorageText StorageType = "text"
	// StorageDateTimeTZ represents a timestamp with timezone.
	StorageDateTimeTZ StorageType = "datetime-tz"
	// StorageDecimal represents a decimal number.
	StorageDecimal StorageType = "decimal"
	// StorageJSON represents JSON data.
	StorageJSON StorageType = "json"
)

// TypeMapper maps canonical TEI storage types to physical database types.
type TypeMapper interface {
	// PhysicalType returns the physical column type for the given canonical storage type.
	PhysicalType(storage StorageType) (string, error)
	// CanStore reports whether the dialect can persist the canonical storage type.
	CanStore(storage StorageType) bool
}

// SQLBuilder builds dialect-specific SQL statements from semantic operation descriptions.
type SQLBuilder interface {
	// Select builds a SELECT statement.
	Select(ctx context.Context, opts SelectOptions) (SQL, error)
	// Insert builds an INSERT statement.
	Insert(ctx context.Context, opts InsertOptions) (SQL, error)
	// Update builds an UPDATE statement.
	Update(ctx context.Context, opts UpdateOptions) (SQL, error)
	// Delete builds a DELETE statement.
	Delete(ctx context.Context, opts DeleteOptions) (SQL, error)
}

// SQL holds a generated SQL statement and its bound arguments.
type SQL struct {
	Statement string
	Args      []any
}

// SelectOptions describes a common SELECT operation.
type SelectOptions struct {
	Table      string
	Columns    []string
	Joins      []JoinClause
	Where      []Condition
	GroupBy    []string
	Having     []Condition
	OrderBy    []OrderClause
	Limit      *int
	Offset     *int
	Parameters map[string]any
	ForUpdate  bool
	ForShare   bool
}

// JoinClause describes a common JOIN.
type JoinClause struct {
	Kind  string // INNER, LEFT, RIGHT, FULL
	Table string
	On    []Condition
	Alias string
}

// Condition describes a WHERE/HAVING predicate.
type Condition struct {
	Column    string
	Operator  string // =, !=, <, >, <=, >=, IN, NOT IN, IS NULL, IS NOT NULL, LIKE
	Value     any
	Not       bool
	LogicalOp string // AND, OR
	// Expression holds a raw SQL fragment when the condition cannot be expressed
	// as a structured Column/Operator/Value triple.
	Expression string
}

// OrderClause describes an ORDER BY item.
type OrderClause struct {
	Column string
	Desc   bool
	Nulls  string // FIRST, LAST, "" for default
}

// InsertOptions describes a common INSERT operation.
type InsertOptions struct {
	Table     string
	Columns   []string
	Values    map[string]any
	Returning []string
}

// UpdateOptions describes a common UPDATE operation.
type UpdateOptions struct {
	Table       string
	Assignments map[string]any
	Where       []Condition
	Parameters  map[string]any
	Returning   []string
}

// DeleteOptions describes a common DELETE operation.
type DeleteOptions struct {
	Table      string
	Where      []Condition
	Parameters map[string]any
	Returning  []string
}

// Dialect is the common SQL dialect contract.
type Dialect interface {
	TypeMapper
	SQLBuilder

	// PlaceholderStyle returns the parameter binding style.
	PlaceholderStyle() PlaceholderStyle
	// Name returns the dialect identifier.
	Name() string
	// BeginTransaction starts a transaction on the given *sql.DB.
	BeginTransaction(ctx context.Context, db *sql.DB) (*sql.Tx, error)
	// IsUniqueViolation reports whether err is a unique constraint violation.
	IsUniqueViolation(err error) bool
	// IsForeignKeyViolation reports whether err is a foreign key violation.
	IsForeignKeyViolation(err error) bool
	// IsNullViolation reports whether err is a NOT NULL violation.
	IsNullViolation(err error) bool
	// IsCheckViolation reports whether err is a CHECK constraint violation.
	IsCheckViolation(err error) bool
}
