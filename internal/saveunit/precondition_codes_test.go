package saveunit

import (
	"context"
	"errors"
	"testing"
)

// Three refusal codes answer "the saver cannot run" questions. A code that no
// test names is a rule nothing proves can fire, so each is produced here by
// withholding one precondition (#4032).
func TestSaverPreconditionCodesAreProduced(t *testing.T) {
	t.Run("no store configured", func(t *testing.T) {
		_, err := NewSaver().Save(context.Background(), &SaveUnit{Changes: []EntityChange{{}}})
		assertSaveCode(t, err, "SAVE_STORE_UNAVAILABLE")
	})

	t.Run("transaction begin fails", func(t *testing.T) {
		store := newTestStore()
		adapter := &MemoryStoreAdapter{Store: store}
		saver := NewSaver(WithStore(adapter), WithUnitOfWork(UnitOfWorkTx{Start: func(context.Context) (Transaction, error) { return nil, errors.New("begin failed") }}))
		_, err := saver.Save(context.Background(), &SaveUnit{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			World:       orderLineWorld(),
			Changes: []EntityChange{{
				Data: "sales.order", Kind: ChangeKindCreate, Key: "order=ORD-1",
				Fields: PatchSet{"sales.order.order_id": {Value: strp("ORD-1")}},
			}},
		})
		assertSaveCode(t, err, "SAVE_TRANSACTION_BEGIN_FAILED")
	})

	t.Run("unknown change kind", func(t *testing.T) {
		adapter := &MemoryStoreAdapter{Store: newTestStore()}
		_, _, err := adapter.ApplyChanges(context.Background(), &SaveUnit{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			World:       orderLineWorld(),
			Changes:     []EntityChange{{Data: "sales.order", Kind: "reticulate", Key: "order=ORD-1"}},
		}, &InMemoryTransaction{adapter: adapter})
		assertSaveCode(t, err, "SAVE_UNKNOWN_KIND")
	})
}

func assertSaveCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got no error", want)
	}
	var saveErr *SaveError
	if !errors.As(err, &saveErr) {
		t.Fatalf("expected a save error with code %s, got %v", want, err)
	}
	if saveErr.Code != want {
		t.Fatalf("expected code %s, got %s (%v)", want, saveErr.Code, err)
	}
}
