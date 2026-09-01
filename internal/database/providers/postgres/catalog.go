package postgres

import "context"

type TableMeta struct {
	Schema string
	Name   string
	Kind   string
}

type ColumnMeta struct {
	Schema       string
	Table        string
	Name         string
	PhysicalType string
	Nullable     bool
	Default      string
	Ordinal      int
}

type PrimaryKeyMeta struct {
	Schema  string
	Table   string
	Columns []string
}

type UniqueConstraintMeta struct {
	Schema  string
	Table   string
	Name    string
	Columns []string
}

type ForeignKeyMeta struct {
	Schema            string
	Table             string
	Name              string
	Columns           []string
	ReferencedSchema  string
	ReferencedTable   string
	ReferencedColumns []string
}

type IndexKeyMeta struct {
	Column        string
	Expression    string
	Collation     string
	OperatorClass string
	Descending    bool
	NullsFirst    bool
}

type IndexMeta struct {
	Schema           string
	Table            string
	Name             string
	Keys             []IndexKeyMeta
	IncludedColumns  []string
	Predicate        string
	Unique           bool
	NullsNotDistinct bool
	Method           string
	Valid            bool
	Ready            bool
	Live             bool
}

// CatalogReader isolates PostgreSQL catalog access from TEI's provider model.
// A database/sql, pgx or test implementation can satisfy this interface.
type CatalogReader interface {
	Tables(context.Context) ([]TableMeta, error)
	Columns(context.Context) ([]ColumnMeta, error)
	PrimaryKeys(context.Context) ([]PrimaryKeyMeta, error)
	UniqueConstraints(context.Context) ([]UniqueConstraintMeta, error)
	ForeignKeys(context.Context) ([]ForeignKeyMeta, error)
	Indexes(context.Context) ([]IndexMeta, error)
}
