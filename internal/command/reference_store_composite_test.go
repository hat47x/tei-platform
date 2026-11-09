package command

import (
	"context"
	"testing"

	"tei/internal/semantic"
)

func TestReferenceDataExecutorCompositeKey(t *testing.T) {
	world := valueDomainWorld()
	store := NewMemoryStore()
	executor := ReferenceDataExecutor{World: world, Store: store, KeyFields: []string{"order", "status"}}
	request := request()
	request.Inputs = map[string]string{"order": "ORD-1", "status": "open"}
	if _, err := executor.Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	key := recordKeyFields(request.Inputs, []string{"order", "status"})
	if _, ok := store.Get("sales.order", key); !ok {
		t.Fatalf("composite key %q is not stored", key)
	}
}

func TestMemoryStoreRecordsAreSorted(t *testing.T) {
	store := NewMemoryStore()
	effect := &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	if err := store.Apply(effect, map[string]string{"order": "2"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Apply(effect, map[string]string{"order": "1"}); err != nil {
		t.Fatal(err)
	}
	records := store.Records("sales.order")
	if len(records) != 2 || records[0]["order"] != "1" || records[1]["order"] != "2" {
		t.Fatalf("records=%+v", records)
	}
}
