package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tei/internal/database/physical"
	"tei/internal/database/plan"
	dbprovider "tei/internal/database/provider"
	"tei/internal/environment"
	"tei/internal/query"
)

type Provider struct {
	Catalog    CatalogReader
	PlanReader ExplainReader
	Now        func() time.Time
}

func (Provider) ID() environment.DatabaseProvider { return environment.ProviderPostgres }

func (Provider) Capabilities(context.Context, environment.DatabaseInstance) (dbprovider.Capabilities, error) {
	return dbprovider.Capabilities{
		dbprovider.CapabilityMetadataRead:   true,
		dbprovider.CapabilityQuery:          true,
		dbprovider.CapabilityDataMutation:   true,
		dbprovider.CapabilitySchemaChange:   true,
		dbprovider.CapabilityExplain:        true,
		dbprovider.CapabilityExplainAnalyze: true,
		dbprovider.CapabilityTransaction:    true,
		dbprovider.CapabilityCancel:         true,
	}, nil
}

func (p Provider) Introspect(ctx context.Context, instance environment.DatabaseInstance) (physical.SchemaSnapshot, error) {
	if p.Catalog == nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("postgres catalog reader is nil")
	}
	tables, err := p.Catalog.Tables(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("postgres tables: %w", err)
	}
	columns, err := p.Catalog.Columns(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("postgres columns: %w", err)
	}
	pks, err := p.Catalog.PrimaryKeys(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("postgres primary keys: %w", err)
	}
	fks, err := p.Catalog.ForeignKeys(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("postgres foreign keys: %w", err)
	}
	indexes, err := p.Catalog.Indexes(ctx)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("postgres indexes: %w", err)
	}

	byName := map[string]int{}
	snapshot := physical.SchemaSnapshot{Database: instance.Definition}
	for _, meta := range tables {
		kind := physical.TableKindTable
		switch strings.ToLower(meta.Kind) {
		case "view":
			kind = physical.TableKindView
		case "materialized view", "materialized-view":
			kind = physical.TableKindMaterializedView
		}
		snapshot.Tables = append(snapshot.Tables, physical.Table{Schema: meta.Schema, Name: meta.Name, Kind: kind})
		byName[key(meta.Schema, meta.Name)] = len(snapshot.Tables) - 1
	}
	for _, meta := range columns {
		i, ok := byName[key(meta.Schema, meta.Table)]
		if !ok {
			continue
		}
		t := &snapshot.Tables[i]
		t.Columns = append(t.Columns, physical.Column{Name: meta.Name, PhysicalType: meta.PhysicalType, Nullable: meta.Nullable, Default: meta.Default, Ordinal: meta.Ordinal})
	}
	for _, meta := range pks {
		if i, ok := byName[key(meta.Schema, meta.Table)]; ok {
			snapshot.Tables[i].PrimaryKey = append([]string(nil), meta.Columns...)
		}
	}
	for _, meta := range fks {
		if i, ok := byName[key(meta.Schema, meta.Table)]; ok {
			t := &snapshot.Tables[i]
			t.ForeignKeys = append(t.ForeignKeys, physical.ForeignKey{Name: meta.Name, Columns: append([]string(nil), meta.Columns...), ReferencedSchema: meta.ReferencedSchema, ReferencedTable: meta.ReferencedTable, ReferencedColumns: append([]string(nil), meta.ReferencedColumns...)})
		}
	}
	for _, meta := range indexes {
		if i, ok := byName[key(meta.Schema, meta.Table)]; ok {
			t := &snapshot.Tables[i]
			t.Indexes = append(t.Indexes, physical.Index{Name: meta.Name, Columns: append([]string(nil), meta.Columns...), Unique: meta.Unique, Method: meta.Method})
		}
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	snapshot.CapturedAt = now().UTC().Format(time.RFC3339)
	return snapshot.Normalize(), nil
}

func (p Provider) Explain(ctx context.Context, instance environment.DatabaseInstance, q query.EmbeddedSQL, snapshot *physical.SchemaSnapshot, opts dbprovider.ExplainOptions) (plan.IR, error) {
	if p.PlanReader == nil {
		return plan.IR{}, fmt.Errorf("postgres explain reader is nil")
	}
	raw, err := p.PlanReader.ExplainJSON(ctx, ExplainRequest{SQL: q.Source, Analyze: opts.Analyze, Buffers: opts.Buffers})
	if err != nil {
		return plan.IR{}, fmt.Errorf("postgres explain: %w", err)
	}
	return NormalizeExplainJSON(q.ID, instance.Definition, raw, snapshot)
}

func key(schema, table string) string {
	return strings.ToLower(strings.TrimSpace(schema)) + "." + strings.ToLower(strings.TrimSpace(table))
}
