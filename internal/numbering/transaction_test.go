package numbering_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/numbering"
)

// memoryTransaction is a transaction that records whether it finished and can be
// told to fail its commit, so the binding's ordering is observable.
type memoryTransaction struct {
	committed  bool
	rolledBack bool
	failCommit bool
}

func (t *memoryTransaction) Commit(context.Context) error {
	if t.failCommit {
		return errors.New("commit failed")
	}
	t.committed = true
	return nil
}

func (t *memoryTransaction) Rollback(context.Context) error {
	t.rolledBack = true
	return nil
}

// memoryUnitOfWork hands out the same transaction so a test can read its state.
type memoryUnitOfWork struct {
	transaction *memoryTransaction
	failBegin   bool
}

func (u *memoryUnitOfWork) Begin(context.Context) (numbering.Transaction, error) {
	if u.failBegin {
		return nil, errors.New("begin failed")
	}
	return u.transaction, nil
}

// TestReserveInTransactionCommitsBoth checks that a committed transaction keeps
// the counter consumed.
func TestReserveInTransactionCommitsBoth(t *testing.T) {
	store := numbering.NewMemoryCounterStore()
	transaction := &memoryTransaction{}
	unit := &memoryUnitOfWork{transaction: transaction}
	contract := numbering.Contract{Digits: 4, Prefix: "A-"}
	ctx := context.Background()
	reservation, err := contract.ReserveInTransaction(ctx, store, unit, "order")
	if err != nil {
		t.Fatal(err)
	}
	if reservation.Number() != "A-0001" {
		t.Fatalf("number=%q", reservation.Number())
	}
	if err := reservation.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if !transaction.committed {
		t.Fatal("the transaction was not committed")
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 1 {
		t.Fatalf("state=%+v", state)
	}
}

// TestReserveInTransactionRollbackReturnsTheCounter checks that rolling back
// returns the counter, so a failed caller does not leave a gap.
func TestReserveInTransactionRollbackReturnsTheCounter(t *testing.T) {
	store := numbering.NewMemoryCounterStore()
	transaction := &memoryTransaction{}
	unit := &memoryUnitOfWork{transaction: transaction}
	contract := numbering.Contract{Digits: 4}
	ctx := context.Background()
	reservation, err := contract.ReserveInTransaction(ctx, store, unit, "order")
	if err != nil {
		t.Fatal(err)
	}
	if err := reservation.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if !transaction.rolledBack {
		t.Fatal("the transaction was not rolled back")
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 0 {
		t.Fatalf("state=%+v", state)
	}
	next, err := contract.ReserveInTransaction(ctx, store, &memoryUnitOfWork{transaction: &memoryTransaction{}}, "order")
	if err != nil {
		t.Fatal(err)
	}
	if next.Number() != "0001" {
		t.Fatalf("next=%q", next.Number())
	}
}

// TestReserveInTransactionCommitFailureReturnsTheCounter checks that a failed
// transaction commit does not consume the number: the counter goes back and the
// caller sees the commit error.
func TestReserveInTransactionCommitFailureReturnsTheCounter(t *testing.T) {
	store := numbering.NewMemoryCounterStore()
	transaction := &memoryTransaction{failCommit: true}
	unit := &memoryUnitOfWork{transaction: transaction}
	contract := numbering.Contract{Digits: 4}
	ctx := context.Background()
	reservation, err := contract.ReserveInTransaction(ctx, store, unit, "order")
	if err != nil {
		t.Fatal(err)
	}
	if err := reservation.Commit(ctx); err == nil {
		t.Fatal("a failed transaction commit was reported as success")
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 0 {
		t.Fatalf("state=%+v", state)
	}
}

// TestReserveInTransactionRefusesAMissingUnitOfWork checks the configuration fault.
func TestReserveInTransactionRefusesAMissingUnitOfWork(t *testing.T) {
	contract := numbering.Contract{Digits: 4}
	ctx := context.Background()
	if _, err := contract.ReserveInTransaction(ctx, numbering.NewMemoryCounterStore(), nil, "order"); err == nil {
		t.Fatal("a missing unit of work was accepted")
	}
}

// TestReserveInTransactionRollsBackAFailedBegin checks that a begin failure
// reports before a reserve and that a reserve failure rolls the transaction back.
func TestReserveInTransactionRollsBackAFailedBegin(t *testing.T) {
	contract := numbering.Contract{Digits: 4}
	ctx := context.Background()
	if _, err := contract.ReserveInTransaction(ctx, numbering.NewMemoryCounterStore(), &memoryUnitOfWork{failBegin: true}, "order"); err == nil {
		t.Fatal("a begin failure was accepted")
	}
	// An invalid contract fails the reserve after the transaction began, so the
	// transaction must be rolled back instead of left open.
	bad := numbering.Contract{Digits: 0}
	transaction := &memoryTransaction{}
	if _, err := bad.ReserveInTransaction(ctx, numbering.NewMemoryCounterStore(), &memoryUnitOfWork{transaction: transaction}, "order"); err == nil {
		t.Fatal("an invalid contract was accepted")
	}
	if !transaction.rolledBack {
		t.Fatal("the transaction was left open after a reserve failure")
	}
}
