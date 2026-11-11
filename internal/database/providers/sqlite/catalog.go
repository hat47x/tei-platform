package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// TableMeta describes one table or view in a SQLite database.
type TableMeta struct {
	Schema string
	Name   string
	Kind   string
}

// ColumnMeta describes one column in a SQLite table.
type ColumnMeta struct {
	Schema       string
	Table        string
	Name         string
	PhysicalType string
	Nullable     bool
	Default      string
	Ordinal      int
}

// PrimaryKeyMeta describes the primary key of a SQLite table.
type PrimaryKeyMeta struct {
	Schema  string
	Table   string
	Columns []string
}

// UniqueConstraintMeta describes a unique constraint in SQLite.
type UniqueConstraintMeta struct {
	Schema  string
	Table   string
	Name    string
	Columns []string
}

// ForeignKeyMeta describes a foreign key in SQLite.
type ForeignKeyMeta struct {
	Schema            string
	Table             string
	Name              string
	Columns           []string
	ReferencedSchema  string
	ReferencedTable   string
	ReferencedColumns []string
}

// IndexKeyMeta describes one key column in a SQLite index.
type IndexKeyMeta struct {
	Column     string
	Collation  string
	Order      string
	NullsOrder string
}

// IndexMeta describes one index in SQLite.
type IndexMeta struct {
	Schema          string
	Table           string
	Name            string
	Keys            []IndexKeyMeta
	IncludedColumns []string
	Unique          bool
	Method          string
}

// CatalogReader isolates SQLite catalog access from TEI's provider model.
// Any database/sql.DB or sqlite3 driver connection can satisfy this interface.
type CatalogReader interface {
	Tables(context.Context) ([]TableMeta, error)
	Columns(context.Context) ([]ColumnMeta, error)
	PrimaryKeys(context.Context) ([]PrimaryKeyMeta, error)
	UniqueConstraints(context.Context) ([]UniqueConstraintMeta, error)
	ForeignKeys(context.Context) ([]ForeignKeyMeta, error)
	Indexes(context.Context) ([]IndexMeta, error)
}

// SQLCatalogReader implements CatalogReader using a database/sql.DB.
type SQLCatalogReader struct {
	DB *sql.DB
}

