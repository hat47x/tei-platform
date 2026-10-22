//go:build !tei_repair_only

package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

// These tests need an empty, disposable PostgreSQL database named by
// TEI_TEST_POSTGRES_DSN. Without it they skip. The table is the physical
// public.orders declared by reference/workspace databases/sales-core; it is
// created and dropped here, never read from an existing database.
//
// What they show: with a real store injected, the accepted Command's effect is
// a committed row change, and the Observation is read back over another pool
// and, separately, by another OS process. What they do not show: Source R1
// binding, a real Host Grant, or that the browse read comes from PostgreSQL
// (it is still the fixture executor), nor a generated CRUD.

const (
	orderSubmitDSNEnv    = "TEI_TEST_POSTGRES_DSN"
	orderSubmitHelperEnv = "TEI_ORDER_SUBMIT_HELPER_READ"
)

func orderSubmitTestDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv(orderSubmitDSNEnv)
	if dsn == "" {
		t.Skip(orderSubmitDSNEnv + " is not set")
	}
	return dsn
}

func openOrderFixtureDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var userRelations int
	err = db.QueryRowContext(ctx, `
SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r','p','v','m','S','f')
  AND n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%'`).Scan(&userRelations)
	if err != nil {
		t.Fatal(err)
	}
	if userRelations != 0 {
		t.Fatalf("refusing to use a database that already has %d user relations", userRelations)
	}
	if _, err := db.ExecContext(ctx, `
CREATE TABLE public.orders (
  order_id bigint PRIMARY KEY,
  customer_id bigint NOT NULL,
  ordered_at timestamptz NOT NULL,
  status varchar NOT NULL,
  total_amount numeric NOT NULL,
  external_id varchar
);
INSERT INTO public.orders VALUES
  (1001, 11, '2026-09-18T09:30:00Z', 'draft', 12500.00, NULL),
  (1002, 12, '2026-09-18T08:15:00Z', 'draft', 8300.00, NULL);`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_, _ = db.ExecContext(cleanup, `DROP TABLE IF EXISTS public.orders`)
	})
	return db
}

func orderStatus(t *testing.T, db *sql.DB, orderID int64) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM public.orders WHERE order_id = $1`, orderID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func newOrderSubmitRunner(t *testing.T, store orderSubmitStore) *referenceCommandScenarioRunner {
	t.Helper()
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	runner.submitStore = store
	return runner
}

var orderSubmitInput = commandScenarioInput{
	LoginID:    "legacy-order-editor",
	Credential: "editor-secret",
	From:       "2026-09-18T08:30:00Z",
	RowIndex:   0,
}

func TestPostgresOrderSubmitWritesRealRowAndIsReadByAnotherProcess(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	db := openOrderFixtureDB(t, dsn)
	store, err := newPostgresOrderSubmitStore(dsn, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	result, err := newOrderSubmitRunner(t, store).RunCommand(context.Background(), orderSubmitInput)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stage != "completed" {
		t.Fatalf("stage=%s result=%+v", result.Stage, result)
	}
	if result.Observation == nil ||
		result.Observation.ReadIdentity != "postgres.order_status.read" ||
		result.Observation.Nature != "external-observed" ||
		result.Observation.Result != "submitted" {
		t.Fatalf("observation=%+v", result.Observation)
	}
	// The write is a committed row change, visible to a connection that did not write it.
	if got := orderStatus(t, db, 1001); got != "submitted" {
		t.Fatalf("order 1001 status=%q", got)
	}
	if got := orderStatus(t, db, 1002); got != "draft" {
		t.Fatalf("order 1002 must not change, status=%q", got)
	}

	// A different OS process reads the same row; this is the independent invocation.
	command := exec.Command(os.Args[0], "-test.run=^TestHelperReadOrderStatus$", "-test.v")
	command.Env = append(os.Environ(), orderSubmitHelperEnv+"=1001")
	var stdout bytes.Buffer
	command.Stdout = &stdout
	if err := command.Run(); err != nil {
		t.Fatalf("helper process: %v\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "ORDER_STATUS status=submitted") {
		t.Fatalf("helper output:\n%s", stdout.String())
	}
}

// TestHelperReadOrderStatus runs only as the child process of the test above.
func TestHelperReadOrderStatus(t *testing.T) {
	orderID := os.Getenv(orderSubmitHelperEnv)
	if orderID == "" {
		t.Skip("child process only")
	}
	db, err := postgresdb.OpenSQL(os.Getenv(orderSubmitDSNEnv))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var status string
	if err := db.QueryRow(
		`SELECT status FROM public.orders WHERE order_id = $1::bigint`, orderID,
	).Scan(&status); err != nil {
		t.Fatal(err)
	}
	t.Logf("ORDER_STATUS status=%s", status)
}

func TestPostgresOrderSubmitRejectsAlreadySubmittedOrderWithoutChangingState(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	db := openOrderFixtureDB(t, dsn)
	if _, err := db.Exec(`UPDATE public.orders SET status = 'cancelled' WHERE order_id = 1001`); err != nil {
		t.Fatal(err)
	}
	store, err := newPostgresOrderSubmitStore(dsn, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	result, err := newOrderSubmitRunner(t, store).RunCommand(context.Background(), orderSubmitInput)
	if !errors.Is(err, errOrderNotSubmittable) {
		t.Fatalf("err=%v result=%+v", err, result)
	}
	if result.Observation != nil || result.Command != nil {
		t.Fatalf("no Command result or Observation may be reported: %+v", result)
	}
	if got := orderStatus(t, db, 1001); got != "cancelled" {
		t.Fatalf("status changed to %q", got)
	}
}

type failingReadStore struct{ orderSubmitStore }

func (failingReadStore) ReadStatus(context.Context, int64) (string, error) {
	return "", errors.New("read unavailable")
}

func TestPostgresOrderSubmitDoesNotReportObservedWhenReadFailsAfterCommit(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	db := openOrderFixtureDB(t, dsn)
	store, err := newPostgresOrderSubmitStore(dsn, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	result, err := newOrderSubmitRunner(t, failingReadStore{store}).RunCommand(context.Background(), orderSubmitInput)
	if err == nil {
		t.Fatalf("a failed Observation read must be an error: %+v", result)
	}
	if result.Observation != nil || result.Stage == "completed" {
		t.Fatalf("observation must not be reported: %+v", result)
	}
	// The Command committed; the failed read is not a rollback and not a success.
	if got := orderStatus(t, db, 1001); got != "submitted" {
		t.Fatalf("status=%q", got)
	}
}
