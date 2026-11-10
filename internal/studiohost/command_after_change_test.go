package studiohost

import (
	"os"
	"path/filepath"
	"testing"
)

// afterChangeWorkspace writes a workspace whose record pattern declares a
// before-create and an after-create hook over a create capability, so a test can
// apply a change and then have the after phase consulted.
//
// A create is used because the reference store starts empty: an update would be
// refused as record-missing before the after phase had a point to run at.
func afterChangeWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic", "sales"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "applications", "order_entry", "patterns"), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(rel, content string) {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: studio_after_fixture\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n")
	write(filepath.Join("semantic", "sales", "order.tei"), `namespace sales

data Order {
    attribute orderId type std.identifier
}

capability sales.create_order {
    input order sales.order
    output sales.order
    effect create sales.order
}
`)
	write(filepath.Join("applications", "order_entry", "application.tei"), `application order_entry {
    uses semantic sales
}
`)
	write(filepath.Join("applications", "order_entry", "patterns", "order-record.tei"), `pattern order_entry.order_record record {
    data sales.order

    interaction {
        item sales.order.order_id

        command sales.create_order {
            input order sales.order.order_id
        }

        hook before-create {
            reads sales.order.order_id
            on-failure reject
        }

        hook after-create {
            reads sales.order.order_id
            on-failure reject
        }
    }
}
`)
	return root
}

// TestExecuteCommandWithReferenceStoreReachesTheAfterPhase checks the wiring this
// test exists for.
//
// A reference execution has no change, so it has no after-change point and the
// after-* phases are reported as skipped. A run that applies its change to the
// in-memory store does have one, so the after-* phases are consulted after the
// effect was applied. Studio had no such entrance before this.
func TestExecuteCommandWithReferenceStoreReachesTheAfterPhase(t *testing.T) {
	host, err := NewReadOnly([]string{afterChangeWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommandWithReferenceStore("workspace-1", "order_entry",
		"order_entry.order_record", "sales.create_order",
		map[string]string{"order": "ORD-1", "sales.order.order_id": "ORD-1"}, nil, nil)
	if !run.OK || run.Failure != nil {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
	if run.Value.Hooks == nil {
		t.Fatal("the run recorded no hook report")
	}
	executed := map[string]bool{}
	for _, phase := range run.Value.Hooks.Executed {
		executed[phase] = true
	}
	if !executed["before-create"] {
		t.Fatalf("the before phase did not run: %v", run.Value.Hooks.Executed)
	}
	if !executed["after-create"] {
		t.Fatalf("the after phase did not run: %v", run.Value.Hooks.Executed)
	}
}

// TestExecuteCommandKeepsTheAfterPhaseSkipped checks that the default entrance is
// unchanged: without a store there is no after-change point.
func TestExecuteCommandKeepsTheAfterPhaseSkipped(t *testing.T) {
	host, err := NewReadOnly([]string{afterChangeWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	run := host.ExecuteCommand("workspace-1", "order_entry",
		"order_entry.order_record", "sales.create_order",
		map[string]string{"order": "ORD-1", "sales.order.order_id": "ORD-1"})
	if !run.OK || run.Failure != nil {
		t.Fatalf("run=%+v failure=%+v", run, run.Failure)
	}
	for _, phase := range run.Value.Hooks.Executed {
		if phase == "after-create" {
			t.Fatalf("the after phase ran without an applied change: %v", run.Value.Hooks.Executed)
		}
	}
	skipped := map[string]bool{}
	for _, phase := range run.Value.Hooks.Skipped {
		skipped[phase] = true
	}
	if !skipped["after-create"] {
		t.Fatalf("the after phase was not reported as skipped: %v", run.Value.Hooks.Skipped)
	}
}
