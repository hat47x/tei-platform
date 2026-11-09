package numbering_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/numbering"
)

// TestReserveIssuesAndKeepsTheCounter checks the happy path: the number is
// issued, the counter moves, and Commit keeps it consumed.
func TestReserveIssuesAndKeepsTheCounter(t *testing.T) {
	store := numbering.NewMemoryCounterStore()
	contract := numbering.Contract{Digits: 5, Prefix: "ORD-"}
	ctx := context.Background()
	first, reservation, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	if first != "ORD-00001" || reservation.Number() != first {
		t.Fatalf("first=%q number=%q", first, reservation.Number())
	}
	if err := reservation.Commit(); err != nil {
		t.Fatal(err)
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 1 {
		t.Fatalf("state=%+v", state)
	}
	second, _, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	if second != "ORD-00002" {
		t.Fatalf("second=%q", second)
	}
}

// TestReleaseReturnsTheCounter checks that a caller that cannot use the number
// returns the counter, so the next caller gets the same number. X-TEA left a gap
// and returned an empty string after the counter may already have moved
// (D:Session.java L2004–2007).
func TestReleaseReturnsTheCounter(t *testing.T) {
	store := numbering.NewMemoryCounterStore()
	contract := numbering.Contract{Digits: 3}
	ctx := context.Background()
	first, reservation, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	if first != "001" {
		t.Fatalf("first=%q", first)
	}
	if err := reservation.Release(ctx); err != nil {
		t.Fatal(err)
	}
	again, _, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	if again != "001" {
		t.Fatalf("again=%q", again)
	}
}

// TestReserveReportsAConcurrentChange checks that a counter that moved between the
// read and the write is reported, not overwritten. X-TEA's optimistic update
// result was never inspected (同 L1995–2002).
func TestReserveReportsAConcurrentChange(t *testing.T) {
	store := numbering.NewMemoryCounterStore()
	contract := numbering.Contract{Digits: 3}
	ctx := context.Background()
	_, first, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	// Another caller issues the next number before the first one finishes.
	_, second, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	// Releasing the first reservation must not overwrite the second issuance.
	releaseErr := first.Release(ctx)
	if !errors.Is(releaseErr, numbering.ErrNumberingConflict) {
		t.Fatalf("err=%v", releaseErr)
	}
	if second.Number() != "002" {
		t.Fatalf("second=%q", second.Number())
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 2 {
		t.Fatalf("state=%+v", state)
	}
}

// TestReleaseAfterCommitIsRefused checks that a finished reservation cannot be
// released, so a caller cannot return a number it already used.
func TestReleaseAfterCommitIsRefused(t *testing.T) {
	store := numbering.NewMemoryCounterStore()
	contract := numbering.Contract{Digits: 3}
	ctx := context.Background()
	_, reservation, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	if err := reservation.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := reservation.Release(ctx); err == nil {
		t.Fatal("a committed reservation was released")
	}
	if err := reservation.Commit(); err == nil {
		t.Fatal("a committed reservation was committed twice")
	}
}

// TestReserveRefusesAMissingStoreOrBadContract checks the configuration faults.
func TestReserveRefusesAMissingStoreOrBadContract(t *testing.T) {
	ctx := context.Background()
	contract := numbering.Contract{Digits: 3}
	if _, _, err := contract.Reserve(ctx, nil, "order"); err == nil {
		t.Fatal("a missing store was accepted")
	}
	bad := numbering.Contract{Digits: 0}
	if _, _, err := bad.Reserve(ctx, numbering.NewMemoryCounterStore(), "order"); err == nil {
		t.Fatal("an invalid contract was accepted")
	}
}
