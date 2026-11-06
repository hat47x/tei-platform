package studiohost

import (
	"strings"
	"testing"
)

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
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
}

func TestExecuteCommandRejectsInvalidPeriodState(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommandWithPeriods("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", map[string]string{"order": "ORD-1"}, map[string]string{"sales.period": "unknown"})
	if run.OK || run.Failure == nil || run.Failure.Code != "INVALID_PERIOD_STATE" {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
}

func TestExecuteCommandWithPolicyRejectsUnlistedCapability(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommandWithPolicy("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", map[string]string{"order": "ORD-1"}, nil, []string{"sales.other"})
	if run.OK || run.Failure == nil || !strings.Contains(run.Failure.Message, "COMMAND_UNAUTHORIZED") {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
	allowed := host.ExecuteCommandWithPolicy("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", map[string]string{"order": "ORD-1"}, nil, []string{"sales.order.submit"})
	if !allowed.OK {
		t.Fatalf("allowed run=%+v", allowed.Failure)
	}
}
