package numberingpostgres

import (
	"context"
	"os"
	"testing"

	"tei/internal/numbering"
)

// The suite runs only when a PostgreSQL DSN is supplied through
// TEI_NUMBERING_POSTGRES_DSN (or TEI_TEST_POSTGRES_DSN). Without it the tests SKIP,
// and a SKIP is not evidence. The table is throwaway, so a pass does not qualify a
// production database.
//
// Why a separate suite at all: the SQLite store keeps two writers apart with
// _txlock=immediate and a single connection. PostgreSQL has no such trick here; it
// keeps them apart with the WHERE clause on the version alone. The invariant is the
// same, the mechanism is not, so the invariant has to be pressed on this store too.

func postgresDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEI_NUMBERING_POSTGRES_DSN")
	if dsn == "" {
		dsn = os.Getenv("TEI_TEST_POSTGRES_DSN")
	}
	if dsn == "" {
		t.Skip("set TEI_NUMBERING_POSTGRES_DSN (or TEI_TEST_POSTGRES_DSN) to run the PostgreSQL numbering suite")
	}
	return dsn
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	store, err := Open(ctx, postgresDSN(t), "numbering_counter_test")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	// A previous run's rows would make the first load look like a continuation.
	if _, err := store.db.ExecContext(ctx, `DELETE FROM "numbering_counter_test"`); err != nil {
		t.Fatalf("reset: %v", err)
	}
	return store
}

// TestPostgresStoreStartsAtZero checks the state a rule begins in.
func TestPostgresStoreStartsAtZero(t *testing.T) {
	store := openTestStore(t)
	state, version, err := store.Load(context.Background(), "order")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if state.Current != 0 || version != 0 {
		t.Fatalf("state=%+v version=%d, want zero at version zero", state, version)
	}
}

// TestPostgresReservesAcrossReopen checks that an issued number survives a new
// connection, which is the point of a durable store.
func TestPostgresReservesAcrossReopen(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	contract := numbering.Contract{Digits: 4}
	number, reservation, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if number != "0001" {
		t.Fatalf("number=%q", number)
	}
	if err := reservation.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	reopened, err := Open(ctx, postgresDSN(t), "numbering_counter_test")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = reopened.Close() }()
	next, reservation2, err := contract.Reserve(ctx, reopened, "order")
	if err != nil {
		t.Fatalf("reserve after reopen: %v", err)
	}
	if next != "0002" {
		t.Fatalf("the counter did not survive the reopen: %q", next)
	}
	if err := reservation2.Commit(); err != nil {
		t.Fatalf("commit 2: %v", err)
	}
}

// TestPostgresRefusesAStaleVersion checks the conditional write: a caller holding an
// old version is refused rather than allowed to overwrite a newer counter.
func TestPostgresRefusesAStaleVersion(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 5}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	// The row is now at version 1, so a writer that still thinks it is at 0 is stale.
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 9}); err == nil {
		t.Fatal("a stale version was accepted")
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if state.Current != 5 {
		t.Fatalf("the refused write still changed the counter: %d", state.Current)
	}
}

// TestPostgresRefusesAForeignFirstWrite checks that two "first writes" cannot both
// succeed: the second one is refused by the primary key.
func TestPostgresRefusesAForeignFirstWrite(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 1}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 2}); err == nil {
		t.Fatal("a second first write was accepted")
	}
}

// TestPostgresRefusesANegativeCounter checks that an impossible counter is refused
// before it reaches the database.
func TestPostgresRefusesANegativeCounter(t *testing.T) {
	store := openTestStore(t)
	if err := store.Save(context.Background(), "order", 0, numbering.State{Current: -1}); err == nil {
		t.Fatal("a negative counter was accepted")
	}
}

// TestPostgresLoadRequiresARuleID checks that a missing identity is refused rather
// than read as some other rule.
func TestPostgresLoadRequiresARuleID(t *testing.T) {
	store := openTestStore(t)
	if _, _, err := store.Load(context.Background(), ""); err == nil {
		t.Fatal("a load without a rule id was accepted")
	}
}
