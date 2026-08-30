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
	"tei/internal/database/persistence"
	dbprovider "tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

const databaseID = "reference_postgres"

type result struct {
	Database       string          `json:"database"`
	BaselineDiff   schemadiff.Diff `json:"baselineDiff"`
	DriftDiff      schemadiff.Diff `json:"driftDiff"`
	ActualTables   int             `json:"actualTables"`
	ForeignKeys    int             `json:"foreignKeys"`
	ConnectionData bool            `json:"connectionDataPersisted"`
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

	mappings := referenceMappings()
	desired, err := persistence.RealizeDesired(mappings, databaseID, environment.ProviderPostgres)
	if err != nil {
		panic(err)
	}
	instance := environment.DatabaseInstance{
		ID:         core.MustCanonicalID("reference.postgres.local"),
		Definition: core.MustCanonicalID(databaseID),
		Provider:   environment.ProviderPostgres,
	}
	p := postgres.Provider{
		Catalog: postgres.SQLCatalogReader{DB: db},
		Now:     func() time.Time { return time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC) },
	}
	actual, err := dbprovider.Introspect(ctx, p, instance)
	if err != nil {
		panic(err)
	}
	baseline, err := schemadiff.Compare(desired, actual)
	if err != nil {
		panic(err)
	}
	if len(baseline.Changes) != 0 {
		raw, _ := json.MarshalIndent(baseline, "", "  ")
		panic(fmt.Errorf("reference PostgreSQL baseline has schema drift:\n%s", raw))
	}

	if _, err := db.ExecContext(ctx, `ALTER TABLE tei_reference.orders ADD COLUMN drift_note text`); err != nil {
		panic(err)
	}
	drifted, err := dbprovider.Introspect(ctx, p, instance)
	if err != nil {
		panic(err)
	}
	drift, err := schemadiff.Compare(desired, drifted)
	if err != nil {
		panic(err)
	}
	if !hasDropColumn(drift, "tei_reference.orders", "drift_note") {
		raw, _ := json.MarshalIndent(drift, "", "  ")
		panic(fmt.Errorf("expected actual-only drift_note column in diff:\n%s", raw))
	}

	foreignKeys := 0
	for _, table := range actual.Tables {
		foreignKeys += len(table.ForeignKeys)
	}
	out := result{
		Database:       databaseID,
		BaselineDiff:   baseline,
		DriftDiff:      drift,
		ActualTables:   len(actual.Tables),
		ForeignKeys:    foreignKeys,
		ConnectionData: containsConnectionData(actual, drift, dsn),
	}
	if out.ConnectionData {
		panic("connection data leaked into physical snapshot or diff")
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}

func referenceMappings() []persistence.Mapping {
	return []persistence.Mapping{
		{
			ID:           core.MustCanonicalID("reference_postgres.customers"),
			Database:     core.MustCanonicalID(databaseID),
			SemanticData: core.MustCanonicalID("reference.customer"),
			Table:        "tei_reference.customers",
			Columns: []persistence.ColumnMapping{
				{Attribute: core.MustCanonicalID("reference.customer.id"), Column: "id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
				{Attribute: core.MustCanonicalID("reference.customer.name"), Column: "name", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
				{Attribute: core.MustCanonicalID("reference.customer.profile"), Column: "profile", Storage: persistence.StorageJSON, Nullability: persistence.Optional},
			},
			PrimaryKey: []core.CanonicalID{core.MustCanonicalID("reference.customer.id")},
			Indexes: []persistence.IndexMapping{{Name: "customers_name_idx", Attributes: []core.CanonicalID{core.MustCanonicalID("reference.customer.name")}}},
		},
		{
			ID:           core.MustCanonicalID("reference_postgres.orders"),
			Database:     core.MustCanonicalID(databaseID),
			SemanticData: core.MustCanonicalID("reference.order"),
			Table:        "tei_reference.orders",
			Columns: []persistence.ColumnMapping{
				{Attribute: core.MustCanonicalID("reference.order.id"), Column: "id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
				{Attribute: core.MustCanonicalID("reference.order.customer_id"), Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
				{Attribute: core.MustCanonicalID("reference.order.note"), Column: "note", Storage: persistence.StorageText, Nullability: persistence.Optional},
				{Attribute: core.MustCanonicalID("reference.order.created_at"), Column: "created_at", Storage: persistence.StorageDateTimeTZ, Nullability: persistence.Required},
				{Attribute: core.MustCanonicalID("reference.order.total"), Column: "total", Storage: persistence.StorageDecimal, Nullability: persistence.Required},
			},
			PrimaryKey: []core.CanonicalID{core.MustCanonicalID("reference.order.id")},
			Indexes: []persistence.IndexMapping{{Name: "orders_customer_id_idx", Attributes: []core.CanonicalID{core.MustCanonicalID("reference.order.customer_id")}}},
		},
	}
}

func prepareFixture(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
DROP SCHEMA IF EXISTS tei_reference CASCADE;
CREATE SCHEMA tei_reference;
CREATE TABLE tei_reference.customers (
    id bigint NOT NULL,
    name varchar NOT NULL,
    profile jsonb,
    CONSTRAINT customers_pkey PRIMARY KEY (id)
);
CREATE INDEX customers_name_idx ON tei_reference.customers USING btree (name);
CREATE TABLE tei_reference.orders (
    id bigint NOT NULL,
    customer_id bigint NOT NULL,
    note text,
    created_at timestamptz NOT NULL,
    total numeric NOT NULL,
    CONSTRAINT orders_pkey PRIMARY KEY (id),
    CONSTRAINT orders_customer_fk FOREIGN KEY (customer_id)
        REFERENCES tei_reference.customers(id)
);
CREATE INDEX orders_customer_id_idx ON tei_reference.orders USING btree (customer_id);
`)
	return err
}

func hasDropColumn(diff schemadiff.Diff, table, column string) bool {
	for _, c := range diff.Changes {
		if c.Kind == schemadiff.DropColumn && c.Table == table && c.BeforeColumn != nil && c.BeforeColumn.Name == column {
			return true
		}
	}
	return false
}

func containsConnectionData(actual any, diff schemadiff.Diff, dsn string) bool {
	actualJSON, _ := json.Marshal(actual)
	diffJSON, _ := json.Marshal(diff)
	return dsn != "" && (string(actualJSON) == dsn || string(diffJSON) == dsn || contains(string(actualJSON), dsn) || contains(string(diffJSON), dsn))
}

func contains(haystack, needle string) bool {
	if needle == "" || len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
