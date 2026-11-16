package hook

import (
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/semantic"
)

// storeWithOrder puts one record in a store, with the attributes the command
// plan mapped, so ObserveFromStore has something to read.
func storeWithOrder(t *testing.T, attributes map[core.CanonicalID]string) *command.MemoryStore {
	t.Helper()
	store := command.NewMemoryStore()
	effect := &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	inputs := map[string]string{"order": "ORD-1"}
	if err := store.ApplyRecord(effect, "order=ORD-1", inputs, attributes); err != nil {
		t.Fatalf("apply: %v", err)
	}
	return store
}

func orderPattern() *application.Pattern {
	return &application.Pattern{ID: "order_entry.order_record", Data: "sales.order"}
}

// TestObserveFromStoreAnswersDeclaredReads checks the path this exists for: an
// after-change phase reads the record the command wrote, not the request.
func TestObserveFromStoreAnswersDeclaredReads(t *testing.T) {
	store := storeWithOrder(t, map[core.CanonicalID]string{
		"sales.order.order_id": "ORD-1",
		"sales.order.status":   "SHIPPED",
	})
	observed := ObserveFromStore(store, orderPattern(), []core.CanonicalID{
		"sales.order.order_id",
		"sales.order.status",
	})
	if observed["sales.order.order_id"] != "ORD-1" {
		t.Fatalf("order_id=%q", observed["sales.order.order_id"])
	}
	if observed["sales.order.status"] != "SHIPPED" {
		t.Fatalf("status=%q", observed["sales.order.status"])
	}
}

// TestObserveFromStoreOmitsUndeclaredAttributes checks that the store is not a
// way around the declaration: an attribute the hook did not name is not
// returned, even though the record holds it.
func TestObserveFromStoreOmitsUndeclaredAttributes(t *testing.T) {
	store := storeWithOrder(t, map[core.CanonicalID]string{
		"sales.order.order_id": "ORD-1",
		"sales.order.status":   "SHIPPED",
	})
	observed := ObserveFromStore(store, orderPattern(), []core.CanonicalID{"sales.order.order_id"})
	if _, held := observed["sales.order.status"]; held {
		t.Fatal("an undeclared attribute was observed")
	}
}

// TestObserveFromStoreLeavesUnstoredAttributesOut checks the split ReadsFor
// depends on: the attribute is absent, not empty. Omitting it is what makes
// ReadsFor report it as unobserved instead of handing over "".
func TestObserveFromStoreLeavesUnstoredAttributesOut(t *testing.T) {
	store := storeWithOrder(t, map[core.CanonicalID]string{"sales.order.order_id": "ORD-1"})
	observed := ObserveFromStore(store, orderPattern(), []core.CanonicalID{
		"sales.order.order_id",
		"sales.order.status",
	})
	if _, held := observed["sales.order.status"]; held {
		t.Fatalf("an attribute the record does not hold was observed as %q", observed["sales.order.status"])
	}
}

// TestObserveFromStoreWithoutARecordAnswersNothing checks the empty store: a
// run that applied no effect has nothing to read, and must not report values.
func TestObserveFromStoreWithoutARecordAnswersNothing(t *testing.T) {
	observed := ObserveFromStore(command.NewMemoryStore(), orderPattern(), []core.CanonicalID{"sales.order.order_id"})
	if len(observed) != 0 {
		t.Fatalf("observed %v from an empty store", observed)
	}
}
