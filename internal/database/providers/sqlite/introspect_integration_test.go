package sqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"tei/internal/core"
	"tei/internal/database/provider"
	"tei/internal/database/providers/sqlite"
	"tei/internal/database/schematosemantic"
	"tei/internal/environment"
	"tei/internal/semantic"
)

// TestSQLiteEndToEnd verifies the full SQLite introspection + semantic
// projection path end-to-end. It is an integration test because it touches
// a live SQLite database and exercises both the provider and the projection.
func TestSQLiteEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE customers (
			customer_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE orders (
			order_id INTEGER PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			total DECIMAL(10,2),
			FOREIGN KEY (customer_id) REFERENCES customers(customer_id)
		);
		CREATE VIEW active_customers AS
			SELECT * FROM customers WHERE created_at > '2020-01-01';
	`)
	if err != nil {
		t.Fatal(err)
	}

	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("test.dev"), Definition: core.MustCanonicalID("test"), Provider: environment.ProviderSQLite}
	p := sqlite.Provider{Catalog: sqlite.SQLCatalogReader{DB: db}}

	snap, err := provider.Introspect(context.Background(), p, inst)
	if err != nil {
		t.Fatal(err)
	}

	// Verify physical snapshot
	if len(snap.Tables) != 3 {
		t.Fatalf("expected 3 tables/views, got %d: %#v", len(snap.Tables), snap.Tables)
	}

	customers, ok := snap.Table("main.customers")
	if !ok {
		t.Fatal("main.customers not resolved")
	}
	if len(customers.Columns) != 4 {
		t.Fatalf("customers columns=%#v", customers.Columns)
	}
	if len(customers.ForeignKeys) != 0 {
		t.Fatalf("customers foreign keys=%#v", customers.ForeignKeys)
	}

	orders, ok := snap.Table("main.orders")
	if !ok {
		t.Fatal("main.orders not resolved")
	}
	if len(orders.ForeignKeys) != 1 {
		t.Fatalf("orders foreign keys=%#v", orders.ForeignKeys)
	}
	if orders.ForeignKeys[0].ReferencedTable != "customers" {
		t.Fatalf("orders fk target=%q", orders.ForeignKeys[0].ReferencedTable)
	}

	// Verify semantic projection
	result, err := schematosemantic.Project(snap, schematosemantic.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	if result.Namespace.ID != "test" {
		t.Fatalf("namespace id=%q", result.Namespace.ID)
	}
	if len(result.Namespace.Data) != 3 {
		t.Fatalf("expected 3 data entities, got %d", len(result.Namespace.Data))
	}

	// Find orders data entity
	var ordersData *semantic.Data
	for i := range result.Namespace.Data {
		if result.Namespace.Data[i].ID == "main_orders" {
			ordersData = &result.Namespace.Data[i]
			break
		}
	}
	if ordersData == nil {
		t.Fatal("main_orders not found in semantic model")
	}

	if len(ordersData.Relations) != 1 {
		t.Fatalf("orders relations=%#v", ordersData.Relations)
	}
	if ordersData.Relations[0].Target != "main_customers" {
		t.Fatalf("orders relation target=%q", ordersData.Relations[0].Target)
	}

	if len(result.Provenance.ContentFingerprint) == 0 {
		t.Fatal("missing content fingerprint")
	}
	if result.Provenance.CapturedAt != snap.CapturedAt {
		t.Fatalf("provenance capturedAt mismatch: %q != %q", result.Provenance.CapturedAt, snap.CapturedAt)
	}
}
