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
	if got := r.InspectWiring(WiringRequirement{Contract: probeRead, Scope: "tenant-a", ProviderID: "storage-c"}); got.Verdict != ProposalConflict || got.Reason != ReasonExplicitMismatch {
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
