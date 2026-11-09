package studiohost

import (
	"testing"
)

// TestExecuteCommandReportsTheDeclaredHook checks that the Studio reference
// execution runs the pattern's before-change hook when the caller supplied its
// declared read and the hook declares no call, and reports it as executed.
func TestExecuteCommandReportsTheDeclaredHook(t *testing.T) {
	host, err := NewReadOnly([]string{hookWorkspaceWithoutCall(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommand(
		"workspace-1",
		"order_entry",
		"order_entry.order_record",
		"sales.submit",
		map[string]string{"order": "open", "sales.order.status": "open"},
	)
	if !run.OK || run.Failure != nil {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
	if run.Value.Hooks == nil {
		t.Fatal("no hook report for a pattern that declares a hook")
	}
	if len(run.Value.Hooks.Executed) != 1 || run.Value.Hooks.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", run.Value.Hooks.Executed)
	}
	if run.Value.Hooks.Note == "" {
		t.Fatal("the hook report does not state its boundary")
	}
}

// TestExecuteCommandCallsTheDeclaredCapability checks that a hook which declares
// `uses` actually reaches the Command boundary. Before the caller was wired,
// this phase was reported as executed while the capability was never called.
func TestExecuteCommandCallsTheDeclaredCapability(t *testing.T) {
	host, err := NewReadOnly([]string{hookWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommand(
		"workspace-1",
		"order_entry",
		"order_entry.order_record",
		"sales.submit",
		map[string]string{"order": "open", "sales.order.status": "open"},
	)
	if !run.OK || run.Failure != nil {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
	if run.Value.Hooks == nil {
		t.Fatal("no hook report")
	}
	if len(run.Value.Hooks.Executed) != 1 || run.Value.Hooks.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", run.Value.Hooks.Executed)
	}
	if len(run.Value.Hooks.Uncalled) != 0 {
		t.Fatalf("a declared call was not made: %v", run.Value.Hooks.Uncalled)
	}
	calls := run.Value.Hooks.Calls["before-update"]
	if len(calls) != 1 || !calls[0].Called || calls[0].Capability != "sales.submit" {
		t.Fatalf("calls=%+v", calls)
	}
	if calls[0].Error != "" {
		t.Fatalf("the call recorded an error: %+v", calls[0])
	}
}

// TestExecuteCommandRecordsAMissingReadAsNotEvaluated checks that a declared
// read the caller did not supply is reported as not evaluated with a reason,
// not as a pass.
func TestExecuteCommandRecordsAMissingReadAsNotEvaluated(t *testing.T) {
	host, err := NewReadOnly([]string{hookWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommand(
		"workspace-1",
		"order_entry",
		"order_entry.order_record",
		"sales.submit",
		map[string]string{"order": "open"},
	)
	if !run.OK || run.Failure != nil {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
	if run.Value.Hooks == nil {
		t.Fatal("no hook report")
	}
	if len(run.Value.Hooks.Executed) != 0 {
		t.Fatalf("a hook with no read was reported as executed: %v", run.Value.Hooks.Executed)
	}
	if len(run.Value.Hooks.NotEvaluated) != 1 || run.Value.Hooks.NotEvaluated[0] != "before-update" {
		t.Fatalf("notEvaluated=%v", run.Value.Hooks.NotEvaluated)
	}
	if run.Value.Hooks.Reasons["before-update"] == "" {
		t.Fatal("the not-evaluated phase has no stated reason")
	}
}
