package command

import (
	"context"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func requiredWorld() ir.World {
	world := commandWorld()
	world.Semantics[0].Capabilities[0].Effect = &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	for i := range world.Semantics[0].Data {
		if world.Semantics[0].Data[i].ID != "sales.order" {
			continue
		}
		for j := range world.Semantics[0].Data[i].Attributes {
			if world.Semantics[0].Data[i].Attributes[j].ID == "sales.order.order_id" {
				world.Semantics[0].Data[i].Attributes[j].Required = true
			}
		}
	}
	return world
}

func TestReferenceDataExecutorRequiresAttributes(t *testing.T) {
	store := NewMemoryStore()
	executor := ReferenceDataExecutor{World: requiredWorld(), Store: store}
	if _, err := executor.Execute(context.Background(), request()); err != nil {
		t.Fatalf("provided required attribute was rejected: %v", err)
	}

	missing := request()
	missing.Inputs = nil
	if _, err := executor.Execute(context.Background(), missing); err == nil {
		t.Fatal("missing required attribute was accepted")
	}
}
