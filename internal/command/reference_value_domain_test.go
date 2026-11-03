package command

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func valueDomainWorld() ir.World {
	world := commandWorld()
	world.Semantics[0].Types = append(world.Semantics[0].Types, semantic.Type{
		ID:     "sales.status",
		Kind:   semantic.TypeAlias,
		Values: []string{"open", "closed"},
	})
	for i := range world.Semantics[0].Data {
		if world.Semantics[0].Data[i].ID == "sales.order" {
			world.Semantics[0].Data[i].Attributes = append(world.Semantics[0].Data[i].Attributes, semantic.Attribute{
				ID:      "sales.order.status",
				TypeRef: "sales.status",
			})
		}
	}
	world.Semantics[0].Capabilities[0].Inputs = append(world.Semantics[0].Capabilities[0].Inputs, semantic.Parameter{Name: "status", Type: "sales.status"})
	world.Semantics[0].Capabilities[0].Effect = &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	pattern := &world.Applications[0].Patterns[0]
	pattern.Interaction.Items = append(pattern.Interaction.Items, application.InteractionItem{Attribute: "sales.order.status"})
	pattern.Interaction.Commands[0].Inputs = append(pattern.Interaction.Commands[0].Inputs, application.CommandInputBinding{Name: "status", Attribute: "sales.order.status"})
	return world
}

func TestReferenceDataExecutorEnforcesValueDomain(t *testing.T) {
	store := NewMemoryStore()
	executor := ReferenceDataExecutor{World: valueDomainWorld(), Store: store}
	valid := request()
	valid.Inputs = map[string]string{"order": "ORD-1", "status": "open"}
	if _, err := executor.Execute(context.Background(), valid); err != nil {
		t.Fatalf("valid domain value was rejected: %v", err)
	}

	invalid := request()
	invalid.Inputs = map[string]string{"order": "ORD-2", "status": "unknown"}
	if _, err := executor.Execute(context.Background(), invalid); err == nil {
		t.Fatal("out-of-domain value was accepted")
	}
}
