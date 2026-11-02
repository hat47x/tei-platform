package expectedstate

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"tei/tools/internal/postgresdb"
)

// These tests run against a real PostgreSQL and check that an update guarded by
// the expected state can not overwrite a row another writer already changed.
// They are skipped unless TEI_TEST_POSTGRES_DSN is set, and each test uses its
// own throwaway database.
func realDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEI_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEI_TEST_POSTGRES_DSN is not set")
	}
	return dsn
}

func throwawayDB(t *testing.T, adminDSN string) *sql.DB {
	t.Helper()
	admin, err := postgresdb.OpenSQL(adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	name := fmt.Sprintf("tei_expected_state_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() { admin.Exec(`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`) })
	u, err := url.Parse(adminDSN)
	if err != nil {
		t.Fatal("invalid DSN")
	}
	u.Path = "/" + name
	db, err := postgresdb.OpenSQL(u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func setupOrders(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE orders (order_id text PRIMARY KEY, status text NOT NULL)`,
		`INSERT INTO orders (order_id, status) VALUES ('O-1', 'new')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
}

// updateGuarded applies the update only when the stored status still equals the
// value the caller read. It returns how many rows changed.
func updateGuarded(ctx context.Context, tx *sql.Tx, orderID, expected, next string) (int64, error) {
	result, err := tx.ExecContext(ctx,
		`UPDATE orders SET status = $1 WHERE order_id = $2 AND status = $3`,
		next, orderID, expected)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// TestExpectedStateStopsTheSecondWriter fixes H1 on a real database: the second
// writer, which read the old status, changes zero rows and the first writer's
// value survives.
func TestExpectedStateStopsTheSecondWriter(t *testing.T) {
	db := throwawayDB(t, realDSN(t))
	setupOrders(t, db)
	ctx := context.Background()

	first, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	firstChanged, err := updateGuarded(ctx, first, "O-1", "new", "confirmed")
	if err != nil {
		t.Fatalf("first update: %v", err)
	}
	if firstChanged != 1 {
		t.Fatalf("the first writer must change the row: %d", firstChanged)
	}
	if err := first.Commit(); err != nil {
		t.Fatalf("commit the first writer: %v", err)
	}

	secondChanged, err := updateGuarded(ctx, second, "O-1", "new", "cancelled")
	if err != nil {
		t.Fatalf("second update: %v", err)
	}
	if secondChanged != 0 {
		t.Fatalf("the second writer read the old state, so it must change nothing: %d", secondChanged)
	}
	if err := second.Rollback(); err != nil {
		t.Fatal(err)
	}

	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM orders WHERE order_id = 'O-1'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "confirmed" {
		t.Fatalf("the first writer's value must survive: %s", status)
	}
}

// TestExpectedStateAllowsTheSecondWriterAfterAReRead is the positive control:
// after re-reading the current state, the second writer succeeds.
func TestExpectedStateAllowsTheSecondWriterAfterAReRead(t *testing.T) {
	db := throwawayDB(t, realDSN(t))
	setupOrders(t, db)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `UPDATE orders SET status = 'confirmed' WHERE order_id = 'O-1' AND status = 'new'`); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := updateGuarded(ctx, tx, "O-1", "confirmed", "cancelled")
	if err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Fatalf("an update that matches the current state must change the row: %d", changed)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
