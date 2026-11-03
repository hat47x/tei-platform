package command

import (
	"context"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func immutableWorld() ir.World {
	world := valueDomainWorld()
	world.Semantics[0].Capabilities[0].Effect = &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}
	for i := range world.Semantics[0].Data {
		if world.Semantics[0].Data[i].ID != "sales.order" {
			continue
		}
		for j := range world.Semantics[0].Data[i].Attributes {
			if world.Semantics[0].Data[i].Attributes[j].ID == "sales.order.status" {
				world.Semantics[0].Data[i].Attributes[j].Immutable = true
			}
		}
	}
	return world
}

func TestReferenceDataExecutorEnforcesImmutableAttribute(t *testing.T) {
	store := NewMemoryStore()
	if err := store.ApplyWithKey(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}, "order=ORD-1", map[string]string{"order": "ORD-1", "status": "open"}); err != nil {
		t.Fatal(err)
	}
	executor := ReferenceDataExecutor{World: immutableWorld(), Store: store, KeyFields: []string{"order"}}

	valid := request()
	valid.Inputs = map[string]string{"order": "ORD-1", "status": "open"}
	if _, err := executor.Execute(context.Background(), valid); err != nil {
		t.Fatalf("unchanged immutable value was rejected: %v", err)
	}

	invalid := request()
	invalid.Inputs = map[string]string{"order": "ORD-1", "status": "closed"}
	if _, err := executor.Execute(context.Background(), invalid); err == nil {
		t.Fatal("changed immutable value was accepted")
	}
}
