package studiohost

import (
	"testing"

	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// TestStudioAfterHalfReadsFromTheStore checks the Studio counterpart of the CLI
// wiring: an after-change phase reads the record the command wrote.
//
// Without the store the phase would be reported as not evaluated, because the
// caller never supplied the declared attribute — the request carries the command
// parameter, not the attribute the hook names.
func TestStudioAfterHalfReadsFromTheStore(t *testing.T) {
	host, err := NewReadOnly([]string{afterChangeWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	loaded, loadErr := workspace.Load(afterChangeWorkspace(t))
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	world := loaded.World
	store := command.NewMemoryStore()
	// The record the command would have written, with the attribute view the
	// declaration maps to.
	effect := &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	if err := store.ApplyRecord(effect, "order=ORD-1",
		map[string]string{"order": "ORD-1"},
		map[core.CanonicalID]string{"sales.order.order_id": "ORD-1"}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// The caller supplies nothing: only the store knows the attribute.
	report, err := host.runHooksHalfWithRowsAndStore(world, "order_entry",
		"order_entry.order_record", nil, nil, hook.ReferenceRule{},
		hook.AppliedChangeReach{}, hookHalfAfter, store)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil || report.PhaseReport == nil {
		t.Fatal("no report")
	}
	executed := map[string]bool{}
	for _, phase := range report.Executed {
		executed[phase] = true
	}
	if !executed["after-create"] {
		t.Fatalf("the after phase did not read the stored record: %+v", report.PhaseReport)
	}
}
