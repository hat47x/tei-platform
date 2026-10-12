package composition

import (
	"reflect"
	"testing"
)

var probeRead = Contract{ID: "test.read", Version: "1", Shape: "read/request->result"}
var probeApp = Contract{ID: "test.app", Version: "1", Shape: "app/request->result"}

func TestInspectWiringUnknownIsNotProofOfAbsence(t *testing.T) {
	r := NewRegistry()
	got := r.InspectWiring(WiringRequirement{Contract: probeRead, Scope: "tenant-a"})
	if got.Verdict != ProposalUnknown || got.Reason != ReasonNoKnownProvider || got.Selected != nil {
		t.Fatalf("missing catalog data was treated as complete: %+v", got)
	}
	if got := r.InspectWiring(WiringRequirement{Scope: "tenant-a"}); got.Reason != ReasonInvalidRequirement {
		t.Fatalf("invalid contract not rejected: %+v", got)
	}
}

func TestInspectWiringAmbiguityAndExplicitSelection(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{
		{ID: "storage-b", Scope: "tenant-a", Offers: probeRead},
		{ID: "storage-a", Scope: "tenant-a", Offers: probeRead},
		{ID: "storage-c", Scope: "tenant-b", Offers: probeRead},
	}, nil); err != nil {
		t.Fatal(err)
	}
	unknown := r.InspectWiring(WiringRequirement{Contract: probeRead, Scope: "tenant-a"})
	if unknown.Verdict != ProposalConflict || unknown.Reason != ReasonAmbiguousProviders || !reflect.DeepEqual(unknown.Candidates, []string{"storage-a", "storage-b"}) {
		t.Fatalf("ambiguous wiring silently chosen: %+v", unknown)
	}
	got := r.InspectWiring(WiringRequirement{Contract: probeRead, Scope: "tenant-a", ProviderID: "storage-a"})
	if got.Verdict != ProposalCandidate || got.Selected == nil || got.Selected.ProviderID != "storage-a" || got.Selected.Scope != "tenant-a" {
		t.Fatalf("explicit valid wiring not proposed: %+v", got)
	}
	if got := r.InspectWiring(WiringRequirement{Contract: probeRead, Scope: "tenant-a", ProviderID: "storage-c"}); got.Verdict != ProposalUnknown || got.Reason != ReasonNoKnownProvider {
		t.Fatalf("cross-tenant provider accepted: %+v", got)
	}
	if got := r.InspectWiring(WiringRequirement{Contract: probeRead, Scope: "tenant-a", ProviderID: "not-present"}); got.Verdict != ProposalUnknown || got.Reason != ReasonNoKnownProvider {
		t.Fatalf("unknown selected provider declared impossible: %+v", got)
	}
}

