package studiohost

import "testing"

func TestPlanAndExecuteCommand(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	plan := host.PlanCommand("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", map[string]string{"order": "ORD-1"})
	if !plan.OK || plan.Failure != nil {
		t.Fatalf("plan=%+v failure=%+v", plan, plan.Failure)
	}
	if !plan.Value.Ready || len(plan.Value.Inputs) != 1 {
		t.Fatalf("plan=%+v", plan.Value)
	}

	run := host.ExecuteCommand("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", map[string]string{"order": "ORD-1"})
	if !run.OK || run.Failure != nil {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
	if run.Value.Mode != "reference" || !run.Value.Committed {
		t.Fatalf("run=%+v", run.Value)
	}
}

func TestExecuteCommandReportsInvalidInput(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommand("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", nil)
	if run.OK || run.Failure == nil || run.Failure.Code != "COMMAND_FAILED" {
		t.Fatalf("run=%+v", run)
	}
}
