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
	"tei/internal/database/physical"
	dbprovider "tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

const databaseID = "reference_postgres"

type result struct {
	Database                        string          `json:"database"`
	BaselineDiff                    schemadiff.Diff `json:"baselineDiff"`
	DriftDiff                       schemadiff.Diff `json:"driftDiff"`
	ActualTables                    int             `json:"actualTables"`
	UniqueConstraints               int             `json:"uniqueConstraints"`
	UniqueConstraintDrift           bool            `json:"uniqueConstraintDriftDetected"`
	UniqueBackingIndexExcluded      bool            `json:"uniqueBackingIndexExcluded"`
	ExplicitUniqueIndexObserved     bool            `json:"explicitUniqueIndexObserved"`
	ForeignKeys                     int             `json:"foreignKeys"`
	ForeignKeyDrift                 bool            `json:"foreignKeyDriftDetected"`
	ViewColumnsObserved             bool            `json:"viewColumnsObserved"`
	MaterializedViewColumnsObserved bool            `json:"materializedViewColumnsObserved"`
	ConnectionData                  bool            `json:"connectionDataPersisted"`
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

	observedRelations, err := dbprovider.Introspect(ctx, p, instance)
	if err != nil {
		panic(err)
	}
	viewColumnsObserved := hasColumns(observedRelations, "tei_reference.customer_names_v", physical.TableKindView, "id", "name")
	materializedViewColumnsObserved := hasColumns(observedRelations, "tei_reference.customer_names_mv", physical.TableKindMaterializedView, "id", "name")
	if !viewColumnsObserved || !materializedViewColumnsObserved {
		raw, _ := json.MarshalIndent(observedRelations, "", "  ")
		panic(fmt.Errorf("reference PostgreSQL did not expose expected view/materialized-view columns:\n%s", raw))
	}
	if _, err := db.ExecContext(ctx, `
DROP VIEW tei_reference.customer_names_v;
DROP MATERIALIZED VIEW tei_reference.customer_names_mv;
`); err != nil {
		panic(err)
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
	if !hasUniqueConstraint(actual, "tei_reference.customers", "external_id") {
		raw, _ := json.MarshalIndent(actual, "", "  ")
		panic(fmt.Errorf("reference PostgreSQL did not expose customers UNIQUE constraint:\n%s", raw))
	}
	uniqueBackingIndexExcluded := !hasIndexNamed(actual, "tei_reference.customers", "customers_external_id_key")
	if !uniqueBackingIndexExcluded {
		raw, _ := json.MarshalIndent(actual, "", "  ")
		panic(fmt.Errorf("UNIQUE constraint backing index was double-counted as a normal index:\n%s", raw))
	}
	explicitUniqueIndexObserved := hasIndex(actual, "tei_reference.customers", "customers_name_uidx", true, "name")
	if !explicitUniqueIndexObserved {
		raw, _ := json.MarshalIndent(actual, "", "  ")
		panic(fmt.Errorf("explicit unique index was not preserved as an index:\n%s", raw))
	}

	if _, err := db.ExecContext(ctx, `
ALTER TABLE tei_reference.customers DROP CONSTRAINT customers_external_id_key;
ALTER TABLE tei_reference.orders ADD COLUMN drift_note text;
ALTER TABLE tei_reference.orders DROP CONSTRAINT orders_customer_fk;
`); err != nil {
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
	if !hasAddForeignKey(drift, "tei_reference.orders", "customer_id", "tei_reference", "customers", "id") {
		raw, _ := json.MarshalIndent(drift, "", "  ")
		panic(fmt.Errorf("expected missing orders_customer_fk in diff:\n%s", raw))
	}
	if !hasAddUniqueConstraint(drift, "tei_reference.customers", "external_id") {
		raw, _ := json.MarshalIndent(drift, "", "  ")
		panic(fmt.Errorf("expected missing customers UNIQUE constraint in diff:\n%s", raw))
	}

	uniqueConstraints := 0
	foreignKeys := 0
	for _, table := range actual.Tables {
		uniqueConstraints += len(table.UniqueConstraints)
		foreignKeys += len(table.ForeignKeys)
	}
	out := result{
		Database:                        databaseID,
		BaselineDiff:                    baseline,
		DriftDiff:                       drift,
		ActualTables:                    len(actual.Tables),
		UniqueConstraints:               uniqueConstraints,
		UniqueConstraintDrift:           true,
		UniqueBackingIndexExcluded:      uniqueBackingIndexExcluded,
		ExplicitUniqueIndexObserved:     explicitUniqueIndexObserved,
		ForeignKeys:                     foreignKeys,
		ForeignKeyDrift:                 true,
		ViewColumnsObserved:             viewColumnsObserved,
		MaterializedViewColumnsObserved: materializedViewColumnsObserved,
		ConnectionData:                  containsConnectionData(actual, drift, dsn),
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
				{Attribute: core.MustCanonicalID("reference.customer.external_id"), Column: "external_id", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
				{Attribute: core.MustCanonicalID("reference.customer.profile"), Column: "profile", Storage: persistence.StorageJSON, Nullability: persistence.Optional},
			},
			PrimaryKey: []core.CanonicalID{core.MustCanonicalID("reference.customer.id")},
			UniqueConstraints: []persistence.UniqueConstraintMapping{{
				Name:       "customers_external_id_key",
				Attributes: []core.CanonicalID{core.MustCanonicalID("reference.customer.external_id")},
			}},
			Indexes: []persistence.IndexMapping{{
				Name:       "customers_name_uidx",
				Attributes: []core.CanonicalID{core.MustCanonicalID("reference.customer.name")},
				Unique:     true,
			}},
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
			Indexes:    []persistence.IndexMapping{{Name: "orders_customer_id_idx", Attributes: []core.CanonicalID{core.MustCanonicalID("reference.order.customer_id")}}},
			ForeignKeys: []persistence.ForeignKeyMapping{{
				Name:                 "orders_customer_fk",
				Attributes:           []core.CanonicalID{core.MustCanonicalID("reference.order.customer_id")},
				ReferencedMapping:    core.MustCanonicalID("reference_postgres.customers"),
				ReferencedAttributes: []core.CanonicalID{core.MustCanonicalID("reference.customer.id")},
			}},
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
    external_id varchar NOT NULL,
    profile jsonb,
    CONSTRAINT customers_pkey PRIMARY KEY (id),
    CONSTRAINT customers_external_id_key UNIQUE (external_id)
);
CREATE UNIQUE INDEX customers_name_uidx ON tei_reference.customers USING btree (name);
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
CREATE VIEW tei_reference.customer_names_v AS
    SELECT id, name FROM tei_reference.customers;
CREATE MATERIALIZED VIEW tei_reference.customer_names_mv AS
    SELECT id, name FROM tei_reference.customers;
`)
	return err
}

func hasColumns(snapshot physical.SchemaSnapshot, table string, kind physical.TableKind, names ...string) bool {
	t, ok := snapshot.Table(table)
	if !ok || t.Kind != kind || len(t.Columns) != len(names) {
		return false
	}
	for i, name := range names {
		if t.Columns[i].Name != name {
			return false
		}
	}
	return true
}

func hasUniqueConstraint(snapshot physical.SchemaSnapshot, table string, columns ...string) bool {
	t, ok := snapshot.Table(table)
	if !ok {
		return false
	}
	for _, constraint := range t.UniqueConstraints {
		if sameStrings(constraint.Columns, columns) {
			return true
		}
	}
	return false
}

func hasIndexNamed(snapshot physical.SchemaSnapshot, table, name string) bool {
	t, ok := snapshot.Table(table)
	if !ok {
		return false
	}
	for _, index := range t.Indexes {
		if index.Name == name {
			return true
		}
	}
	return false
}

func hasIndex(snapshot physical.SchemaSnapshot, table, name string, unique bool, columns ...string) bool {
	t, ok := snapshot.Table(table)
	if !ok {
		return false
	}
	for _, index := range t.Indexes {
		if index.Name == name && index.Unique == unique && sameStrings(index.Columns, columns) {
			return true
		}
	}
	return false
}

func hasDropColumn(diff schemadiff.Diff, table, column string) bool {
	for _, c := range diff.Changes {
		if c.Kind == schemadiff.DropColumn && c.Table == table && c.BeforeColumn != nil && c.BeforeColumn.Name == column {
			return true
		}
	}
	return false
}

func hasAddUniqueConstraint(diff schemadiff.Diff, table string, columns ...string) bool {
	for _, c := range diff.Changes {
		if c.Kind == schemadiff.AddUniqueConstraint && c.Table == table && c.UniqueConstraint != nil && sameStrings(c.UniqueConstraint.Columns, columns) {
			return true
		}
	}
	return false
}

func hasAddForeignKey(diff schemadiff.Diff, table, column, refSchema, refTable, refColumn string) bool {
	for _, c := range diff.Changes {
		if c.Kind != schemadiff.AddForeignKey || c.Table != table || c.ForeignKey == nil {
			continue
		}
		fk := c.ForeignKey
		if len(fk.Columns) == 1 && fk.Columns[0] == column && fk.ReferencedSchema == refSchema && fk.ReferencedTable == refTable && len(fk.ReferencedColumns) == 1 && fk.ReferencedColumns[0] == refColumn {
			return true
		}
	}
	return false
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
