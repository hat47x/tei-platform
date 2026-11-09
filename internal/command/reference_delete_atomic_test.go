package command

import (
	"context"
	"testing"
)

func TestReferenceDeleteRestoresStoreOnDeepRestrict(t *testing.T) {
	world := cascadeNoteWorld()
	for i := range world.Semantics[0].Data {
		if world.Semantics[0].Data[i].ID != "sales.note" {
			continue
		}
		for j := range world.Semantics[0].Data[i].Relations {
			world.Semantics[0].Data[i].Relations[j].OnDelete = "restrict"
		}
	}
	store := NewMemoryStore()
	executor := ReferenceDataExecutor{World: world, Store: store}
	seedOrder(t, store, "ORD-1")
	createLine(t, executor, "ORD-1")
	if _, err := executor.Execute(context.Background(), Request{
		Application: "order_entry",
		Pattern:     "order_entry.note_record",
		Capability:  "sales.create_note",
		Inputs:      map[string]string{"note_line": "ORD-1"},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := executor.Execute(context.Background(), request()); err == nil {
		t.Fatal("deep restrict did not reject the delete")
	}
	if len(store.Records("sales.order")) != 1 || len(store.Records("sales.line")) != 1 || len(store.Records("sales.note")) != 1 {
		t.Fatalf("partial delete was not rolled back: orders=%d lines=%d notes=%d",
			len(store.Records("sales.order")), len(store.Records("sales.line")), len(store.Records("sales.note")))
	}
}
