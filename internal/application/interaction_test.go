package application_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
)

func TestInteractionNavigationTargetsReturnsRecordPatternsInIDOrder(t *testing.T) {
	app := application.Application{Patterns: []application.Pattern{
		{ID: "app.browse", Kind: application.PatternBrowse},
		{
			ID:   "app.record_b",
			Kind: application.PatternRecord,
			Interaction: &application.Interaction{
				Navigations: []application.InteractionNavigation{{From: "app.browse"}},
			},
		},
		{
			ID:   "app.record_a",
			Kind: application.PatternRecord,
			Interaction: &application.Interaction{
				Navigations: []application.InteractionNavigation{{From: "app.browse"}},
			},
		},
		{
			ID:   "app.other",
			Kind: application.PatternRecord,
			Interaction: &application.Interaction{
				Navigations: []application.InteractionNavigation{{From: "app.other_browse"}},
			},
		},
	}}

	got := application.InteractionNavigationTargets(app, "app.browse")
	if len(got) != 2 || got[0].ID != "app.record_a" || got[1].ID != "app.record_b" {
		t.Fatalf("navigation targets=%#v", got)
	}
	if got := application.InteractionNavigationTargets(app, "app.missing"); len(got) != 0 {
		t.Fatalf("unknown source returned %#v", got)
	}
}

func TestInteractionItemByAttributeDoesNotInferMissingItem(t *testing.T) {
	interaction := &application.Interaction{Items: []application.InteractionItem{
		{Attribute: "sales.order.order_id"},
		{
			Attribute: "sales.order.customer_id",
			Lookup: &application.InteractionLookup{
				Pattern:         "order_entry.customer_reference",
				TargetAttribute: "customer.customer.customer_id",
			},
		},
	}}

	item, ok := application.InteractionItemByAttribute(interaction, core.CanonicalID("sales.order.customer_id"))
	if !ok || item.Lookup == nil || item.Lookup.Pattern != "order_entry.customer_reference" {
		t.Fatalf("item=%#v ok=%v", item, ok)
	}
	if _, ok := application.InteractionItemByAttribute(interaction, core.CanonicalID("sales.order.missing")); ok {
		t.Fatal("missing item was inferred")
	}
	if _, ok := application.InteractionItemByAttribute(nil, core.CanonicalID("sales.order.order_id")); ok {
		t.Fatal("nil interaction returned an item")
	}
}
