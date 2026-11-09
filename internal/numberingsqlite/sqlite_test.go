package numberingsqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"tei/internal/numbering"
	"tei/internal/numberingsqlite"
)

func openStore(t *testing.T) *numberingsqlite.Store {
	t.Helper()
	store, err := numberingsqlite.Open(filepath.Join(t.TempDir(), "counter.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// TestStoreStartsAtZero checks that a rule with no row reads as zero at version
// zero, which is the state a rule starts in.
func TestStoreStartsAtZero(t *testing.T) {
	store := openStore(t)
	state, version, err := store.Load(context.Background(), "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 0 || version != 0 {
		t.Fatalf("state=%+v version=%d", state, version)
	}
}

// TestReserveAcrossRestarts checks that a number issued through the store survives
// a reopen, so a restart does not reissue the same number.
func TestReserveAcrossRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "counter.db")
	ctx := context.Background()
	contract := numbering.Contract{Digits: 5, Prefix: "ORD-"}

	first, err := numberingsqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := contract.ReserveInTransaction(ctx, first, plainUnitOfWork{}, "order")
	if err != nil {
		t.Fatal(err)
	}
	if reservation.Number() != "ORD-00001" {
		t.Fatalf("number=%q", reservation.Number())
	}
	if err := reservation.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := numberingsqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	state, version, err := second.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 1 || version != 1 {
		t.Fatalf("state=%+v version=%d", state, version)
	}
	next, err := contract.ReserveInTransaction(ctx, second, plainUnitOfWork{}, "order")
	if err != nil {
		t.Fatal(err)
	}
	if next.Number() != "ORD-00002" {
		t.Fatalf("next=%q", next.Number())
	}
}

// TestSaveRefusesAStaleVersion checks that a write against an old version is
// reported as a conflict, not applied. X-TEA's optimistic update result was never
// inspected (D:Session.java L1995–2002).
func TestSaveRefusesAStaleVersion(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 10}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 20}); !errors.Is(err, numbering.ErrNumberingConflict) {
		t.Fatalf("err=%v", err)
	}
	state, version, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 10 || version != 1 {
		t.Fatalf("state=%+v version=%d", state, version)
	}
}

// TestSaveRefusesAForeignFirstWrite checks that a first write for a rule that
// already exists is a conflict, so two callers cannot both create it.
func TestSaveRefusesAForeignFirstWrite(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 5}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 7}); !errors.Is(err, numbering.ErrNumberingConflict) {
		t.Fatalf("err=%v", err)
	}
}

// TestSaveRefusesANegativeCounter checks that a negative counter never reaches the
// table.
func TestSaveRefusesANegativeCounter(t *testing.T) {
	store := openStore(t)
	if err := store.Save(context.Background(), "order", 0, numbering.State{Current: -1}); err == nil {
		t.Fatal("a negative counter was stored")
	}
}

// TestLoadRequiresARuleID checks the argument guard.
func TestLoadRequiresARuleID(t *testing.T) {
	store := openStore(t)
	if _, _, err := store.Load(context.Background(), ""); err == nil {
		t.Fatal("an empty rule id was accepted")
	}
	if err := store.Save(context.Background(), "", 0, numbering.State{Current: 1}); err == nil {
		t.Fatal("an empty rule id was accepted by Save")
	}
}

// plainUnitOfWork is a transaction that does nothing, so the reservation's own
// store write is what the test observes.
type plainUnitOfWork struct{}

func (plainUnitOfWork) Begin(context.Context) (numbering.Transaction, error) {
	return plainTransaction{}, nil
}

type plainTransaction struct{}

func (plainTransaction) Commit(context.Context) error   { return nil }
func (plainTransaction) Rollback(context.Context) error { return nil }
