package sqlite_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"tei/internal/core"
	"tei/internal/database/provider"
	"tei/internal/database/providers/sqlite"
	"tei/internal/environment"
)

func openTestSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func setupTestSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE customers (
			customer_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		);
		CREATE TABLE orders (
			order_id INTEGER PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			order_date TEXT,
			FOREIGN KEY (customer_id) REFERENCES customers(customer_id)
		);
		CREATE UNIQUE INDEX idx_customers_email ON customers(email);
		CREATE INDEX idx_orders_date ON orders(order_date);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteProviderIntrospectsPhysicalSchema(t *testing.T) {
	db := openTestSQLite(t)
	defer db.Close()
	setupTestSchema(t, db)

	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("test.dev"), Definition: core.MustCanonicalID("test"), Provider: environment.ProviderSQLite}
	fixed := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	p := sqlite.Provider{Catalog: sqlite.SQLCatalogReader{DB: db}, Now: func() time.Time { return fixed }}
	snap, err := provider.Introspect(context.Background(), p, inst)
	if err != nil {
		t.Fatal(err)
	}

	if snap.CapturedAt != "2026-08-27T00:00:00Z" {
		t.Fatalf("capturedAt=%q", snap.CapturedAt)
	}
	if snap.Database != "test" {
		t.Fatalf("database=%q", snap.Database)
	}

	customers, ok := snap.Table("main.customers")
	if !ok {
		t.Fatal("main.customers not resolved")
	}
	if len(customers.PrimaryKey) != 1 || customers.PrimaryKey[0] != "customer_id" {
		t.Fatalf("pk=%#v", customers.PrimaryKey)
	}
	if len(customers.Columns) != 3 {
		t.Fatalf("columns=%#v", customers.Columns)
	}
	if customers.Columns[1].Name != "name" || customers.Columns[1].Nullable {
		t.Fatalf("name column=%#v", customers.Columns[1])
	}

	orders, ok := snap.Table("main.orders")
	if !ok {
		t.Fatal("main.orders not resolved")
	}
	if len(orders.PrimaryKey) != 1 || orders.PrimaryKey[0] != "order_id" {
		t.Fatalf("pk=%#v", orders.PrimaryKey)
	}
	if len(orders.ForeignKeys) != 1 {
		t.Fatalf("fks=%#v", orders.ForeignKeys)
	}
	if orders.ForeignKeys[0].Columns[0] != "customer_id" || orders.ForeignKeys[0].ReferencedTable != "customers" {
		t.Fatalf("fk=%#v", orders.ForeignKeys[0])
	}

	// Check unique constraint was captured
	if len(customers.UniqueConstraints) != 1 {
		t.Fatalf("unique constraints=%#v", customers.UniqueConstraints)
	}
	if customers.UniqueConstraints[0].Columns[0] != "email" {
		t.Fatalf("unique constraint columns=%#v", customers.UniqueConstraints[0].Columns)
	}

	// Check index was captured
	if len(customers.Indexes) != 1 {
		t.Fatalf("indexes=%#v", customers.Indexes)
	}
	if customers.Indexes[0].Name != "idx_customers_email" {
		t.Fatalf("index name=%q", customers.Indexes[0].Name)
	}
}

func TestSQLiteProviderCapabilities(t *testing.T) {
	p := sqlite.Provider{}
	caps, err := p.Capabilities(context.Background(), environment.DatabaseInstance{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []provider.Capability{provider.CapabilityMetadataRead, provider.CapabilityQuery} {
		if !caps.Has(c) {
			t.Fatalf("missing capability %s", c)
		}
	}
}

func TestSQLiteProviderRejectsNilCatalog(t *testing.T) {
	p := sqlite.Provider{}
	_, err := provider.Introspect(context.Background(), p, environment.DatabaseInstance{})
	if err == nil {
		t.Fatal("expected error for nil catalog")
	}
}

func TestSQLiteProviderCompositeKey(t *testing.T) {
	db := openTestSQLite(t)
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE enrollment (
			student_id INTEGER NOT NULL,
			course_id INTEGER NOT NULL,
			grade TEXT,
			PRIMARY KEY (student_id, course_id)
		);
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

	enrollment, ok := snap.Table("main.enrollment")
	if !ok {
		t.Fatal("main.enrollment not resolved")
	}
	if len(enrollment.PrimaryKey) != 2 || enrollment.PrimaryKey[0] != "student_id" || enrollment.PrimaryKey[1] != "course_id" {
		t.Fatalf("composite pk=%#v", enrollment.PrimaryKey)
	}
}

func TestSQLiteProviderNullableColumns(t *testing.T) {
	db := openTestSQLite(t)
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE items (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			quantity INTEGER DEFAULT 0
		);
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

	items := snap.Tables[0]
	if len(items.Columns) != 4 {
		t.Fatalf("columns=%#v", items.Columns)
	}
	if items.Columns[1].Nullable {
		t.Fatalf("name should be NOT NULL")
	}
	if !items.Columns[2].Nullable {
		t.Fatalf("description should be nullable")
	}
	if items.Columns[3].Default != "0" {
		t.Fatalf("quantity default=%q", items.Columns[3].Default)
	}
}

func TestSQLiteProviderCircularReference(t *testing.T) {
	db := openTestSQLite(t)
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE categories (
			category_id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			name TEXT NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES categories(category_id)
		);
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

	categories := snap.Tables[0]
	if len(categories.ForeignKeys) != 1 {
		t.Fatalf("expected 1 self-referencing FK, got %d", len(categories.ForeignKeys))
	}
	if categories.ForeignKeys[0].ReferencedTable != "categories" {
		t.Fatalf("self-reference target=%q", categories.ForeignKeys[0].ReferencedTable)
	}
}

func TestSQLiteProviderPermissionDenied(t *testing.T) {
	// SQLite does not have a permission system like PostgreSQL.
	// An empty database returns an empty snapshot without error.
	// Real permission errors (file lock, read-only filesystem) are surfaced
	// as driver/runtime errors during catalog reads.
	db := openTestSQLite(t)
	defer db.Close()

	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("test.dev"), Definition: core.MustCanonicalID("test"), Provider: environment.ProviderSQLite}
	p := sqlite.Provider{Catalog: sqlite.SQLCatalogReader{DB: db}}
	snap, err := provider.Introspect(context.Background(), p, inst)
	if err != nil {
		t.Fatalf("unexpected error for empty sqlite database: %v", err)
	}
	if len(snap.Tables) != 0 {
		t.Fatalf("expected empty snapshot for empty database, got %d tables", len(snap.Tables))
	}
}
