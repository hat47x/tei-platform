package main

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/hook"
	"tei/internal/semantic"
)

// refusingRule refuses every phase it is asked to decide, which is what a rule
// supplied for a hook does when it does not hold.
type refusingRule struct{}

func (refusingRule) Run(context.Context, hook.Invocation) error {
	return &hook.NotSatisfied{Reason: "the record does not satisfy the after-change rule"}
}

// afterChangeOnlyHooks declares just the after phase, so a test can apply a
// change and then have the after phase refuse it.
func afterChangeOnlyHooks() []application.Hook {
	return []application.Hook{{Phase: "after-create"}}
}

// TestAfterChangeRefusalPutsTheStoreBack checks the compensation the CLI did not
// have.
//
// The effect has already been applied when the after-change phases run, so a
// refusal there arrives after the change. Leaving it in place would keep
// something the declaration did not accept; restoring makes the run's reported
// state the state before the run, which is what the caller is told.
func TestAfterChangeRefusalPutsTheStoreBack(t *testing.T) {
	world := hookViewWorld(afterChangeOnlyHooks()...)
	store := command.NewMemoryStore()
	effect := &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	key := "order=ORD-1"
	if err := store.ApplyRecord(effect, key, map[string]string{"order": "ORD-1"}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, ok := store.Get("sales.order", key); !ok {
		t.Fatal("the change was not applied")
	}
	// The snapshot stands for "the state before this run applied its change".
	empty := command.NewMemoryStore().Snapshot()

	report, reverted, err := runAfterHalfRevertingOnRefusal(context.Background(), world,
		"order_entry", "order_entry.order_record", nil, refusingRule{}, store, empty, nil)
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

// TestAfterChangeWithoutRefusalKeepsTheChange checks the other side: a report
// that does not refuse leaves the applied change alone.
func TestAfterChangeWithoutRefusalKeepsTheChange(t *testing.T) {
	world := hookViewWorld(afterChangeOnlyHooks()...)
	store := command.NewMemoryStore()
	effect := &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
	key := "order=ORD-1"
	if err := store.ApplyRecord(effect, key, map[string]string{"order": "ORD-1"}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	empty := command.NewMemoryStore().Snapshot()

	_, reverted, err := runAfterHalfRevertingOnRefusal(context.Background(), world,
		"order_entry", "order_entry.order_record", nil, hook.ReferenceRule{}, store, empty, nil)
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
