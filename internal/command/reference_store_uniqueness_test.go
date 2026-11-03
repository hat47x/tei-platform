package command

import (
	"context"
	"testing"

	"tei/internal/semantic"
)

func TestMemoryStoreRejectsDuplicateCreate(t *testing.T) {
	store := NewMemoryStore()
	effect := &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	inputs := map[string]string{"order": "ORD-1"}
	if err := store.Apply(effect, inputs); err != nil {
		t.Fatal(err)
	}
	if err := store.Apply(effect, inputs); err == nil {
		t.Fatal("duplicate create was accepted")
	}
}

func TestReferenceDataExecutorResolvesAliasValueDomain(t *testing.T) {
	world := valueDomainWorld()
	for i := range world.Semantics[0].Types {
		if world.Semantics[0].Types[i].ID == "sales.status" {
			world.Semantics[0].Types[i].Values = nil
			world.Semantics[0].Types[i].Underlying = "std.code"
		}
	}
	world.Semantics[0].Types = append(world.Semantics[0].Types, semantic.Type{
		ID:     "std.code",
		Kind:   semantic.TypePrimitive,
		Values: []string{"open", "closed"},
	})
	executor := ReferenceDataExecutor{World: world, Store: NewMemoryStore()}
	invalid := request()
	invalid.Inputs = map[string]string{"order": "ORD-3", "status": "unknown"}
	if _, err := executor.Execute(context.Background(), invalid); err == nil {
		t.Fatal("alias value domain was not enforced")
	}
}
