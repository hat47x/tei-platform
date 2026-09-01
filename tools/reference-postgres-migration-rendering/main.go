package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	dbprovider "tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

const databaseID = "migration_render_postgres"

type result struct {
	RenderedStatements int  `json:"renderedStatements"`
	AdvancedIndexes    int  `json:"advancedIndexes"`
	Converged          bool `json:"converged"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dsn := os.Getenv("TEI_TEST_POSTGRES_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "TEI_TEST_POSTGRES_DSN is required")
		os.Exit(2)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}
	if err := prepareFixture(ctx, db); err != nil {
		panic(err)
	}

	instance := environment.DatabaseInstance{
		ID:         core.MustCanonicalID("migration_render.postgres.local"),
		Definition: core.MustCanonicalID(databaseID),
		Provider:   environment.ProviderPostgres,
	}
	provider := postgres.Provider{
		Catalog: postgres.SQLCatalogReader{DB: db},
		Now:     func() time.Time { return time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC) },
	}

	actualBefore, err := dbprovider.Introspect(ctx, provider, instance)
	if err != nil {
		panic(err)
	}
	desired := desiredSnapshot()
	diff, err := schemadiff.Compare(desired, actualBefore)
	if err != nil {
		panic(err)
	}
	if len(diff.Unresolved) != 0 || len(diff.Changes) != 3 {
		raw, _ := json.MarshalIndent(diff, "", "  ")
		panic(fmt.Errorf("expected three resolved add-index changes before rendering:\n%s", raw))
	}
	for _, change := range diff.Changes {
		if change.Kind != schemadiff.AddIndex {
			raw, _ := json.MarshalIndent(diff, "", "  ")
			panic(fmt.Errorf("expected only add-index changes before rendering:\n%s", raw))
		}
	}

	statements, err := migration.Render(environment.ProviderPostgres, diff)
	if err != nil {
		panic(err)
	}
	if len(statements) != 3 {
		panic(fmt.Errorf("expected three rendered statements, got %d", len(statements)))
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			panic(fmt.Errorf("execute rendered migration SQL %q: %w", statement, err))
		}
	}

	actualAfter, err := dbprovider.Introspect(ctx, provider, instance)
	if err != nil {
		panic(err)
	}
	converged, err := schemadiff.Compare(desired, actualAfter)
	if err != nil {
		panic(err)
	}
	if len(converged.Changes) != 0 || len(converged.Unresolved) != 0 {
		raw, _ := json.MarshalIndent(converged, "", "  ")
		panic(fmt.Errorf("rendered Advanced Index DDL did not converge after introspection:\n%s", raw))
	}

	table, ok := actualAfter.Table("tei_migration_render.customers")
	if !ok || len(table.Indexes) != 3 {
		raw, _ := json.MarshalIndent(actualAfter, "", "  ")
		panic(fmt.Errorf("expected three observed Advanced Indexes:\n%s", raw))
	}

	out := result{RenderedStatements: len(statements), AdvancedIndexes: len(table.Indexes), Converged: true}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}

func prepareFixture(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
DROP SCHEMA IF EXISTS tei_reference CASCADE;
DROP SCHEMA IF EXISTS tei_migration_render CASCADE;
CREATE SCHEMA tei_migration_render;
CREATE TABLE tei_migration_render.customers (
    id bigint NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    external_id text NOT NULL,
    profile jsonb
);`)
	return err
}

func desiredSnapshot() physical.SchemaSnapshot {
	return physical.SchemaSnapshot{
		Database: core.MustCanonicalID(databaseID),
		Tables: []physical.Table{{
			Schema: "tei_migration_render",
			Name:   "customers",
			Kind:   physical.TableKindTable,
			Columns: []physical.Column{
				{Name: "id", PhysicalType: "pg_catalog.int8", Nullable: false, Ordinal: 1},
				{Name: "name", PhysicalType: "pg_catalog.text", Nullable: false, Ordinal: 2},
				{Name: "email", PhysicalType: "pg_catalog.text", Nullable: false, Ordinal: 3},
				{Name: "external_id", PhysicalType: "pg_catalog.text", Nullable: false, Ordinal: 4},
				{Name: "profile", PhysicalType: "pg_catalog.jsonb", Nullable: true, Ordinal: 5},
			},
			Indexes: []physical.Index{
				{
					Name:   "customers_name_advanced_idx",
					Method: "btree",
					Keys: []physical.IndexKey{{
						Column:        "name",
						Collation:     "pg_catalog.C",
						OperatorClass: "pg_catalog.text_pattern_ops",
						Order:         physical.IndexOrderDesc,
						NullsOrder:    physical.IndexNullsOrderLast,
					}},
					IncludedColumns: []string{"external_id"},
					Predicate:       authoredExpression("profile IS NOT NULL"),
				},
				{
					Name:   "customers_lower_email_expr_idx",
					Method: "btree",
					Keys: []physical.IndexKey{{
						Expression: authoredExpression("lower(email)"),
						Order:      physical.IndexOrderDefault,
						NullsOrder: physical.IndexNullsOrderDefault,
					}},
				},
				{
					Name:                "customers_name_nulls_nd_uidx",
					Method:              "btree",
					Unique:              true,
					UniqueNullTreatment: physical.UniqueNullTreatmentNotDistinct,
					Keys: []physical.IndexKey{{
						Column:     "name",
						Order:      physical.IndexOrderDefault,
						NullsOrder: physical.IndexNullsOrderDefault,
					}},
				},
			},
		}},
	}.Normalize()
}

func authoredExpression(source string) *physical.ProviderExpression {
	return &physical.ProviderExpression{
		Provider:       string(environment.ProviderPostgres),
		Dialect:        "postgresql",
		Representation: physical.ProviderExpressionAuthored,
		Source:         source,
	}
}
