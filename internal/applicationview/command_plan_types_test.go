package applicationview_test

import (
	"testing"

	"tei/internal/applicationview"
	"tei/internal/semantic"
)

func TestBuildCommandPlanValidatesPrimitiveTypes(t *testing.T) {
	world := viewWorld()
	world.Semantics[0].Capabilities[0].Inputs[0].Type = "std.integer"
	plan, err := applicationview.BuildCommandPlan(world, "order_entry", "order_entry.order_record", "sales.confirm_order", map[string]string{"order_id": "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Ready || len(plan.Invalid) != 1 || plan.Invalid[0].Input != "order_id" {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestBuildCommandPlanResolvesAliasTypes(t *testing.T) {
	world := viewWorld()
	world.Semantics[0].Types = append(world.Semantics[0].Types, semantic.Type{
		ID:         "sales.quantity",
		Kind:       semantic.TypeAlias,
		Underlying: "std.integer",
	})
	world.Semantics[0].Capabilities[0].Inputs[0].Type = "sales.quantity"
	plan, err := applicationview.BuildCommandPlan(world, "order_entry", "order_entry.order_record", "sales.confirm_order", map[string]string{"order_id": "12"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Ready || len(plan.Invalid) != 0 {
		t.Fatalf("plan=%+v", plan)
	}

	invalid, err := applicationview.BuildCommandPlan(world, "order_entry", "order_entry.order_record", "sales.confirm_order", map[string]string{"order_id": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Ready || len(invalid.Invalid) != 1 {
		t.Fatalf("plan=%+v", invalid)
	}
}
