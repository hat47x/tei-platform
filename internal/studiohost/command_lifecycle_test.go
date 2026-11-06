package studiohost

import (
	"path/filepath"
	"testing"

	"tei/internal/command"
)

func newTrackedHost(t *testing.T, root, database string) *ReadOnly {
	t.Helper()
	store, err := command.OpenDurableLifecycle(database)
	if err != nil {
		t.Fatal(err)
	}
	host, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	host.CommandLifecycle = store
	return host
}

// #2564 Studio host wiring: a second host (new store handle, same database)
// must refuse a command the first host already observed.
func TestExecuteCommandWithLifecycleRefusesReplayAcrossHosts(t *testing.T) {
	root := referenceWorkspace(t)
	database := filepath.Join(t.TempDir(), "lifecycle.json")
	execute := func(host *ReadOnly) CommandRunResponse {
		return host.ExecuteCommandWithLifecycle(
			"workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit",
			map[string]string{"order": "ORD-1"}, nil, nil,
			"cmd-studio-1", "studio", 1,
		)
	}

	first := execute(newTrackedHost(t, root, database))
	if !first.OK || first.Failure != nil {
		t.Fatalf("first run=%+v failure=%+v", first, first.Failure)
	}
	if first.Value == nil || !first.Value.Committed || first.Value.Persisted ||
		first.Value.Durability != "none" || first.Value.Mode != "reference" {
		t.Fatalf("first value=%+v", first.Value)
	}

	second := execute(newTrackedHost(t, root, database))
	if second.OK || second.Failure == nil || second.Failure.Code != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("second run=%+v failure=%+v", second, second.Failure)
	}
	if second.Value != nil {
		t.Fatalf("a refused replay returned a value: %+v", second.Value)
	}
}

func TestExecuteCommandWithLifecycleRequiresAComposedStore(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommandWithLifecycle(
		"workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit",
		map[string]string{"order": "ORD-1"}, nil, nil,
		"cmd-studio-1", "studio", 1,
	)
	if run.OK || run.Failure == nil || run.Failure.Code != "COMMAND_LIFECYCLE_UNAVAILABLE" {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
}

func TestExecuteCommandWithLifecycleValidatesTheAttempt(t *testing.T) {
	root := referenceWorkspace(t)
	for name, attempt := range map[string]struct {
		commandID  string
		owner      string
		generation int
	}{
		"missing command": {owner: "studio", generation: 1},
		"missing owner":   {commandID: "cmd-studio-1", generation: 1},
		"zero generation": {commandID: "cmd-studio-1", owner: "studio"},
	} {
		t.Run(name, func(t *testing.T) {
			host := newTrackedHost(t, root, filepath.Join(t.TempDir(), "lifecycle.json"))
			run := host.ExecuteCommandWithLifecycle(
				"workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit",
				map[string]string{"order": "ORD-1"}, nil, nil,
				attempt.commandID, attempt.owner, attempt.generation,
			)
			if run.OK || run.Failure == nil || run.Failure.Code != "INVALID_LIFECYCLE_ATTEMPT" {
				t.Fatalf("run=%+v failure=%+v", run, run.Failure)
			}
		})
	}
}
