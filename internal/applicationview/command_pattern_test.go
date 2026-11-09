package applicationview_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/ir"
)

func commandPatternWorld() ir.World {
	world := viewWorld()
	world.Applications[0].Patterns = append(world.Applications[0].Patterns, application.Pattern{
		ID:         "order_entry.submit_order",
		Kind:       application.PatternCommand,
		Capability: "sales.confirm_order",
	})
	return world
}

func TestBuildCommandPattern(t *testing.T) {
	view, err := applicationview.BuildCommandPattern(commandPatternWorld(), "order_entry", "order_entry.submit_order")
	if err != nil {
		t.Fatal(err)
	}
	if view.Capability != "sales.confirm_order" || len(view.Inputs) != 1 || view.Inputs[0].Name != "order_id" {
		t.Fatalf("view=%+v", view)
	}
	if len(view.Outcomes) != 1 || view.Outcomes[0].Name != "confirmed" {
		t.Fatalf("outcomes=%+v", view.Outcomes)
	}
	if view.Effect == nil || view.Effect.Kind != "update" {
		t.Fatalf("effect=%+v", view.Effect)
	}
}

func TestBuildCommandPatternRejectsOtherKind(t *testing.T) {
	world := commandPatternWorld()
	for i := range world.Applications[0].Patterns {
		if world.Applications[0].Patterns[i].ID == "order_entry.submit_order" {
			world.Applications[0].Patterns[i].Kind = application.PatternRecord
		}
	}
	if _, err := applicationview.BuildCommandPattern(world, "order_entry", "order_entry.submit_order"); err == nil {
		t.Fatal("non-command pattern was accepted")
	}
}
