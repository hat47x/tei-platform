package command

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func cascadeNoteWorld() ir.World {
	world := keyedPolicyWorld("cascade")
	world.Semantics[0].Data = append(world.Semantics[0].Data, semantic.Data{
		ID: "sales.note",
		Attributes: []semantic.Attribute{{
			ID:      "sales.note.line_no",
			TypeRef: "std.identifier",
		}},
		Relations: []semantic.Relation{{
			ID:     "sales.note.line",
			Target: "sales.line",
			Keys: []semantic.RelationKey{{
				SourceAttribute: "sales.note.line_no",
				TargetAttribute: "sales.line.order_id",
			}},
			OnDelete: "cascade",
		}},
	})
	world.Semantics[0].Capabilities = append(world.Semantics[0].Capabilities, semantic.Capability{
		ID:       "sales.create_note",
		Symbol:   "create_note",
		Inputs:   []semantic.Parameter{{Name: "note_line", Type: "std.identifier"}},
		Outcomes: []semantic.Outcome{{Name: "created", Payload: "sales.note"}},
		Effect:   &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.note"},
	})
	world.Applications[0].Patterns = append(world.Applications[0].Patterns, application.Pattern{
		ID:   "order_entry.note_record",
		Kind: application.PatternRecord,
		Data: "sales.note",
		Interaction: &application.Interaction{
			Items: []application.InteractionItem{{Attribute: "sales.note.line_no"}},
			Commands: []application.InteractionCommand{{
				Capability: "sales.create_note",
				Inputs:     []application.CommandInputBinding{{Name: "note_line", Attribute: "sales.note.line_no"}},
			}},
		},
	})
	return world
}

func TestKeyedCascadeDeletesDescendants(t *testing.T) {
	store := NewMemoryStore()
	executor := ReferenceDataExecutor{World: cascadeNoteWorld(), Store: store}
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
	if len(store.Records("sales.note")) != 1 {
		t.Fatal("note was not created")
	}
	if _, err := executor.Execute(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(store.Records("sales.order")) != 0 || len(store.Records("sales.line")) != 0 || len(store.Records("sales.note")) != 0 {
		t.Fatalf("orders=%d lines=%d notes=%d",
			len(store.Records("sales.order")), len(store.Records("sales.line")), len(store.Records("sales.note")))
	}
}
