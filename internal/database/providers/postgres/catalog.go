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

type ForeignKeyMeta struct {
	Schema            string
	Table             string
	Name              string
	Columns           []string
	ReferencedSchema  string
	ReferencedTable   string
	ReferencedColumns []string
}

type IndexMeta struct {
	Schema  string
	Table   string
	Name    string
	Columns []string
	Unique  bool
	Method  string
}

// CatalogReader isolates PostgreSQL catalog access from TEI's provider model.
// A database/sql, pgx or test implementation can satisfy this interface.
type CatalogReader interface {
	Tables(context.Context) ([]TableMeta, error)
	Columns(context.Context) ([]ColumnMeta, error)
	PrimaryKeys(context.Context) ([]PrimaryKeyMeta, error)
	ForeignKeys(context.Context) ([]ForeignKeyMeta, error)
	Indexes(context.Context) ([]IndexMeta, error)
}
