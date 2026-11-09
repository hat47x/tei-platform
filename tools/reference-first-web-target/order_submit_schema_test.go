//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"tei/internal/ir"
	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

// openDeclaredOrderFixtureDB is openOrderFixtureDB with the table built from
// the accepted declaration (Persistence Mapping -> existing Provider DDL)
// instead of hand-written DDL. The hand-written fixture lacks the declared
// index orders_customer_idx, so it does not pass the readiness check.
func openDeclaredOrderFixtureDB(t *testing.T, dsn string, world ir.World) *sql.DB {
	t.Helper()
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var userRelations int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r','p','v','m','S','f')
  AND n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%'`).Scan(&userRelations); err != nil {
		t.Fatal(err)
	}
	if userRelations != 0 {
		t.Fatalf("refusing to use a database that already has %d user relations", userRelations)
	}
	resetDeclaredOrders(t, db, world)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_, _ = db.ExecContext(cleanup, `DROP TABLE IF EXISTS public.orders`)
	})
	return db
}

// resetDeclaredOrders (re)creates public.orders from the declaration with
// orders 1001 and 1002 in draft.
func resetDeclaredOrders(t *testing.T, db *sql.DB, world ir.World) {
	t.Helper()
	statements, err := renderOrderSubmitTableDDL(world)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE IF EXISTS public.orders`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("declared DDL failed: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount, external_id) VALUES
  (1001, 11, '2026-09-18T09:30:00Z', 'draft', 12500.00, NULL),
  (1002, 12, '2026-09-18T08:15:00Z', 'draft', 8300.00, NULL)`); err != nil {
		t.Fatal(err)
	}
}

// otherDatabaseDSN creates an empty database in the same disposable server and
// returns its DSN. The database is dropped at cleanup.
func otherDatabaseDSN(t *testing.T, dsn, name string) string {
	t.Helper()
	admin, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	if _, err := admin.Exec(`DROP DATABASE IF EXISTS ` + name + ` WITH (FORCE)`); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(`CREATE DATABASE ` + name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(`DROP DATABASE IF EXISTS ` + name + ` WITH (FORCE)`) })
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal("DSN is not a URL")
	}
	parsed.Path = "/" + name
	return parsed.String()
}

func TestOrderSubmitSchemaReadinessAgainstRealPostgres(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	world := loaded.World
	db := openDeclaredOrderFixtureDB(t, dsn, world)
	store, err := newPostgresOrderSubmitStore(dsn, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()

	readiness, err := store.checkOrderSubmitSchema(ctx, world)
	if err != nil || !readiness.Compatible || readiness.Table != "public.orders" {
		t.Fatalf("the declared table is not compatible: %+v %v", readiness, err)
	}
	checked, err := requireOrderSubmitSchema(ctx, world, store)
	if err != nil {
		t.Fatal(err)
	}

	// Each drift is refused before the write; the row and the row count stay.
	for _, drift := range []struct{ name, sql, change string }{
		{"missing index", `DROP INDEX public.orders_customer_idx`, "orders_customer_idx"},
		{"missing column", `ALTER TABLE public.orders DROP COLUMN external_id`, "external_id"},
		{"changed type", `ALTER TABLE public.orders ALTER COLUMN status TYPE text`, "status"},
	} {
		t.Run(drift.name, func(t *testing.T) {
			resetDeclaredOrders(t, db, world)
			if _, err := db.Exec(drift.sql); err != nil {
				t.Fatal(err)
			}
			readiness, err := store.checkOrderSubmitSchema(ctx, world)
			if !errors.Is(err, errPhysicalSchemaNotReady) || readiness.Compatible ||
				!strings.Contains(strings.Join(readiness.Changes, " "), drift.change) {
				t.Fatalf("drift not detected: %+v %v", readiness, err)
			}
			if err := checked.Submit(ctx, 1001); !errors.Is(err, errPhysicalSchemaNotReady) {
				t.Fatalf("Submit did not refuse before writing: %v", err)
			}
			if got := orderStatus(t, db, 1001); got != "draft" {
				t.Fatalf("order 1001 changed to %q", got)
			}
			if _, err := requireOrderSubmitSchema(ctx, world, store); !errors.Is(err, errPhysicalSchemaNotReady) {
				t.Fatalf("startup check accepted the drifted table: %v", err)
			}
		})
	}

	// After the table is restored, the same checked store writes.
	resetDeclaredOrders(t, db, world)
	if err := checked.Submit(ctx, 1001); err != nil {
		t.Fatal(err)
	}
	if got := orderStatus(t, db, 1001); got != "submitted" {
		t.Fatalf("order 1001 status=%q", got)
	}
}

func TestOrderSubmitSchemaReadinessWithoutDatabaseIsNotReady(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	statements, err := renderOrderSubmitTableDDL(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(statements, "\n")
	for _, want := range []string{"orders", "order_id", "orders_customer_idx"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("declared DDL lacks %s:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "order_lines") {
		t.Fatal("the readiness scope is public.orders only")
	}
	// A store that cannot reach any database is not ready; the error is the
	// public code only.
	store, err := newPostgresOrderSubmitStore("postgres://tei@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", "postgres://tei@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Skip("store could not be constructed without a server")
	}
	defer store.Close()
	_, err = requireOrderSubmitSchema(context.Background(), loaded.World, store)
	if !errors.Is(err, errPhysicalSchemaNotReady) || err.Error() != "PHYSICAL_SCHEMA_NOT_READY" {
		t.Fatalf("unreachable database: %v", err)
	}
}
