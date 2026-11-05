package command

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func normalizedKeyWorld() ir.World {
	world := valueDomainWorld()
	for i := range world.Semantics[0].Data {
		if world.Semantics[0].Data[i].ID != "sales.order" {
			continue
		}
		world.Semantics[0].Data[i].Attributes = append(world.Semantics[0].Data[i].Attributes, semantic.Attribute{
			ID:      "sales.order.code",
			TypeRef: "std.integer",
		})
		world.Semantics[0].Data[i].CandidateKeys = []semantic.CandidateKey{{
			ID:         "sales.order.code",
			Attributes: []core.CanonicalID{"sales.order.code"},
		}}
	}
	capability := &world.Semantics[0].Capabilities[0]
	capability.Inputs = append(capability.Inputs, semantic.Parameter{Name: "code", Type: "std.integer"})
	pattern := &world.Applications[0].Patterns[0]
	pattern.Interaction.Items = append(pattern.Interaction.Items, application.InteractionItem{Attribute: "sales.order.code"})
	pattern.Interaction.Commands[0].Inputs = append(pattern.Interaction.Commands[0].Inputs, application.CommandInputBinding{Name: "code", Attribute: "sales.order.code"})
	return world
}

func TestReferenceDataExecutorNormalizesCandidateKey(t *testing.T) {
	executor := ReferenceDataExecutor{World: normalizedKeyWorld(), Store: NewMemoryStore()}
	first := request()
	first.Inputs = map[string]string{"order": "ORD-1", "status": "open", "code": "1"}
	if _, err := executor.Execute(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	duplicate := request()
	duplicate.Inputs = map[string]string{"order": "ORD-2", "status": "closed", "code": "01"}
	if _, err := executor.Execute(context.Background(), duplicate); err == nil {
		t.Fatal("normalized duplicate candidate key was accepted")
	}
}

func TestNormalizeValueDateTime(t *testing.T) {
	types := map[core.CanonicalID]semantic.Type{
		"std.datetime": {ID: "std.datetime", Kind: semantic.TypePrimitive},
		"std.time":     {ID: "std.time", Kind: semantic.TypePrimitive},
	}
	if normalizeValue("std.datetime", "2026-01-01T00:00:00Z", types) != normalizeValue("std.datetime", "2026-01-01T09:00:00+09:00", types) {
		t.Fatal("equivalent datetimes were not normalized")
	}
	if normalizeValue("std.time", "09:00:00", types) != normalizeValue("std.time", "09:00:00", types) {
		t.Fatal("times were not normalized")
	}
}
