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
	"tei/internal/database/realization"
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
	AdvancedPartialIndexObserved    bool            `json:"advancedPartialIndexObserved"`
	ExpressionIndexObserved         bool            `json:"expressionIndexObserved"`
	IncludeIndexObserved            bool            `json:"includeIndexObserved"`
	PerKeyOptionsObserved           bool            `json:"perKeyOptionsObserved"`
	UniqueNullTreatmentObserved     bool            `json:"uniqueNullTreatmentObserved"`
	InvalidNonReadyIndexObserved    bool            `json:"invalidNonReadyIndexObserved"`
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
	if err := createFailedConcurrentIndex(ctx, db); err != nil {
		panic(err)
	}

	mappings := referenceMappings()
	desired, err := realization.RealizeDesired(mappings, databaseID, environment.ProviderPostgres)
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
		Now:     func() time.Time { return time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC) },
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

	advancedPartialIndexObserved, includeIndexObserved, perKeyOptionsObserved := inspectAdvancedColumnIndex(observedRelations)
	expressionIndexObserved := inspectExpressionIndex(observedRelations)
	uniqueNullTreatmentObserved := inspectUniqueNullTreatment(observedRelations)
	invalidNonReadyIndexObserved := inspectInvalidNonReadyIndex(observedRelations)
	if !advancedPartialIndexObserved || !includeIndexObserved || !perKeyOptionsObserved || !expressionIndexObserved || !uniqueNullTreatmentObserved || !invalidNonReadyIndexObserved {
		raw, _ := json.MarshalIndent(observedRelations, "", "  ")
		panic(fmt.Errorf("reference PostgreSQL did not expose expected advanced index semantics:\n%s", raw))
	}

	if _, err := db.ExecContext(ctx, `
DROP VIEW tei_reference.customer_names_v;
DROP MATERIALIZED VIEW tei_reference.customer_names_mv;
DROP INDEX tei_reference.customers_name_advanced_idx;
DROP INDEX tei_reference.customers_lower_name_expr_idx;
DROP INDEX tei_reference.customers_name_nulls_nd_uidx;
DROP TABLE tei_reference.index_state_probe;
DROP FUNCTION tei_reference.fail_on_zero(bigint);
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
	explicitUniqueIndexObserved := hasSimpleIndex(actual, "tei_reference.customers", "customers_name_uidx", true, "name")
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
		AdvancedPartialIndexObserved:    advancedPartialIndexObserved,
		ExpressionIndexObserved:         expressionIndexObserved,
		IncludeIndexObserved:            includeIndexObserved,
		PerKeyOptionsObserved:           perKeyOptionsObserved,
		UniqueNullTreatmentObserved:     uniqueNullTreatmentObserved,
		InvalidNonReadyIndexObserved:    invalidNonReadyIndexObserved,
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
CREATE INDEX customers_name_advanced_idx
    ON tei_reference.customers USING btree
    (name COLLATE "C" varchar_pattern_ops DESC NULLS LAST)
    INCLUDE (external_id)
    WHERE profile IS NOT NULL;
CREATE INDEX customers_lower_name_expr_idx
    ON tei_reference.customers USING btree (lower(name));
CREATE UNIQUE INDEX customers_name_nulls_nd_uidx
    ON tei_reference.customers USING btree (name) NULLS NOT DISTINCT;
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
CREATE FUNCTION tei_reference.fail_on_zero(value bigint)
RETURNS bigint
LANGUAGE SQL
IMMUTABLE
AS $$ SELECT 1 / value $$;
CREATE TABLE tei_reference.index_state_probe (value bigint NOT NULL);
INSERT INTO tei_reference.index_state_probe(value) VALUES (0);
`)
	return err
}

func createFailedConcurrentIndex(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE INDEX CONCURRENTLY index_state_probe_idx
ON tei_reference.index_state_probe (tei_reference.fail_on_zero(value))`)
	if err == nil {
		return fmt.Errorf("expected CREATE INDEX CONCURRENTLY state probe to fail")
	}
	return nil
}

func inspectAdvancedColumnIndex(snapshot physical.SchemaSnapshot) (partial, include, options bool) {
	index, ok := findIndex(snapshot, "tei_reference.customers", "customers_name_advanced_idx")
	if !ok || len(index.Keys) != 1 {
		return false, false, false
	}
	key := index.Keys[0]
	partial = index.Predicate != nil && index.Predicate.Provider == string(environment.ProviderPostgres) && index.Predicate.Dialect == "postgresql"
	include = len(index.IncludedColumns) == 1 && index.IncludedColumns[0] == "external_id"
	options = key.Column == "name" && key.Collation == "pg_catalog.C" && key.OperatorClass == "pg_catalog.varchar_pattern_ops" && key.Order == physical.IndexOrderDesc && key.NullsOrder == physical.IndexNullsOrderLast
	return partial, include, options
}

func inspectExpressionIndex(snapshot physical.SchemaSnapshot) bool {
	index, ok := findIndex(snapshot, "tei_reference.customers", "customers_lower_name_expr_idx")
	if !ok || len(index.Keys) != 1 || index.Keys[0].Column != "" || index.Keys[0].Expression == nil {
		return false
	}
	expr := index.Keys[0].Expression
	return expr.Provider == string(environment.ProviderPostgres) && expr.Dialect == "postgresql" && expr.Source != ""
}

func inspectUniqueNullTreatment(snapshot physical.SchemaSnapshot) bool {
	index, ok := findIndex(snapshot, "tei_reference.customers", "customers_name_nulls_nd_uidx")
	return ok && index.Unique && index.UniqueNullTreatment == physical.UniqueNullTreatmentNotDistinct
}

func inspectInvalidNonReadyIndex(snapshot physical.SchemaSnapshot) bool {
	index, ok := findIndex(snapshot, "tei_reference.index_state_probe", "index_state_probe_idx")
	return ok && index.State != nil && !index.State.Valid && !index.State.Ready
}

func findIndex(snapshot physical.SchemaSnapshot, table, name string) (physical.Index, bool) {
	t, ok := snapshot.Table(table)
	if !ok {
		return physical.Index{}, false
	}
	for _, index := range t.Indexes {
		if index.Name == name {
			return index, true
		}
	}
	return physical.Index{}, false
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
	_, ok := findIndex(snapshot, table, name)
	return ok
}

func hasSimpleIndex(snapshot physical.SchemaSnapshot, table, name string, unique bool, columns ...string) bool {
	index, ok := findIndex(snapshot, table, name)
	if !ok || index.Unique != unique || len(index.Keys) != len(columns) {
		return false
	}
	for i, column := range columns {
		key := index.Keys[i]
		if key.Column != column || key.Expression != nil || key.Order != physical.IndexOrderDefault || key.NullsOrder != physical.IndexNullsOrderDefault {
			return false
		}
	}
	return true
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
