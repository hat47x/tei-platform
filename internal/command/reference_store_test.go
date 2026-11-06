package command

import (
	"context"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func referenceWorld(effect semantic.DataEffectKind) ir.World {
	world := commandWorld()
	world.Semantics[0].Capabilities[0].Effect = &semantic.DataEffect{Kind: effect, Data: "sales.order"}
	return world
}

func TestReferenceDataExecutorCreatesRecord(t *testing.T) {
	store := NewMemoryStore()
	invoker := Invoker{
		World:      referenceWorld(semantic.DataEffectCreate),
		Executor:   ReferenceDataExecutor{World: referenceWorld(semantic.DataEffectCreate), Store: store},
		UnitOfWork: &StoreUnitOfWork{Store: store},
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	records := store.Records("sales.order")
	if len(records) != 1 || records[0]["order"] != "ORD-1" {
		t.Fatalf("records=%+v", records)
	}
}

func TestReferenceStoreRollsBackOnCommitFailure(t *testing.T) {
	store := NewMemoryStore()
	world := referenceWorld(semantic.DataEffectCreate)
	invoker := Invoker{
		World:      world,
		Executor:   ReferenceDataExecutor{World: world, Store: store},
		UnitOfWork: &StoreUnitOfWork{Store: store, FailCommit: true},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_COMMIT_FAILED" {
		t.Fatalf("code=%s", code)
	}
	if records := store.Records("sales.order"); len(records) != 0 {
		t.Fatalf("rollback did not restore the store: %+v", records)
	}
}

func TestReferenceDataExecutorDeletesRecord(t *testing.T) {
	store := NewMemoryStore()
	create := referenceWorld(semantic.DataEffectCreate)
	if _, err := (Invoker{
		World:      create,
		Executor:   ReferenceDataExecutor{World: create, Store: store},
		UnitOfWork: &StoreUnitOfWork{Store: store},
	}).Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	remove := referenceWorld(semantic.DataEffectDelete)
	if _, err := (Invoker{
		World:      remove,
		Executor:   ReferenceDataExecutor{World: remove, Store: store},
		UnitOfWork: &StoreUnitOfWork{Store: store},
	}).Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if records := store.Records("sales.order"); len(records) != 0 {
		t.Fatalf("records=%+v", records)
	}
}
