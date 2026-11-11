package postgresquery

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// verifyRelationQueryWithPostgres compares relation query results with explicit SQL.
// It is opt-in and requires TEI_RELATION_QUERY_PG_DSN to point to a disposable database.
func verifyRelationQueryWithPostgres(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("TEI_RELATION_QUERY_PG_DSN")
	if dsn == "" {
		t.Skip("physical relation query PostgreSQL read NOT RUN: TEI_RELATION_QUERY_PG_DSN unset")
	}
	if os.Getenv("TEI_RELATION_QUERY_PG_CONFIRM_ISOLATED") != "tei_relation_query_e2e" {
		t.Fatal("refusing to create relation query fixture without explicit disposable database confirmation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize PostgreSQL driver")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("isolated PostgreSQL connection unavailable")
	}
	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); err != nil ||
		dbName != "tei_relation_query_e2e" {
		t.Fatalf("refusing to create fixture in unexpected database: %q", dbName)
	}

	// Create fixture tables
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE public.orders (
			id VARCHAR PRIMARY KEY,
			customer_id VARCHAR NOT NULL,
			status VARCHAR NOT NULL
		);
		CREATE TABLE public.customers (
			id VARCHAR PRIMARY KEY,
			name VARCHAR NOT NULL
		);
	`); err != nil {
		t.Fatalf("cannot create fixture tables: %v", err)
	}
	defer func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := db.ExecContext(cleanupCtx, "DROP TABLE IF EXISTS public.orders, public.customers"); err != nil {
			t.Errorf("cannot clean up fixture tables: %v", err)
		}
	}()

	// Seed data
	for _, entry := range [][3]string{
		{"O1", "C1", "active"},
		{"O2", "C1", "active"},
		{"O3", "C2", "inactive"},
	} {
		if _, err := db.ExecContext(ctx,
			"INSERT INTO public.orders (id, customer_id, status) VALUES ($1, $2, $3)",
			entry[0], entry[1], entry[2]); err != nil {
			t.Fatalf("cannot seed orders: %v", err)
		}
	}
	for _, entry := range [][2]string{
		{"C1", "Alice"},
		{"C2", "Bob"},
	} {
		if _, err := db.ExecContext(ctx,
			"INSERT INTO public.customers (id, name) VALUES ($1, $2)",
			entry[0], entry[1]); err != nil {
			t.Fatalf("cannot seed customers: %v", err)
		}
	}

	world := irWorldWithRelationQuery()
	loaded := workspace.Result{World: world}
	resolver := driver.NewResolver(loaded.World)
	resolved, err := resolver.ResolveRelationQuery("order_entry", "order_entry.order_with_customer")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	mappings := []persistence.Mapping{
		{
			SemanticData: "sales.order",
			Table:        "orders",
			Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "id"},
				{Attribute: "sales.order.customer_id", Column: "customer_id"},
				{Attribute: "sales.order.status", Column: "status"},
			},
		},
		{
			SemanticData: "customer.customer",
			Table:        "customers",
			Columns: []persistence.ColumnMapping{
				{Attribute: "customer.customer.customer_id", Column: "id"},
				{Attribute: "customer.customer.name", Column: "name"},
			},
		},
	}

	executor := Executor{DB: db, Database: "master_data", Mappings: mappings}
	execution, err := executor.ExecuteRelationQuery(ctx, resolved, application.DatabaseRequirement{TargetDatabase: "master_data"}, nil)
	if err != nil {
		t.Fatalf("ExecuteRelationQuery error: %v", err)
	}

	// Verify we got 3 rows (all orders with LEFT JOIN to customers)
	if len(execution.Raw.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(execution.Raw.Rows))
	}

	// Verify explicit SQL produces the same row count
	var explicitCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orders
		LEFT JOIN customers ON orders.customer_id = customers.id
	`).Scan(&explicitCount); err != nil {
		t.Fatalf("explicit SQL count error: %v", err)
	}
	if explicitCount != 3 {
		t.Fatalf("explicit SQL count mismatch: got %d, want 3", explicitCount)
	}
}

func TestRelationQueryPostgresIntegration(t *testing.T) {
	verifyRelationQueryWithPostgres(t)
}

func irWorldWithRelationQuery() ir.World {
	orderData := semantic.Data{
		ID: "sales.order",
		Attributes: []semantic.Attribute{
			{ID: "sales.order.order_id", TypeRef: "sales.order_id"},
			{ID: "sales.order.customer_id", TypeRef: "customer.customer_id"},
			{ID: "sales.order.status", TypeRef: "std.text"},
		},
	}
	customerData := semantic.Data{
		ID: "customer.customer",
		Attributes: []semantic.Attribute{
			{ID: "customer.customer.customer_id", TypeRef: "customer.customer_id"},
			{ID: "customer.customer.name", TypeRef: "std.text"},
		},
	}
	relation := semantic.Relation{
		ID:          "sales.order.customer",
		Target:      "customer.customer",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"},
		},
	}
	orderData.Relations = []semantic.Relation{relation}
	app := application.Application{
		ID:           "order_entry",
		Databases:    []application.DatabaseRequirement{{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead}},
		SemanticUses: []core.CanonicalID{"sales", "customer"},
	}
	relationQuery := query.RelationQuery{
		ID:          "order_entry.order_with_customer",
		Application: "order_entry",
		Datasource:  "master",
		Root:        "sales.order",
		Relations:   []query.RelationStep{{Relation: "sales.order.customer"}},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "order_id", Attribute: "sales.order.order_id"},
			{Column: "customer_id", Attribute: "sales.order.customer_id"},
			{Column: "name", Attribute: "customer.customer.name"},
			{Column: "status", Attribute: "sales.order.status"},
		},
	}
	return ir.World{
		Applications: []application.Application{app},
		Semantics: []semantic.Namespace{
			{ID: "sales", Data: []semantic.Data{orderData}},
			{ID: "customer", Data: []semantic.Data{customerData}},
		},
		RelationQueries: []query.RelationQuery{relationQuery},
	}
}