func TestInspectWiringDependencyFailureAndRecovery(t *testing.T) {
	r := NewRegistry()
	before, err := r.Apply(0, []Provider{
		{ID: "app", Scope: "tenant-a", Offers: probeApp, Requires: []Dependency{{Contract: probeRead}}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.InspectWiring(WiringRequirement{Contract: probeApp, Scope: "tenant-a"})
	if got.Verdict != ProposalUnknown || got.Reason != ReasonDependencyIneligible || got.Detail == "" {
		t.Fatalf("missing dependency reported as usable: %+v", got)
	}
	if _, err := r.Apply(before.Revision, []Provider{{ID: "storage", Scope: "tenant-a", Offers: probeRead}}, nil); err != nil {
		t.Fatal(err)
	}
	got = r.InspectWiring(WiringRequirement{Contract: probeApp, Scope: "tenant-a"})
	if got.Verdict != ProposalCandidate || got.Selected == nil || len(got.Selected.Dependencies) != 1 || got.Selected.Dependencies[0].ProviderID != "storage" {
		t.Fatalf("dependency closure not recognized: %+v", got)
	}
	// The design-time answer is a snapshot; callers cannot edit Registry state.
	got.Selected.Dependencies[0].ProviderID = "forged"
	fresh := r.InspectWiring(WiringRequirement{Contract: probeApp, Scope: "tenant-a"})
	if fresh.Selected.Dependencies[0].ProviderID != "storage" {
		t.Fatalf("shared dependency slice leaked: %+v", fresh)
	}
}

func TestInspectWiringProviderEpochChangesOnlyOnRelevantChanges(t *testing.T) {
	r := NewRegistry()
	first, err := r.Apply(0, []Provider{{ID: "one", Scope: "tenant-a", Offers: probeRead}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := WiringRequirement{Contract: probeRead, Scope: "tenant-a"}
	old := r.InspectWiring(req)
	if old.Selected == nil {
		t.Fatalf("expected candidate: %+v", old)
	}
	second, err := r.Apply(first.Revision, []Provider{{ID: "other", Scope: "tenant-b", Offers: probeApp}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	still := r.InspectWiring(req)
	if still.Selected == nil || still.Selected.Epoch != old.Selected.Epoch {
		t.Fatalf("unrelated edit invalidated proposal: %+v", still)
	}
	third, err := r.Apply(second.Revision, nil, []string{"one"})
	if err != nil {
		t.Fatal(err)
	}
	if got := r.InspectWiring(req); got.Verdict != ProposalUnknown {
		t.Fatalf("removed provider is still a candidate: %+v", got)
	}
	if _, err := r.Apply(third.Revision, []Provider{{ID: "one", Scope: "tenant-a", Offers: probeRead}}, nil); err != nil {
		t.Fatal(err)
	}
	newest := r.InspectWiring(req)
	if newest.Selected == nil || newest.Selected.Epoch <= old.Selected.Epoch {
		t.Fatalf("re-registered provider revived stale selection: %+v", newest)
	}
}

func TestInspectWiringRejectsWrongContractInSameScope(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{{ID: "known", Scope: "tenant-a", Offers: probeRead}}, nil); err != nil {
		t.Fatal(err)
	}
	got := r.InspectWiring(WiringRequirement{Contract: probeApp, Scope: "tenant-a", ProviderID: "known"})
	if got.Verdict != ProposalConflict || got.Reason != ReasonExplicitMismatch || got.Selected != nil {
		t.Fatalf("incompatible existing provider was accepted: %+v", got)
	}
}

func TestInspectWiringConcurrentCatalogUpdates(t *testing.T) {
	r := NewRegistry()
	start, err := r.Apply(0, []Provider{{ID: "one", Scope: "tenant-a", Offers: probeRead}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		rev := start.Revision
		for i := 0; i < 80; i++ {
			next, err := r.Apply(rev, []Provider{{ID: "other", Scope: "tenant-b", Offers: probeApp}}, nil)
			if err != nil {
				done <- err
				return
			}
			rev = next.Revision
		}
		done <- nil
	}()
	for i := 0; i < 80; i++ {
		got := r.InspectWiring(WiringRequirement{Contract: probeRead, Scope: "tenant-a"})
		if got.Verdict != ProposalCandidate || got.Selected == nil || got.Selected.ProviderID != "one" {
			t.Fatalf("concurrent unrelated updates changed service selection: %+v", got)
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestInspectWiringSetConsistentRevisionAndScope(t *testing.T) {
	r := NewRegistry()
	snap, err := r.Apply(0, []Provider{
		{ID: "data", Scope: "tenant-a", Offers: probeRead},
		{ID: "application", Scope: "tenant-a", Offers: probeApp, Requires: []Dependency{{Contract: probeRead}}},
		{ID: "outsider", Scope: "tenant-b", Offers: probeRead},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := r.InspectWiringSet([]WiringRequirement{
		{Contract: probeRead, Scope: "tenant-a"},
		{Contract: probeApp, Scope: "tenant-a"},
	})
	if out.Revision != snap.Revision || out.Scope != "tenant-a" || out.Verdict != ProposalCandidate || len(out.Findings) != 2 {
		t.Fatalf("invalid proposal snapshot: %+v", out)
	}
	if out.Findings[0].Selected == nil || out.Findings[1].Selected == nil {
		t.Fatalf("not all requirements evaluated: %+v", out)
	}
	if out.Findings[0].Selected.Revision != out.Revision || out.Findings[1].Selected.Revision != out.Revision {
		t.Fatalf("mixed revision in one evaluation: %+v", out)
	}
	if out.Findings[1].Selected.Dependencies[0].ProviderID != "data" {
		t.Fatal("dependency not resolved")
	}
	// A batch must not bridge tenants without a separate authorization.
	mixed := r.InspectWiringSet([]WiringRequirement{
		{Contract: probeRead, Scope: "tenant-a"},
		{Contract: probeRead, Scope: "tenant-b"},
	})
	if mixed.Verdict != ProposalUnknown || len(mixed.Findings) != 2 || mixed.Findings[0].Selected != nil || mixed.Findings[1].Selected != nil ||
		mixed.Findings[0].Reason != ReasonMixedScopes || mixed.Findings[1].Reason != ReasonMixedScopes {
		t.Fatalf("mixed tenant batch leaked candidates: %+v", mixed)
	}
	if got := r.InspectWiringSet(nil); got.Verdict != ProposalUnknown || len(got.Findings) != 0 {
		t.Fatalf("empty batch treated as complete: %+v", got)
	}
}

func TestInspectWiringSetConflictTakesPrecedenceOverUnknown(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{
		{ID: "data-a", Scope: "tenant-a", Offers: probeRead},
		{ID: "data-b", Scope: "tenant-a", Offers: probeRead},
	}, nil); err != nil {
		t.Fatal(err)
	}
	result := r.InspectWiringSet([]WiringRequirement{
		{Contract: probeApp, Scope: "tenant-a"},
		{Contract: probeRead, Scope: "tenant-a"},
	})
	if result.Verdict != ProposalConflict || result.Findings[0].Verdict != ProposalUnknown || result.Findings[1].Verdict != ProposalConflict {
		t.Fatalf("uncertain and conflicting requirements collapsed: %+v", result)
	}
}

func TestInspectWiringSetNoMixedEpochUnderConcurrentUpdates(t *testing.T) {
	r := NewRegistry()
	start, err := r.Apply(0, []Provider{{ID: "data", Scope: "tenant-a", Offers: probeRead}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		rev := start.Revision
		for i := 0; i < 50; i++ {
			next, err := r.Apply(rev, nil, []string{"data"})
			if err != nil {
				done <- err
				return
			}
			rev = next.Revision
			next, err = r.Apply(rev, []Provider{{ID: "data", Scope: "tenant-a", Offers: probeRead}}, nil)
			if err != nil {
				done <- err
				return
			}
			rev = next.Revision
		}
		done <- nil
	}()
	reqs := []WiringRequirement{{Contract: probeRead, Scope: "tenant-a"}, {Contract: probeRead, Scope: "tenant-a"}}
	for i := 0; i < 100; i++ {
		set := r.InspectWiringSet(reqs)
		a, b := set.Findings[0], set.Findings[1]
		if a.Verdict != b.Verdict || a.Reason != b.Reason {
			t.Fatalf("batch contains two registry states: %+v", set)
		}
		if a.Selected != nil && (b.Selected == nil || a.Selected.Epoch != b.Selected.Epoch || a.Selected.Revision != set.Revision || b.Selected.Revision != set.Revision) {
			t.Fatalf("batch contains two provider epochs: %+v", set)
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestInspectWiringSetRejectsContradictorySameSlot(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{
		{ID: "read-a", Scope: "tenant-a", Offers: probeRead},
		{ID: "read-b", Scope: "tenant-a", Offers: probeRead},
		{ID: "app", Scope: "tenant-a", Offers: probeApp},
	}, nil); err != nil {
		t.Fatal(err)
	}
	// Every individual explicit Provider exists and is eligible. The pair is
	// still an invalid proposal for one logical consumer binding point.
	conflict := r.InspectWiringSet([]WiringRequirement{
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-a", SlotID: "view.data"},
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-b", SlotID: "view.data"},
	})
	if conflict.Verdict != ProposalConflict {
		t.Fatalf("contradictory slot accepted: %+v", conflict)
	}
	for i, f := range conflict.Findings {
		if f.Verdict != ProposalConflict || f.Reason != ReasonInconsistentSlot ||
			f.Selected != nil || len(f.Candidates) != 0 {
			t.Fatalf("slot %d leaked an actionable candidate: %+v", i, f)
		}
	}
	// A different Contract must not be assigned to the same named slot.
	wrongContract := r.InspectWiringSet([]WiringRequirement{
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-a", SlotID: "view.data"},
		{Contract: probeApp, Scope: "tenant-a", ProviderID: "app", SlotID: "view.data"},
	})
	if wrongContract.Verdict != ProposalConflict ||
		wrongContract.Findings[1].Reason != ReasonInconsistentSlot {
		t.Fatalf("contradictory contract accepted: %+v", wrongContract)
	}
}

func TestInspectWiringSetKeepsIndependentSameContractSlots(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{
		{ID: "read-a", Scope: "tenant-a", Offers: probeRead},
		{ID: "read-b", Scope: "tenant-a", Offers: probeRead},
	}, nil); err != nil {
		t.Fatal(err)
	}
	distinct := r.InspectWiringSet([]WiringRequirement{
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-a", SlotID: "view.primary"},
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-b", SlotID: "view.secondary"},
	})
	if distinct.Verdict != ProposalCandidate || len(distinct.Findings) != 2 ||
		distinct.Findings[0].Selected == nil || distinct.Findings[1].Selected == nil ||
		distinct.Findings[0].Selected.ProviderID != "read-a" ||
		distinct.Findings[1].Selected.ProviderID != "read-b" ||
		distinct.Findings[0].Selected.Revision != distinct.Findings[1].Selected.Revision {
		t.Fatalf("independent slots incorrectly unified: %+v", distinct)
	}
	repeated := r.InspectWiringSet([]WiringRequirement{
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-a"},
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-a"},
	})
	if repeated.Verdict != ProposalCandidate {
		t.Fatalf("identical shared requirements refused: %+v", repeated)
	}
	// An omitted slot is shared for the same exact Contract by definition.
	implicit := r.InspectWiringSet([]WiringRequirement{
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-a"},
		{Contract: probeRead, Scope: "tenant-a", ProviderID: "read-b"},
	})
	if implicit.Verdict != ProposalConflict || implicit.Findings[0].Reason != ReasonInconsistentSlot {
		t.Fatalf("implicit slot contradiction accepted: %+v", implicit)
	}
}
