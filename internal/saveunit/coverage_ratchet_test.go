package saveunit

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
)

// TestSaverRefusesAMissingStore checks the refusal a caller sees when no store is
// configured. The code was declared but no test named it, so the repository's
// coverage ratchet (tools/repository-plan-check) failed on it.
func TestSaverRefusesAMissingStore(t *testing.T) {
	_, err := (&Saver{}).Save(context.Background(), &SaveUnit{
		Changes: []EntityChange{{Data: "sales.order", Kind: ChangeKindCreate, Key: "ORD-1"}},
	})
	if err == nil {
		t.Fatal("a save with no store was accepted")
	}
	var saveErr *SaveError
	if !errors.As(err, &saveErr) {
		t.Fatalf("the refusal is not a SaveError: %v", err)
	}
	if saveErr.Code != "SAVE_STORE_UNAVAILABLE" {
		t.Fatalf("code=%s", saveErr.Code)
	}
}

// TestSaverRefusesAnUnknownChangeKind checks that a change kind the store cannot
// apply is reported by its own code, not folded into a generic failure.
func TestSaverRefusesAnUnknownChangeKind(t *testing.T) {
	adapter := &MemoryStoreAdapter{Store: staticMemoryStore{}}
	// The change-kind check runs inside the save transaction, so the saver needs the
	// in-memory unit of work the other saveunit tests use.
	saver := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter)), WithWorld(orderLineWorld()))
	_, err := saver.Save(context.Background(), &SaveUnit{
		Changes: []EntityChange{{Data: core.CanonicalID("sales.order"), Kind: ChangeKind("invented"), Key: "ORD-1"}},
	})
	if err == nil {
		t.Fatal("an unknown change kind was accepted")
	}
	var saveErr *SaveError
	if !errors.As(err, &saveErr) {
		t.Fatalf("the refusal is not a SaveError: %v", err)
	}
	if saveErr.Code != "SAVE_UNKNOWN_KIND" {
		t.Fatalf("code=%s", saveErr.Code)
	}
}

// staticMemoryStore is the smallest MemoryStore the adapter accepts, so a test can
// reach the change-kind check without describing a real store.
type staticMemoryStore struct{}

func (staticMemoryStore) Get(core.CanonicalID, string) (map[string]string, bool) { return nil, false }
func (staticMemoryStore) Apply(memoryEffect, map[string]string) error            { return nil }

// TestSaverReportsATransactionBeginFailure checks that a unit of work which cannot
// start is reported by its own code, not as an unknown failure.
func TestSaverReportsATransactionBeginFailure(t *testing.T) {
	adapter := &MemoryStoreAdapter{Store: staticMemoryStore{}}
	saver := NewSaver(
		WithStore(adapter),
		WithUnitOfWork(UnitOfWorkTx{Start: func(context.Context) (Transaction, error) {
			return nil, errors.New("the transaction could not start")
		}}),
		WithWorld(orderLineWorld()),
	)
	_, err := saver.Save(context.Background(), &SaveUnit{
		Changes: []EntityChange{{Data: core.CanonicalID("sales.order"), Kind: ChangeKindCreate, Key: "ORD-1"}},
	})
	if err == nil {
		t.Fatal("a save whose transaction could not start was accepted")
	}
	var saveErr *SaveError
	if !errors.As(err, &saveErr) {
		t.Fatalf("the refusal is not a SaveError: %v", err)
	}
	if saveErr.Code != "SAVE_TRANSACTION_BEGIN_FAILED" {
		t.Fatalf("code=%s", saveErr.Code)
	}
}
