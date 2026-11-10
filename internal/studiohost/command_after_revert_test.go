package studiohost

import (
	"context"
	"testing"

	"tei/internal/command"
	"tei/internal/hook"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// studioRefusingRule refuses every phase it is asked to decide, which is what a
// rule supplied for a hook does when it does not hold.
type studioRefusingRule struct{}

func (studioRefusingRule) Run(context.Context, hook.Invocation) error {
	return &hook.NotSatisfied{Reason: "the record does not satisfy the after-change rule"}
}

// TestAfterChangeRefusalPutsTheStudioStoreBack checks the compensation Studio did
// not have.
//
// The effect has already been applied when the after-change phases run, so a
// refusal there arrives after the change. Leaving it in place would keep
// something the declaration did not accept; restoring makes the reported state
// the state before the run.
func TestAfterChangeRefusalPutsTheStudioStoreBack(t *testing.T) {
	host := &ReadOnly{}
	loaded, err := workspace.Load(afterChangeWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	world := loaded.World
	store := command.NewMemoryStore()
	effect := &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	key := "order=ORD-1"
	if err := store.ApplyRecord(effect, key, map[string]string{"order": "ORD-1"}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, ok := store.Get("sales.order", key); !ok {
		t.Fatal("the change was not applied")
	}
	empty := command.NewMemoryStore().Snapshot()

	report, reverted, err := host.runAfterHalfRevertingOnRefusal(world, "order_entry",
		"order_entry.order_record", map[string]string{"sales.order.order_id": "ORD-1"}, studioRefusingRule{}, store, empty, nil)
	if err == nil {
		t.Fatal("a refusing after phase did not report an error")
	}
	if !reverted {
		t.Fatal("the effect was not reported as reverted")
	}
	if _, ok := store.Get("sales.order", key); ok {
		t.Fatal("the applied change is still in the store")
	}
	if report == nil || report.Refused != "after-create" {
		t.Fatalf("report=%+v", report)
	}
}

// TestAfterChangeWithoutRefusalKeepsTheStudioStore checks the other side.
func TestAfterChangeWithoutRefusalKeepsTheStudioStore(t *testing.T) {
	host, err := NewReadOnly([]string{afterChangeWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(afterChangeWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	world := loaded.World
	store := command.NewMemoryStore()
	effect := &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	key := "order=ORD-1"
	if err := store.ApplyRecord(effect, key, map[string]string{"order": "ORD-1"}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	empty := command.NewMemoryStore().Snapshot()

	_, reverted, err := host.runAfterHalfRevertingOnRefusal(world, "order_entry",
		"order_entry.order_record", map[string]string{"sales.order.order_id": "ORD-1"}, hook.ReferenceRule{}, store, empty, nil)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if reverted {
		t.Fatal("a non-refusing run reported a revert")
	}
	if _, ok := store.Get("sales.order", key); !ok {
		t.Fatal("the applied change was dropped without a refusal")
	}
}