func (r SQLCatalogReader) Tables(ctx context.Context) ([]TableMeta, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT type, name
		FROM sqlite_master
		WHERE type IN ('table', 'view')
		ORDER BY type, name
	`)
	if err != nil {
		return nil, fmt.Errorf("sqlite tables: %w", err)
	}
	defer rows.Close()

	var out []TableMeta
	for rows.Next() {
		var kind, name string
		if err := rows.Scan(&kind, &name); err != nil {
			return nil, fmt.Errorf("sqlite tables scan: %w", err)
		}
		out = append(out, TableMeta{Schema: "main", Name: name, Kind: kind})
	}
	return out, rows.Err()
}

func (r SQLCatalogReader) Columns(ctx context.Context) ([]ColumnMeta, error) {
	tables, err := r.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var out []ColumnMeta
	for _, t := range tables {
		if t.Kind != "table" {
			continue
		}
		query := fmt.Sprintf(`PRAGMA table_info(%s)`, escapeIdentifier(t.Name))
		rows, err := r.DB.QueryContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("sqlite columns %s: %w", t.Name, err)
		}
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull int
			var dflt sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				rows.Close()
				return nil, fmt.Errorf("sqlite columns scan %s: %w", t.Name, err)
			}
			out = append(out, ColumnMeta{
				Schema:       t.Schema,
				Table:        t.Name,
				Name:         name,
				PhysicalType: typ,
				Nullable:     notnull == 0 && !dflt.Valid && pk == 0 && cid > 0,
				Default:      dflt.String,
				Ordinal:      cid,
			})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r SQLCatalogReader) PrimaryKeys(ctx context.Context) ([]PrimaryKeyMeta, error) {
	tables, err := r.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var out []PrimaryKeyMeta
	for _, t := range tables {
		if t.Kind != "table" {
			continue
		}
		query := fmt.Sprintf(`PRAGMA table_info(%s)`, escapeIdentifier(t.Name))
		rows, err := r.DB.QueryContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("sqlite primary keys %s: %w", t.Name, err)
		}
		var cols []string
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull int
			var dflt sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				rows.Close()
				return nil, fmt.Errorf("sqlite primary keys scan %s: %w", t.Name, err)
			}
			if pk > 0 {
				for len(cols) < pk {
					cols = append(cols, "")
				}
				cols[pk-1] = name
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		// Filter out empty slots (for non-contiguous PKs, though SQLite doesn't support that)
		var filtered []string
		for _, c := range cols {
			if c != "" {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) > 0 {
			out = append(out, PrimaryKeyMeta{Schema: t.Schema, Table: t.Name, Columns: filtered})
		}
	}
	return out, nil
}

func (r SQLCatalogReader) UniqueConstraints(ctx context.Context) ([]UniqueConstraintMeta, error) {
	tables, err := r.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var out []UniqueConstraintMeta
	for _, t := range tables {
		if t.Kind != "table" {
			continue
		}
		query := fmt.Sprintf(`PRAGMA index_list(%s)`, escapeIdentifier(t.Name))
		rows, err := r.DB.QueryContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("sqlite unique constraints %s: %w", t.Name, err)
		}
		var ucs []UniqueConstraintMeta
		for rows.Next() {
			var seq int
			var name, origin string
			var unique int
			var partial string
			if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
				rows.Close()
				return nil, fmt.Errorf("sqlite unique constraints scan %s: %w", t.Name, err)
			}
			_ = seq
			// 'u' = unique constraint created by UNIQUE constraint or CREATE UNIQUE INDEX
			// We include 'u' origin. SQLite also reports 'c' for auto-created unique indexes from UNIQUE constraints.
			if origin == "u" || (origin == "c" && unique == 1) {
				ucs = append(ucs, UniqueConstraintMeta{
					Schema: t.Schema,
					Table:  t.Name,
					Name:   name,
				})
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}

		for i := range ucs {
			keys, err := r.indexColumns(ctx, ucs[i].Schema, ucs[i].Table, ucs[i].Name)
			if err != nil {
				return nil, err
			}
			cols := make([]string, 0, len(keys))
			for _, k := range keys {
				cols = append(cols, k.Column)
			}
			ucs[i].Columns = cols
		}
		out = append(out, ucs...)
	}
	return out, nil
}

func (r SQLCatalogReader) ForeignKeys(ctx context.Context) ([]ForeignKeyMeta, error) {
	tables, err := r.Tables(ctx)
	if err != nil {
		return nil, err
	}

	type fkKey struct {
		schema string
		table  string
		id     int
	}
	type fkData struct {
		fromCols []string
		toCols   []string
		toTable  string
	}

	groups := make(map[fkKey]*fkData)

	for _, t := range tables {
		if t.Kind != "table" {
			continue
		}
		query := fmt.Sprintf(`PRAGMA foreign_key_list(%s)`, escapeIdentifier(t.Name))
		rows, err := r.DB.QueryContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("sqlite foreign keys %s: %w", t.Name, err)
		}
		for rows.Next() {
			var id, seq int
			var fromCol, toTable, toCol string
			var onUpdate, onDelete, match sql.NullString
			if err := rows.Scan(&id, &seq, &toTable, &fromCol, &toCol, &onUpdate, &onDelete, &match); err != nil {
				rows.Close()
				return nil, fmt.Errorf("sqlite foreign keys scan %s: %w", t.Name, err)
			}
			key := fkKey{schema: t.Schema, table: t.Name, id: id}
			data, ok := groups[key]
			if !ok {
				data = &fkData{}
				groups[key] = data
			}
			for len(data.fromCols) <= seq {
				data.fromCols = append(data.fromCols, "")
				data.toCols = append(data.toCols, "")
			}
			data.fromCols[seq] = fromCol
			data.toCols[seq] = toCol
			data.toTable = toTable
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	// Build the final result
	var out []ForeignKeyMeta
	for key, data := range groups {
		out = append(out, ForeignKeyMeta{
			Schema:            key.schema,
			Table:             key.table,
			Name:              fmt.Sprintf("fk_%s_%d", key.table, key.id),
			Columns:           data.fromCols,
			ReferencedSchema:  "",
			ReferencedTable:   data.toTable,
			ReferencedColumns: data.toCols,
		})
	}
	return out, nil
}

func (r SQLCatalogReader) Indexes(ctx context.Context) ([]IndexMeta, error) {
	tables, err := r.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var out []IndexMeta
	for _, t := range tables {
		if t.Kind != "table" {
			continue
		}
		query := fmt.Sprintf(`PRAGMA index_list(%s)`, escapeIdentifier(t.Name))
		rows, err := r.DB.QueryContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("sqlite indexes %s: %w", t.Name, err)
		}
		for rows.Next() {
			var seq int
			var name, origin string
			var unique int
			var partial string
			if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
				rows.Close()
				return nil, fmt.Errorf("sqlite indexes scan %s: %w", t.Name, err)
			}
			_ = seq
			// Skip auto-indexes for PRIMARY KEY (origin = 'pk') and internal indexes
			// We want user-defined indexes and unique constraints
			if origin == "pk" || origin == "fk" {
				continue
			}
			keys, err := r.indexColumns(ctx, t.Schema, t.Name, name)
			if err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, IndexMeta{
				Schema: t.Schema,
				Table:  t.Name,
				Name:   name,
				Keys:   keys,
				Unique: unique == 1,
				Method: "sqlite",
			})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r SQLCatalogReader) indexColumns(ctx context.Context, schema, table, indexName string) ([]IndexKeyMeta, error) {
	query := fmt.Sprintf(`PRAGMA %s.index_info('%s')`, escapeSchema(schema), escapeIdentifier(indexName))
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite index_info %s.%s.%s: %w", schema, table, indexName, err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var seqno, cid int
		var col string
		if err := rows.Scan(&seqno, &cid, &col); err != nil {
			return nil, fmt.Errorf("sqlite index_info scan: %w", err)
		}
		for len(cols) <= seqno {
			cols = append(cols, "")
		}
		cols[seqno] = col
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []IndexKeyMeta
	for _, col := range cols {
		if col == "" {
			continue
		}
		out = append(out, IndexKeyMeta{
			Column:     col,
			Collation:  "",
			Order:      "default",
			NullsOrder: "default",
		})
	}
	return out, nil
}

func escapeSchema(schema string) string {
	if schema == "" {
		return "main"
	}
	return strings.ReplaceAll(schema, "'", "''")
}

func escapeIdentifier(name string) string {
	return strings.ReplaceAll(name, "'", "''")
}
