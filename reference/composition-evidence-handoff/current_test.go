package handoff

import (
	"testing"

	"tei/internal/composition"
)

func TestReconcileCurrentRejectsStaleExternalReference(t *testing.T) {
	set, registry := proposal(t)
	targets := Reconcile(set, nil).Subjects
	if len(targets) != 2 {
		t.Fatalf("unexpected target closure: %+v", targets)
	}
	references := []EvidenceReference{ref(targets[0].Target), ref(targets[1].Target)}
	linked := ReconcileCurrent(registry, set, references)
	if linked.Blocked || len(linked.Subjects) != 2 {
		t.Fatalf("current proposal not linked: %+v", linked)
	}
	// Unrelated edits must not invalidate the selected dependency closure.
	state, err := registry.Apply(set.Revision, []composition.Provider{{
		ID: "unrelated", Scope: "tenant-b",
		Offers: composition.Contract{ID: "test.unrelated", Version: "1", Shape: "unrelated"},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := ReconcileCurrent(registry, set, references); got.Blocked {
		t.Fatalf("unrelated edit invalidated current proposal: %+v", got)
	}
	// Changing the transitive dependency's declaration invalidates the old
	// proposal, even if its root provider and contract remain unchanged.
	_, err = registry.Apply(state.Revision, []composition.Provider{{
		ID: "database", Scope: "tenant-a", Offers: read,
		Requires: []composition.Dependency{{Contract: app, ProviderID: "consumer"}},
	}}, nil)
	// The new definition introduces a cycle. Registry must reject it and
	// leave the previous proposal current.
	if err == nil {
		t.Fatal("dependency cycle accepted")
	}
	if !registry.ProposalCurrent(set) {
		t.Fatal("rejected registry edit invalidated current proposal")
	}
	// Replace a provider's contract, making its consumer ineligible.
	state, err = registry.Apply(state.Revision, []composition.Provider{{
		ID: "database", Scope: "tenant-a",
		Offers: composition.Contract{ID: "test.changed", Version: "1", Shape: "changed"},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	stale := ReconcileCurrent(registry, set, references)
	if !stale.Blocked || len(stale.Subjects) != 0 {
		t.Fatalf("stale reference still exposed as current: %+v", stale)
	}
	// A historical report can still describe the old exact revision, but
	// must not be confused with a verified/current result.
	if archived := Reconcile(set, references); archived.Blocked {
		t.Fatal("historical reconciliation unexpectedly discarded")
	}
	_ = state
}

func TestReconcileCurrentRejectsForgedSnapshotAndNilRegistry(t *testing.T) {
	set, registry := proposal(t)
	if r := ReconcileCurrent(nil, set, nil); !r.Blocked || len(r.Subjects) != 0 {
		t.Fatalf("nil Registry linked references: %+v", r)
	}
	set.Findings[0].VerificationTargets[0].Epoch++
	if r := ReconcileCurrent(registry, set, nil); !r.Blocked || len(r.Subjects) != 0 {
		t.Fatalf("forged Registry proposal linked references: %+v", r)
	}
}
