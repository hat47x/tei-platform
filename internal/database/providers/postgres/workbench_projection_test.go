package postgres_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/database/physical"
	dbprovider "tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/environment"
	"tei/internal/index"
	workbench "tei/internal/workbench/database"
)

// This exercises the production CatalogReader -> Introspect -> Workbench
// projection path with a synthetic catalog. It is not a live database test
// and does not establish semantic acceptance of the observed constraints.
func TestObservedPhysicalRelationsReachWorkbenchWithoutSemanticInference(t *testing.T) {
	instance := environment.DatabaseInstance{
		ID: core.MustCanonicalID("sales_core.dev"), Definition: core.MustCanonicalID("sales_core"),
		Provider: environment.ProviderPostgres,
	}
	fixed := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	p := postgres.Provider{Catalog: fakeCatalog{}, Now: func() time.Time { return fixed }}
	snapshot, err := dbprovider.Introspect(context.Background(), p, instance)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Fingerprint = "synthetic-observation-r0"
	got := workbench.Build(index.Derived{PhysicalSchemas: []physical.SchemaSnapshot{snapshot}})
	if len(got.Schemas) != 1 || got.Schemas[0].Database != instance.Definition {
		t.Fatalf("schema=%#v", got.Schemas)
	}
	schema := got.Schemas[0]
	if schema.CapturedAt != fixed.Format(time.RFC3339) || schema.Fingerprint != snapshot.Fingerprint {
		t.Fatalf("observation provenance was lost: %#v", schema)
	}
	if len(got.Queries) != 0 {
		t.Fatalf("physical catalog must not synthesize application query meaning: %#v", got.Queries)
	}
	var orders *workbench.TableView
	for i := range schema.Tables {
		if schema.Tables[i].Name == "public.orders" {
			orders = &schema.Tables[i]
		}
		if schema.Tables[i].Name == "reporting.orders" && len(schema.Tables[i].ForeignKeys) != 0 {
			t.Fatalf("view unexpectedly gained catalog relationships: %#v", schema.Tables[i].ForeignKeys)
		}
	}
	if orders == nil {
		t.Fatal("qualified physical table public.orders was not projected")
	}
	if len(orders.PrimaryKey) != 1 || orders.PrimaryKey[0] != "order_id" {
		t.Fatalf("physical primary key was lost: %#v", orders.PrimaryKey)
	}
	if len(orders.UniqueConstraints) != 1 {
		t.Fatalf("physical uniqueness was lost: %#v", orders.UniqueConstraints)
	}
	unique := orders.UniqueConstraints[0]
	if unique.Name != "orders_customer_key" || len(unique.Columns) != 1 || unique.Columns[0] != "customer_id" {
		t.Fatalf("unique constraint changed: %#v", unique)
	}
	if len(orders.Indexes) != 2 {
		t.Fatalf("physical indexes were lost: %#v", orders.Indexes)
	}
	regular, expression := orders.Indexes[0], orders.Indexes[1]
	if regular.Name != "orders_customer_idx" || regular.State == nil || !regular.State.Usable() ||
		len(regular.Keys) != 1 || regular.Keys[0].Column != "customer_id" {
		t.Fatalf("regular index state and key changed: %#v", regular)
	}
	if expression.Name != "orders_note_expr_idx" || expression.State == nil ||
		expression.State.Valid || !expression.State.Ready || !expression.State.Live ||
		len(expression.Keys) != 1 || expression.Keys[0].Expression == nil ||
		expression.Keys[0].Expression.Source != "lower(note)" ||
		expression.Keys[0].Expression.Normalization == nil ||
		expression.Keys[0].Expression.Normalization.Authority != "postgresql-catalog-deparse" ||
		expression.Predicate == nil || expression.Predicate.Source != "note IS NOT NULL" ||
		len(expression.IncludedColumns) != 1 || expression.IncludedColumns[0] != "order_id" {
		t.Fatalf("expression index details lost: %#v", expression)
	}
	if len(orders.ForeignKeys) != 1 {
		t.Fatalf("physical relationship was lost: %#v", orders.ForeignKeys)
	}
	fk := orders.ForeignKeys[0]
	if fk.Name != "orders_customer_fk" || len(fk.Columns) != 1 || fk.Columns[0] != "customer_id" ||
		fk.ReferencedSchema != "public" || fk.ReferencedTable != "customers" ||
		len(fk.ReferencedColumns) != 1 || fk.ReferencedColumns[0] != "customer_id" {
		t.Fatalf("physical FK identity/topology changed: %#v", fk)
	}

	// A UI/JSON consumer must not alias the original schema's mutable slices.
	original, exists := snapshot.Table("public.orders")
	if !exists {
		t.Fatal("source catalog table missing")
	}
	original.PrimaryKey[0] = "changed"
	original.UniqueConstraints[0].Columns[0] = "changed"
	original.ForeignKeys[0].Columns[0] = "changed"
	original.ForeignKeys[0].ReferencedColumns[0] = "changed"
	original.Indexes[0].Keys[0].Column = "changed"
	original.Indexes[1].Keys[0].Expression.Source = "changed"
	original.Indexes[1].Keys[0].Expression.Normalization.Authority = "changed"
	original.Indexes[1].Predicate.Source = "changed"
	original.Indexes[1].IncludedColumns[0] = "changed"
	original.Indexes[1].State.Valid = true
	if orders.PrimaryKey[0] != "order_id" || orders.UniqueConstraints[0].Columns[0] != "customer_id" ||
		orders.ForeignKeys[0].Columns[0] != "customer_id" || orders.ForeignKeys[0].ReferencedColumns[0] != "customer_id" ||
		orders.Indexes[0].Keys[0].Column != "customer_id" ||
		orders.Indexes[1].Keys[0].Expression.Source != "lower(note)" ||
		orders.Indexes[1].Keys[0].Expression.Normalization.Authority != "postgresql-catalog-deparse" ||
		orders.Indexes[1].Predicate.Source != "note IS NOT NULL" ||
		orders.Indexes[1].IncludedColumns[0] != "order_id" || orders.Indexes[1].State.Valid {
		t.Fatalf("Workbench result aliases mutable physical source: %#v", orders)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"capturedAt":"2026-11-07T00:00:00Z"`, `"fingerprint":"synthetic-observation-r0"`, `"uniqueConstraints"`, `"foreignKeys"`, `"referencedTable":"customers"`, `"indexes"`, `"predicate"`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("read-only JSON output lacks %s: %s", key, raw)
		}
	}
}
