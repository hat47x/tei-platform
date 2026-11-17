package saveunit

import (
	"context"
	"strings"
	"testing"
)

// TestOneUnitRefusesTheSameKeyTwice checks the fail-closed choice for a key written
// twice in one unit.
//
// The store applies each occurrence in declaration order, so a delete followed by an
// update reads the delete as "no current value" and the row comes back partly
// filled. Planning cannot order something written twice, so the unit is refused
// whole rather than half-applied.
func TestOneUnitRefusesTheSameKeyTwice(t *testing.T) {
	store := newTestStore()
	adapter := &MemoryStoreAdapter{Store: store}
	if err := store.Apply(memoryEffect{Kind: ChangeKindCreate, Data: "sales.order"}, map[string]string{
		"sales.order.order_id": "ORD-1", "_key": "order=ORD-1",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes: []EntityChange{
			{Data: "sales.order", Kind: ChangeKindDelete, Key: "order=ORD-1"},
			{Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1",
				Fields: PatchSet{"sales.order.status": {Value: strp("open")}}},
		},
	}
	_, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter))).Save(context.Background(), unit)
	if err == nil {
		t.Fatal("a unit that writes one key twice was accepted")
	}
	if !strings.Contains(err.Error(), "SAVE_DUPLICATE_KEY") {
		t.Fatalf("the refusal does not name the cause: %v", err)
	}
	if !strings.Contains(err.Error(), "order=ORD-1") {
		t.Fatalf("the refusal does not name the key: %v", err)
	}
}

// TestDuplicateKeysWalksChildrenAndRelations checks that the detector looks where the
// store applies: entity children and relation children.
func TestDuplicateKeysWalksChildrenAndRelations(t *testing.T) {
	unit := &SaveUnit{Changes: []EntityChange{{
		Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1",
		Children: []EntityChange{
			{Data: "sales.line", Kind: ChangeKindUpdate, Key: "order=ORD-1/line=1"},
			{Data: "sales.line", Kind: ChangeKindDelete, Key: "order=ORD-1/line=1"},
		},
		Relations: []RelationChange{{
			Relation: "sales.order.lines",
			Children: []EntityChange{{Data: "sales.line", Kind: ChangeKindCreate, Key: "order=ORD-1/line=9"}},
		}},
	}}}
	duplicates := DuplicateKeys(unit)
	if len(duplicates) != 1 || duplicates[0] != "order=ORD-1/line=1" {
		t.Fatalf("duplicates=%v", duplicates)
	}
}
