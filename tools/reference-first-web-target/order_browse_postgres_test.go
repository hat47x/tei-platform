//go:build !tei_repair_only

package main

import (
	"context"
	"strings"
	"testing"

	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

func TestBindNamedParameters(t *testing.T) {
	t.Run("rewrites to positions and leaves casts alone", func(t *testing.T) {
		statement, args, err := bindNamedParameters(
			"SELECT 1 WHERE a >= :from AND b::text = 'x' AND c <= :from AND d = :to",
			map[string]any{"from": 1, "to": 2},
		)
		if err != nil {
			t.Fatal(err)
		}
		if statement != "SELECT 1 WHERE a >= $1 AND b::text = 'x' AND c <= $1 AND d = $2" {
			t.Fatalf("statement=%q", statement)
		}
		if len(args) != 2 || args[0] != 1 || args[1] != 2 {
			t.Fatalf("args=%v", args)
		}
	})
	t.Run("rejects an unbound name", func(t *testing.T) {
		if _, _, err := bindNamedParameters("SELECT :from", map[string]any{}); err == nil {
			t.Fatal("unbound parameter was accepted")
		}
	})
	t.Run("rejects an unused value", func(t *testing.T) {
		if _, _, err := bindNamedParameters("SELECT 1", map[string]any{"from": 1}); err == nil {
			t.Fatal("unused parameter was accepted")
		}
	})
}

// The browse read comes from PostgreSQL: the newest order is not one of the
// fixed fixture rows, so selecting row 0 and submitting it changes that row
// only. The customer and order-line reads are still the existing fixtures.
func TestPostgresBrowseSelectsTheOrderThatTheCommandSubmits(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	db := openOrderFixtureDB(t, dsn)
	if _, err := db.Exec(`INSERT INTO public.orders VALUES (2001, 11, '2026-09-19T10:00:00Z', 'draft', 9900.00, NULL)`); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	store, err := newPostgresOrderSubmitStore(dsn, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	runner := newOrderSubmitRunner(t, store)
	runner.read = newPostgresBrowseReadRunner(loaded.World, db)

	result, err := runner.RunCommand(context.Background(), orderSubmitInput)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stage != "completed" || result.Command == nil ||
		result.Command.Payload["orderId"] != int64(2001) {
		t.Fatalf("result=%+v", result)
	}
	if result.Observation == nil || result.Observation.Result != "submitted" {
		t.Fatalf("observation=%+v", result.Observation)
	}
	if got := orderStatus(t, db, 2001); got != "submitted" {
		t.Fatalf("order 2001 status=%q", got)
	}
	for _, untouched := range []int64{1001, 1002} {
		if got := orderStatus(t, db, untouched); got != "draft" {
			t.Fatalf("order %d status=%q", untouched, got)
		}
	}
}

func TestPostgresBrowseFailureDoesNotLeakConnectionDetail(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	_ = openOrderFixtureDB(t, dsn)
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	closed, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	_ = closed.Close()
	runner := newPostgresBrowseReadRunner(loaded.World, closed)

	_, err = runner.Run(context.Background(), readScenarioInput{From: orderSubmitInput.From, RowIndex: 0})
	if err == nil {
		t.Fatal("a failing browse read must be an error")
	}
	for _, secret := range []string{"127.0.0.1", "postgres", "55432", "pw"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaks %q: %v", secret, err)
		}
	}
}
