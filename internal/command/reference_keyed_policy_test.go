package command

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func keyedPolicyWorld(policy string) ir.World {
	world := commandWorld()
	for i := range world.Semantics[0].Data {
		if world.Semantics[0].Data[i].ID != "sales.order" {
			continue
		}
		world.Semantics[0].Data[i].Attributes = append(world.Semantics[0].Data[i].Attributes, semantic.Attribute{
			ID:      "sales.order.order_id",
			TypeRef: "std.identifier",
		})
	}
	world.Semantics[0].Data = append(world.Semantics[0].Data, semantic.Data{
		ID: "sales.line",
		Attributes: []semantic.Attribute{{
			ID:      "sales.line.order_id",
			TypeRef: "std.identifier",
		}},
		Relations: []semantic.Relation{{
			ID:     "sales.line.order",
			Target: "sales.order",
			Keys: []semantic.RelationKey{{
				SourceAttribute: "sales.line.order_id",
				TargetAttribute: "sales.order.order_id",
			}},
			OnDelete: policy,
		}},
	})
	world.Semantics[0].Capabilities = append(world.Semantics[0].Capabilities, semantic.Capability{
		ID:       "sales.create_line",
		Symbol:   "create_line",
		Inputs:   []semantic.Parameter{{Name: "line_order", Type: "std.identifier"}},
		Outcomes: []semantic.Outcome{{Name: "created", Payload: "sales.line"}},
		Effect:   &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.line"},
	})
	world.Semantics[0].Capabilities[0].Effect = &semantic.DataEffect{Kind: semantic.DataEffectDelete, Data: "sales.order"}
	world.Applications[0].Patterns = append(world.Applications[0].Patterns, application.Pattern{
		ID:   "order_entry.line_record",
		Kind: application.PatternRecord,
		Data: "sales.line",
		Interaction: &application.Interaction{
			Items: []application.InteractionItem{{Attribute: "sales.line.order_id"}},
			Commands: []application.InteractionCommand{{
				Capability: "sales.create_line",
				Inputs:     []application.CommandInputBinding{{Name: "line_order", Attribute: "sales.line.order_id"}},
			}},
		},
	})
	return world
}

func seedOrder(t *testing.T, store *MemoryStore, orderID string) {
	t.Helper()
	if err := store.ApplyWithKey(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}, "order="+orderID, map[string]string{"order": orderID}); err != nil {
		t.Fatal(err)
	}
}

func createLine(t *testing.T, executor ReferenceDataExecutor, orderID string) {
	t.Helper()
	_, err := executor.Execute(context.Background(), Request{
		Application: "order_entry",
		Pattern:     "order_entry.line_record",
		Capability:  "sales.create_line",
		Inputs:      map[string]string{"line_order": orderID},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKeyedRestrictBlocksMatchingLine(t *testing.T) {
	store := NewMemoryStore()
	executor := ReferenceDataExecutor{World: keyedPolicyWorld("restrict"), Store: store}
	seedOrder(t, store, "ORD-1")
	createLine(t, executor, "ORD-1")
	if _, err := executor.Execute(context.Background(), request()); err == nil {
		t.Fatal("matching line did not restrict delete")
	}
	if len(store.Records("sales.line")) != 1 {
		t.Fatal("restricted delete removed the line")
	}
}

func TestKeyedCascadeDeletesMatchingLine(t *testing.T) {
	store := NewMemoryStore()
	executor := ReferenceDataExecutor{World: keyedPolicyWorld("cascade"), Store: store}
	seedOrder(t, store, "ORD-1")
	createLine(t, executor, "ORD-1")
	if _, err := executor.Execute(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(store.Records("sales.line")) != 0 || len(store.Records("sales.order")) != 0 {
		t.Fatalf("lines=%d orders=%d", len(store.Records("sales.line")), len(store.Records("sales.order")))
	}
}

func TestKeyedRestrictAllowsNonMatchingLine(t *testing.T) {
	store := NewMemoryStore()
	executor := ReferenceDataExecutor{World: keyedPolicyWorld("restrict"), Store: store}
	seedOrder(t, store, "ORD-1")
	createLine(t, executor, "ORD-2")
	if _, err := executor.Execute(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(store.Records("sales.line")) != 1 {
		t.Fatal("non-matching line was removed")
	}
}
