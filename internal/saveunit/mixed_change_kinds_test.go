package saveunit

import (
	"context"
	"testing"

	"tei/internal/core"
)

// TestOneSaveUnitCarriesHeaderUpdateAndEveryDetailChange covers the X-TEA
// header-detail maintenance shape (XF310, knowledge inventory F-21).
//
// X-TEA kept each detail row as CURRENT, NEW or marked-for-delete and sorted the
// three into update, insert and delete at confirm time, so one confirmation carried
// all three kinds. A save unit carries the same thing through EntityChange.Kind and
// Children. What matters here is that the three kinds travel together and all land:
// a design that quietly applied only one kind would still look like a successful
// confirm.
func TestOneSaveUnitCarriesHeaderUpdateAndEveryDetailChange(t *testing.T) {
	store := newTestStore()
	adapter := &MemoryStoreAdapter{Store: store}
	// The rows a confirmation works from: one order, three lines.
	seed := func(kind ChangeKind, data core.CanonicalID, values map[string]string) {
		t.Helper()
		if err := store.Apply(memoryEffect{Kind: kind, Data: data}, values); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	seed(ChangeKindCreate, "sales.order", map[string]string{
		"sales.order.order_id": "ORD-1", "sales.order.customer": "C1", "sales.order.status": "open", "_key": "order=ORD-1",
	})
	for _, line := range []struct{ no, product, qty, key string }{
		{"1", "A", "2", "order=ORD-1/line=1"},
		{"2", "B", "3", "order=ORD-1/line=2"},
		{"3", "C", "4", "order=ORD-1/line=3"},
	} {
		seed(ChangeKindCreate, "sales.line", map[string]string{
			"sales.line.order_id": "ORD-1", "sales.line.line_no": line.no,
			"sales.line.product": line.product, "sales.line.qty": line.qty, "_key": line.key,
		})
	}

	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_lines",
		World:       orderLineWorld(),
		Changes: []EntityChange{{
			Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1",
			Fields: PatchSet{"sales.order.status": {Value: strp("confirmed")}},
			Children: []EntityChange{
				// CURRENT row whose quantity changed.
				{Data: "sales.line", Kind: ChangeKindUpdate, Key: "order=ORD-1/line=1",
					Fields: PatchSet{"sales.line.qty": {Value: strp("5")}}},
				// NEW row added by this confirmation.
				{Data: "sales.line", Kind: ChangeKindCreate, Key: "order=ORD-1/line=4",
					Fields: PatchSet{
						"sales.line.order_id": {Value: strp("ORD-1")},
						"sales.line.line_no":  {Value: strp("4")},
						"sales.line.product":  {Value: strp("D")},
						"sales.line.qty":      {Value: strp("1")},
					}},
				// Row marked for deletion.
				{Data: "sales.line", Kind: ChangeKindDelete, Key: "order=ORD-1/line=3"},
			},
		}},
	}

	if _, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter))).Save(context.Background(), unit); err != nil {
		t.Fatalf("one confirmation did not go through: %v", err)
	}

	snapshot, err := adapter.ReadSnapshot([]EntityKey{
		{Data: "sales.order", Key: "order=ORD-1"},
		{Data: "sales.line", Key: "order=ORD-1/line=1"},
		{Data: "sales.line", Key: "order=ORD-1/line=2"},
		{Data: "sales.line", Key: "order=ORD-1/line=3"},
		{Data: "sales.line", Key: "order=ORD-1/line=4"},
	})
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	value := func(key EntityKey, attribute string) *string {
		row := snapshot[key]
		if row == nil {
			return nil
		}
		return row[attribute]
	}
	if got := value(EntityKey{Data: "sales.order", Key: "order=ORD-1"}, "sales.order.status"); got == nil || *got != "confirmed" {
		t.Fatalf("the header update did not land: %v", got)
	}
	if got := value(EntityKey{Data: "sales.line", Key: "order=ORD-1/line=1"}, "sales.line.qty"); got == nil || *got != "5" {
		t.Fatalf("the detail update did not land: %v", got)
	}
	if got := value(EntityKey{Data: "sales.line", Key: "order=ORD-1/line=4"}, "sales.line.product"); got == nil || *got != "D" {
		t.Fatalf("the new detail row did not land: %v", got)
	}
	if got := value(EntityKey{Data: "sales.line", Key: "order=ORD-1/line=3"}, "sales.line.product"); got != nil {
		t.Fatalf("the deleted row is still there: %v", got)
	}
	// A row this confirmation did not touch must be untouched, or "one
	// confirmation" would mean "rewrite the page".
	if got := value(EntityKey{Data: "sales.line", Key: "order=ORD-1/line=2"}, "sales.line.qty"); got == nil || *got != "3" {
		t.Fatalf("an untouched row changed: %v", got)
	}
}
