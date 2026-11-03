package applicationview_test

import (
	"testing"

	"tei/internal/applicationview"
	"tei/internal/semantic"
)

func TestBuildCommandPlan(t *testing.T) {
	plan, err := applicationview.BuildCommandPlan(viewWorld(), "order_entry", "order_entry.order_record", "sales.confirm_order", map[string]string{"order_id": "ORD-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Ready || len(plan.Missing) != 0 {
		t.Fatalf("plan=%+v", plan)
	}
	if len(plan.Inputs) != 1 || plan.Inputs[0].Name != "order_id" || plan.Inputs[0].Value != "ORD-1" || !plan.Inputs[0].Provided {
		t.Fatalf("inputs=%+v", plan.Inputs)
	}
	if len(plan.Outcomes) != 1 || plan.Outcomes[0].Name != "confirmed" {
		t.Fatalf("outcomes=%+v", plan.Outcomes)
	}
	if plan.Effect == nil || plan.Effect.Kind != "update" {
		t.Fatalf("effect=%+v", plan.Effect)
	}
}

func TestBuildCommandPlanReportsMissingInputs(t *testing.T) {
	plan, err := applicationview.BuildCommandPlan(viewWorld(), "order_entry", "order_entry.order_record", "sales.confirm_order", nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Ready || len(plan.Missing) != 1 || plan.Missing[0] != "order_id" {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestBuildCommandPlanRejectsUnknownCommand(t *testing.T) {
	if _, err := applicationview.BuildCommandPlan(viewWorld(), "order_entry", "order_entry.order_record", "sales.missing", nil); err == nil {
		t.Fatal("unknown command was accepted")
	}
}

func TestBuildCommandPlanCarriesPeriod(t *testing.T) {
	world := viewWorld()
	world.Semantics[0].Periods = []semantic.Period{{
		ID:            "sales.period",
		Data:          "sales.order",
		BusinessDate:  "sales.order.order_id",
		States:        []string{"open", "closed"},
		ClosedRejects: []string{"update"},
	}}
	plan, err := applicationview.BuildCommandPlan(world, "order_entry", "order_entry.order_record", "sales.confirm_order", map[string]string{"order_id": "ORD-1"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Period == nil || plan.Period.ID != "sales.period" || len(plan.Period.States) != 2 || len(plan.Period.ClosedRejects) != 1 {
		t.Fatalf("period=%+v", plan.Period)
	}
}
