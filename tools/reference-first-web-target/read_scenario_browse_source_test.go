//go:build !tei_order_only && !tei_repair_only

package main

import (
	"os"
	"strings"
	"testing"

	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

// The browse half and the detail half of one page must come from the same
// source. Before this was enforced, newPostgresReadRunner replaced only the
// detail executor, so a page could select an order from the fixture and show
// that order's detail rows from the database.
//
// The measurement is the database itself: a database with no public.orders
// cannot answer the browse Query, so a runner that still produced rows here
// would be reading them from somewhere other than the connection it was given.
func TestPostgresReadRunnerServesBrowseFromTheSameDatabase(t *testing.T) {
	dsn := os.Getenv(readPostgresDSNEnv)
	if dsn == "" {
		t.Skip(readPostgresDSNEnv + " is not set; real PostgreSQL read not run")
	}
	if os.Getenv(readPostgresNameEnv) != readPostgresDatabase {
		t.Fatalf("refusing to modify %q without %s=%s", readPostgresDatabase, readPostgresNameEnv, readPostgresDatabase)
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := requireIsolatedWebReadDatabase(t.Context(), db, readPostgresDatabase); err != nil {
		t.Fatal(err)
	}

	// No parent table at all: the detail seed cannot stand in for it.
	if _, err := db.Exec("drop table if exists public.orders"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}

	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newPostgresReadRunner(loaded.World, db)
	if err != nil {
		t.Fatal(err)
	}
	_, runErr := runner.Run(t.Context(), readScenarioInput{From: "2026-09-01T00:00:00Z", RowIndex: 0})
	if runErr == nil {
		t.Fatal("a database without public.orders produced a page, so the browse rows did not come from this database")
	}
	if !strings.Contains(runErr.Error(), "browse") {
		t.Fatalf("the refusal does not name the browse read: %v", runErr)
	}
}

// The same read against a database that holds the parent rows succeeds, so the
// refusal above is about the missing table and not about the read being broken.
func TestPostgresReadRunnerServesBrowseWhenTheTableExists(t *testing.T) {
	dsn := os.Getenv(readPostgresDSNEnv)
	if dsn == "" {
		t.Skip(readPostgresDSNEnv + " is not set; real PostgreSQL read not run")
	}
	if os.Getenv(readPostgresNameEnv) != readPostgresDatabase {
		t.Fatalf("refusing to modify %q without %s=%s", readPostgresDatabase, readPostgresNameEnv, readPostgresDatabase)
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := requireIsolatedWebReadDatabase(t.Context(), db, readPostgresDatabase); err != nil {
		t.Fatal(err)
	}
	if err := seedReadFixture(t, db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("drop table if exists public.order_lines")
		_, _ = db.Exec("drop table if exists public.orders")
	})

	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newPostgresReadRunner(loaded.World, db)
	if err != nil {
		t.Fatal(err)
	}
	read, err := runner.Run(t.Context(), readScenarioInput{From: "2026-09-01T00:00:00Z", RowIndex: 0})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// The browse rows are the seeded parent orders, newest first.
	if len(read.BrowseRows) != 3 {
		t.Fatalf("browse rows=%d, want the 3 seeded orders", len(read.BrowseRows))
	}
	for index, want := range []string{"1001", "1002", "1003"} {
		got, ok := orderLineAttribute(read.BrowseRows[index], "sales.order.order_id")
		if !ok {
			t.Fatalf("browse row %d has no order_id", index)
		}
		if got != want {
			t.Fatalf("browse row %d order_id=%q, want %q in ordered_at DESC order", index, got, want)
		}
	}
}
