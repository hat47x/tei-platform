package applicationview_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/ir"
)

// rowSourceViewWorld is viewWorld with the record pattern's hook carrying a row
// source. A reader of the projection has to see the source, because that is what
// tells a row-scoped declaration with a supplier from one without.
func rowSourceViewWorld() ir.World {
	world := viewWorld()
	for i := range world.Applications[0].Patterns {
		pattern := &world.Applications[0].Patterns[i]
		if pattern.ID != "order_entry.order_record" || pattern.Interaction == nil {
			continue
		}
		pattern.Interaction.Hooks[0].Scope = "row"
		pattern.Interaction.Hooks[0].Rows = &application.HookRowSource{Query: "order_entry.order_list"}
	}
	return world
}

// TestBuildCarriesTheHookRowSource checks that the projection carries the source
// the declaration names instead of dropping it: a host reading the view can then
// say which rows apply.
func TestBuildCarriesTheHookRowSource(t *testing.T) {
	view, err := applicationview.Build(rowSourceViewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Hooks) != 1 {
		t.Fatalf("hooks=%+v", view.Hooks)
	}
	declared := view.Hooks[0]
	if declared.Scope != "row" {
		t.Fatalf("scope=%q", declared.Scope)
	}
	if declared.Rows == nil || declared.Rows.Query != "order_entry.order_list" {
		t.Fatalf("rows=%+v", declared.Rows)
	}
}

// TestBuildLeavesTheRowSourceAbsentWhenNoneIsDeclared checks the other side: no
// source is projected as no source, not as an empty one that reads like a
// declaration.
func TestBuildLeavesTheRowSourceAbsentWhenNoneIsDeclared(t *testing.T) {
	world := viewWorld()
	for i := range world.Applications[0].Patterns {
		pattern := &world.Applications[0].Patterns[i]
		if pattern.ID != "order_entry.order_record" || pattern.Interaction == nil {
			continue
		}
		pattern.Interaction.Hooks[0].Scope = "row"
		pattern.Interaction.Hooks[0].Rows = nil
	}
	view, err := applicationview.Build(world, "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if view.Hooks[0].Rows != nil {
		t.Fatalf("rows=%+v", view.Hooks[0].Rows)
	}
}
