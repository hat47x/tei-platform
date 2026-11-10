package sqlite

import (
	"context"
	"fmt"
	"time"

	"tei/internal/database/physical"
	"tei/internal/database/provider"
	"tei/internal/environment"
)

// Provider implements the database/provider.Provider interface for SQLite.
type Provider struct {
	Catalog CatalogReader
	Now     func() time.Time
}

func (Provider) ID() environment.DatabaseProvider { return environment.ProviderSQLite }

func (Provider) Capabilities(context.Context, environment.DatabaseInstance) (provider.Capabilities, error) {
	return provider.Capabilities{
		provider.CapabilityMetadataRead: true,
		provider.CapabilityQuery:        true,
	}, nil
}

func (p Provider) Introspect(ctx context.Context, instance environment.DatabaseInstance) (physical.SchemaSnapshot, error) {
	if p.Catalog == nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("sqlite catalog reader is nil")
	}
	tables, err := p.Catalog.Tables(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("sqlite tables: %w", err)
	}
	columns, err := p.Catalog.Columns(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("sqlite columns: %w", err)
	}
	pks, err := p.Catalog.PrimaryKeys(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("sqlite primary keys: %w", err)
	}
	uniqueConstraints, err := p.Catalog.UniqueConstraints(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("sqlite unique constraints: %w", err)
	}
	fks, err := p.Catalog.ForeignKeys(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("sqlite foreign keys: %w", err)
	}
	indexes, err := p.Catalog.Indexes(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("sqlite indexes: %w", err)
	}

	byName := map[string]int{}
	snapshot := physical.SchemaSnapshot{Database: instance.Definition}
	for _, meta := range tables {
		snapshot.Tables = append(snapshot.Tables, physical.Table{Schema: meta.Schema, Name: meta.Name, Kind: physical.TableKind(meta.Kind)})
		byName[meta.Schema+"."+meta.Name] = len(snapshot.Tables) - 1
	}
	for _, meta := range columns {
		i, ok := byName[meta.Schema+"."+meta.Table]
		if !ok {
			continue
		}
		t := &snapshot.Tables[i]
		t.Columns = append(t.Columns, physical.Column{Name: meta.Name, PhysicalType: meta.PhysicalType, Nullable: meta.Nullable, Default: meta.Default, Ordinal: meta.Ordinal})
	}
	for _, meta := range pks {
		if i, ok := byName[meta.Schema+"."+meta.Table]; ok {
			snapshot.Tables[i].PrimaryKey = append([]string(nil), meta.Columns...)
		}
	}
	for _, meta := range uniqueConstraints {
		if i, ok := byName[meta.Schema+"."+meta.Table]; ok {
			t := &snapshot.Tables[i]
			t.UniqueConstraints = append(t.UniqueConstraints, physical.UniqueConstraint{Name: meta.Name, Columns: append([]string(nil), meta.Columns...)})
		}
	}
	for _, meta := range fks {
		if i, ok := byName[meta.Schema+"."+meta.Table]; ok {
			t := &snapshot.Tables[i]
			t.ForeignKeys = append(t.ForeignKeys, physical.ForeignKey{Name: meta.Name, Columns: append([]string(nil), meta.Columns...), ReferencedSchema: meta.ReferencedSchema, ReferencedTable: meta.ReferencedTable, ReferencedColumns: append([]string(nil), meta.ReferencedColumns...)})
		}
	}
	for _, meta := range indexes {
		if i, ok := byName[meta.Schema+"."+meta.Table]; ok {
			t := &snapshot.Tables[i]
			t.Indexes = append(t.Indexes, physical.Index{
				Name:   meta.Name,
				Unique: meta.Unique,
				Method: meta.Method,
			})
		}
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	snapshot.CapturedAt = now().UTC().Format(time.RFC3339)
	return snapshot.Normalize(), nil
}
