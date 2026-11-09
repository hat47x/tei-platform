package numberingcommand_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/command"
	"tei/internal/numbering"
	"tei/internal/numberingcommand"
)

// recordingTransaction is the Command boundary's transaction, and it records
// whether it finished so the adapter's delegation is observable.
type recordingTransaction struct {
	committed  bool
	rolledBack bool
}

func (t *recordingTransaction) Commit(context.Context) error {
	t.committed = true
	return nil
}

func (t *recordingTransaction) Rollback(context.Context) error {
	t.rolledBack = true
	return nil
}

// recordingUnitOfWork is the Command boundary's UnitOfWork.
type recordingUnitOfWork struct {
	transaction *recordingTransaction
}

func (u recordingUnitOfWork) Begin(context.Context) (command.Transaction, error) {
	return u.transaction, nil
}

// TestAdapterCommitsTheCommandTransaction checks that a numbering reservation
// committed through the adapter commits the Command transaction once.
func TestAdapterCommitsTheCommandTransaction(t *testing.T) {
	transaction := &recordingTransaction{}
	unit := numberingcommand.UnitOfWork{Inner: recordingUnitOfWork{transaction: transaction}}
	store := numbering.NewMemoryCounterStore()
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
	if !transaction.committed || transaction.rolledBack {
		t.Fatalf("transaction=%+v", transaction)
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 1 {
		t.Fatalf("state=%+v", state)
	}
}

// TestAdapterRollsBackAndReturnsTheCounter checks that a rollback returns the
// counter as well, so one transaction covers both sides.
func TestAdapterRollsBackAndReturnsTheCounter(t *testing.T) {
	transaction := &recordingTransaction{}
	unit := numberingcommand.UnitOfWork{Inner: recordingUnitOfWork{transaction: transaction}}
	store := numbering.NewMemoryCounterStore()
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
		t.Fatal("the command transaction was not rolled back")
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 0 {
		t.Fatalf("state=%+v", state)
	}
}

// TestAdapterReportsABeginFailure checks that a Command begin failure is passed
// through, so a caller does not mistake it for a numbering fault.
func TestAdapterReportsABeginFailure(t *testing.T) {
	unit := numberingcommand.UnitOfWork{Inner: failingUnitOfWork{}}
	if _, err := unit.Begin(context.Background()); err == nil {
		t.Fatal("a begin failure was accepted")
	}
}

type failingUnitOfWork struct{}

func (failingUnitOfWork) Begin(context.Context) (command.Transaction, error) {
	return nil, errors.New("begin failed")
}
