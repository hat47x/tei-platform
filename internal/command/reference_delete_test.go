package command

import (
	"context"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func deletePolicyWorld(policy string) ir.World {
	world := commandWorld()
	world.Semantics[0].Data = append(world.Semantics[0].Data, semantic.Data{
		ID: "sales.line",
		Relations: []semantic.Relation{{
			ID:       "sales.line.order",
			Target:   "sales.order",
			OnDelete: policy,
		}},
	})
	world.Semantics[0].Capabilities[0].Effect = &semantic.DataEffect{Kind: semantic.DataEffectDelete, Data: "sales.order"}
	return world
}

func seedDeleteStore(t *testing.T, store *MemoryStore) {
	t.Helper()
	if err := store.Apply(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.line"}, map[string]string{"id": "L1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Apply(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}, map[string]string{"order": "ORD-1"}); err != nil {
		t.Fatal(err)
	}
}

func TestReferenceDataExecutorRestrictsDelete(t *testing.T) {
	store := NewMemoryStore()
	seedDeleteStore(t, store)
	executor := ReferenceDataExecutor{World: deletePolicyWorld("restrict"), Store: store}
	_, err := executor.Execute(context.Background(), request())
	if err == nil {
		t.Fatal("restricted delete was accepted")
	}
	if len(store.Records("sales.line")) != 1 {
		t.Fatal("restricted delete removed lines")
	}
}

func TestReferenceDataExecutorCascadesDelete(t *testing.T) {
	store := NewMemoryStore()
	seedDeleteStore(t, store)
	executor := ReferenceDataExecutor{World: deletePolicyWorld("cascade"), Store: store}
	if _, err := executor.Execute(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(store.Records("sales.line")) != 0 || len(store.Records("sales.order")) != 0 {
		t.Fatalf("lines=%d orders=%d", len(store.Records("sales.line")), len(store.Records("sales.order")))
	}
}

func TestReferenceDataExecutorPreservesDelete(t *testing.T) {
	store := NewMemoryStore()
	seedDeleteStore(t, store)
	executor := ReferenceDataExecutor{World: deletePolicyWorld("preserve"), Store: store}
	if _, err := executor.Execute(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(store.Records("sales.line")) != 1 || len(store.Records("sales.order")) != 0 {
		t.Fatalf("lines=%d orders=%d", len(store.Records("sales.line")), len(store.Records("sales.order")))
	}
}
