package applicationview_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func viewWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Capabilities: []semantic.Capability{
					{ID: "sales.confirm_order", Symbol: "confirm_order",
						Inputs:   []semantic.Parameter{{Name: "order_id", Type: "std.identifier"}},
						Outcomes: []semantic.Outcome{{Name: "confirmed", Payload: "sales.order_confirmed"}},
						Effect:   &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
					},
				},
				Data: []semantic.Data{{
					ID: "sales.order",
					Attributes: []semantic.Attribute{
						{ID: "sales.order.order_id", Symbol: "order_id", Display: "受注番号", TypeRef: "std.identifier", Required: true, Immutable: true},
						{ID: "sales.order.customer_id", TypeRef: "customer.customer_id"},
					},
					Relations: []semantic.Relation{{
						ID:          "sales.order.customer",
						Target:      "customer.customer",
						Cardinality: semantic.CardinalityOne,
					}},
				}},
			},
			{
				ID: "customer",
				Data: []semantic.Data{{
					ID: "customer.customer",
					Attributes: []semantic.Attribute{
						{ID: "customer.customer.customer_id", TypeRef: "customer.customer_id"},
						{ID: "customer.customer.name", TypeRef: "std.text"},
					},
				}},
			},
		},
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{
				{ID: "order_entry.recent_orders", Kind: application.PatternBrowse, Data: "sales.order"},
				{ID: "order_entry.customer_reference", Kind: application.PatternReference, Relation: "sales.order.customer"},
				{
					ID:   "order_entry.order_record",
					Kind: application.PatternRecord,
					Data: "sales.order",
					Interaction: &application.Interaction{
						Items: []application.InteractionItem{
							{Attribute: "sales.order.order_id"},
							{
								Attribute: "sales.order.customer_id",
								Lookup: &application.InteractionLookup{
									Pattern:         "order_entry.customer_reference",
									TargetAttribute: "customer.customer.customer_id",
								},
							},
						},
						Navigations: []application.InteractionNavigation{{From: "order_entry.recent_orders"}},
						Commands:    []application.InteractionCommand{{Capability: "sales.confirm_order"}},
					},
				},
			},
		}},
	}
}

func TestBuildResolvesInteractionItemsAndNavigation(t *testing.T) {
	view, err := applicationview.Build(viewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if view.SchemaVersion != applicationview.SchemaVersion || view.Kind != application.PatternRecord || view.Data != "sales.order" {
		t.Fatalf("view=%+v", view)
	}
	if len(view.Items) != 2 {
		t.Fatalf("items=%+v", view.Items)
	}
	if view.Items[0].Attribute != "sales.order.order_id" || view.Items[0].Type != "std.identifier" || view.Items[0].Lookup != nil {
		t.Fatalf("first item=%+v", view.Items[0])
	}
	lookup := view.Items[1].Lookup
	if lookup == nil || lookup.Pattern != "order_entry.customer_reference" || lookup.Kind != application.PatternReference ||
		lookup.TargetAttribute != "customer.customer.customer_id" || lookup.TargetType != "customer.customer_id" {
		t.Fatalf("lookup=%+v", lookup)
	}
	if len(view.Navigations) != 1 || view.Navigations[0].From != "order_entry.recent_orders" ||
		view.Navigations[0].Kind != application.PatternBrowse || view.Navigations[0].Data != "sales.order" {
		t.Fatalf("navigations=%+v", view.Navigations)
	}
}

func TestBuildFailsClosedOnMissingReferences(t *testing.T) {
	cases := []struct {
		name string
		edit func(*ir.World)
	}{
		{
			name: "unknown application",
			edit: func(w *ir.World) { w.Applications[0].ID = "other" },
		},
		{
			name: "unknown pattern",
			edit: func(w *ir.World) { w.Applications[0].Patterns[2].ID = "order_entry.other" },
		},
		{
			name: "no interaction",
			edit: func(w *ir.World) { w.Applications[0].Patterns[2].Interaction = nil },
		},
		{
			name: "item is not an attribute",
			edit: func(w *ir.World) {
				w.Applications[0].Patterns[2].Interaction.Items[0].Attribute = "sales.order.missing"
			},
		},
		{
			name: "lookup is not a reference pattern",
			edit: func(w *ir.World) {
				w.Applications[0].Patterns[2].Interaction.Items[1].Lookup.Pattern = "order_entry.recent_orders"
			},
		},
		{
			name: "navigation source is not browse",
			edit: func(w *ir.World) {
				w.Applications[0].Patterns[2].Interaction.Navigations[0].From = "order_entry.order_record"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := viewWorld()
			tc.edit(&w)
			if _, err := applicationview.Build(w, "order_entry", "order_entry.order_record"); err == nil {
				t.Fatal("参照の欠落を拒否していません")
			}
		})
	}
}

func TestBuildAddsLabelAndInputKind(t *testing.T) {
	view, err := applicationview.Build(viewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if view.Items[0].Label != "受注番号" || view.Items[0].Input != "text" {
		t.Fatalf("item=%+v", view.Items[0])
	}
	if view.Items[1].Label != "sales.order.customer_id" || view.Items[1].Input != "text" {
		t.Fatalf("item=%+v", view.Items[1])
	}
}

func TestBuildCarriesRequiredAndImmutable(t *testing.T) {
	view, err := applicationview.Build(viewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if !view.Items[0].Required || !view.Items[0].Immutable {
		t.Fatalf("item=%+v", view.Items[0])
	}
	if view.Items[1].Required || view.Items[1].Immutable {
		t.Fatalf("item=%+v", view.Items[1])
	}
}

func TestBuildCarriesCommands(t *testing.T) {
	view, err := applicationview.Build(viewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Commands) != 1 || view.Commands[0].Capability != "sales.confirm_order" || view.Commands[0].Label != "confirm_order" {
		t.Fatalf("commands=%+v", view.Commands)
	}
	command := view.Commands[0]
	if len(command.Inputs) != 1 || command.Inputs[0].Name != "order_id" || command.Inputs[0].Type != "std.identifier" {
		t.Fatalf("inputs=%+v", command.Inputs)
	}
	if len(command.Outcomes) != 1 || command.Outcomes[0].Name != "confirmed" || command.Outcomes[0].Payload != "sales.order_confirmed" {
		t.Fatalf("outcomes=%+v", command.Outcomes)
	}
	if command.Effect == nil || command.Effect.Kind != "update" || command.Effect.Data != "sales.order" {
		t.Fatalf("effect=%+v", command.Effect)
	}
}
