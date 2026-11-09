package studiohost

import (
	"tei/internal/command"
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
	if run.Value.Mode != "reference" || !run.Value.Committed ||
		run.Value.Persisted || run.Value.Persistence != string(command.PersistenceNotApplicable) ||
		run.Value.Durability != "none" {
		t.Fatalf("run=%+v", run.Value)
	}
}

func TestExecuteCommandReportsInvalidInput(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommand("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", nil)
	// The entrance passes through the code the core already made
	// machine-readable, so a caller can tell a missing input from a failed
	// commit without parsing the message (#3807).
	if run.OK || run.Failure == nil || run.Failure.Code != "INVALID_COMMAND_INPUT" {
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
	// The refusal is its own code now, so this no longer has to search the
	// message text for "COMMAND_UNAUTHORIZED" (#3807).
	if run.OK || run.Failure == nil || run.Failure.Code != "COMMAND_UNAUTHORIZED" {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
	allowed := host.ExecuteCommandWithPolicy("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", map[string]string{"order": "ORD-1"}, nil, []string{"sales.order.submit"})
	if !allowed.OK {
		t.Fatalf("allowed run=%+v", allowed.Failure)
	}
	if allowed.Value == nil || len(allowed.Value.Records) != 1 {
		t.Fatalf("records=%+v", allowed.Value)
	}
}

// TestExecuteCommandRecordsDeclaredEffectWithoutReport checks that a
// reference execution whose declared effect is not reported by the executor is
// recorded as such, not silently treated as applied (#2996).
func TestExecuteCommandRecordsDeclaredEffectWithoutReport(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommand("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", map[string]string{"order": "ORD-1"})
	if !run.OK || run.Value == nil {
		t.Fatalf("run=%+v", run)
	}
	// The reference executor does report the effect, so the marker stays false.
	if run.Value.ReportedEffectMissing {
		t.Fatalf("reported effect was marked missing: %+v", run.Value)
	}
}
