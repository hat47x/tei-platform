package composition

import "testing"

func TestProposalCurrentTracksTransitiveProviderChanges(t *testing.T) {
	r := NewRegistry()
	contractApp := Contract{ID: "test.fresh.app", Version: "1", Shape: "app"}
	contractRead := Contract{ID: "test.fresh.read", Version: "1", Shape: "read"}
	contractCache := Contract{ID: "test.fresh.cache", Version: "1", Shape: "cache"}
	state, err := r.Apply(0, []Provider{
		{ID: "storage", Scope: "tenant-a", Offers: contractRead},
		{ID: "consumer", Scope: "tenant-a", Offers: contractApp, Requires: []Dependency{{Contract: contractRead}}},
		{ID: "cache", Scope: "tenant-a", Offers: contractCache},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	reqs := []WiringRequirement{{Scope: "tenant-a", Contract: contractApp}}
	first := r.InspectWiringSet(reqs)
	if !r.ProposalCurrent(first) {
		t.Fatalf("initial structural proposal stale: %+v", first)
	}
	state, err = r.Apply(state.Revision, []Provider{{ID: "unrelated", Scope: "tenant-b", Offers: contractRead}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !r.ProposalCurrent(first) {
		t.Fatalf("unrelated update invalidated selection: %+v", first)
	}
	state, err = r.Apply(state.Revision, []Provider{{
		ID: "storage", Scope: "tenant-a", Offers: contractRead,
		Requires: []Dependency{{Contract: contractCache}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.ProposalCurrent(first) {
		t.Fatal("transitive dependency change did not invalidate old proposal")
	}
	fresh := r.InspectWiringSet(reqs)
	if !r.ProposalCurrent(fresh) || len(fresh.Findings[0].VerificationTargets) != 3 {
		t.Fatalf("fresh dependency closure not tracked: %+v", fresh)
	}
	state, err = r.Apply(state.Revision, nil, []string{"storage"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ProposalCurrent(fresh) {
		t.Fatal("withdrawn transitive provider remained current")
	}
	state, err = r.Apply(state.Revision, []Provider{{
		ID: "storage", Scope: "tenant-a", Offers: contractRead,
		Requires: []Dependency{{Contract: contractCache}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.ProposalCurrent(fresh) {
		t.Fatal("same-name reinstatement revived stale proposal")
	}
	_ = state
}

func TestProposalCurrentRejectsForgedAndPartialSnapshots(t *testing.T) {
	r := NewRegistry()
	c := Contract{ID: "test.fresh.single", Version: "1", Shape: "shape"}
	if _, err := r.Apply(0, []Provider{{ID: "source", Scope: "tenant-a", Offers: c}}, nil); err != nil {
		t.Fatal(err)
	}
	orig := r.InspectWiringSet([]WiringRequirement{{Scope: "tenant-a", Contract: c}})
	if !r.ProposalCurrent(orig) {
		t.Fatalf("candidate unexpectedly invalid: %+v", orig)
	}
	cases := []struct {
		name string
		edit func(*WiringSetFinding)
	}{
		{"wrong-scope", func(s *WiringSetFinding) { s.Scope = "tenant-b" }},
		{"wrong-revision", func(s *WiringSetFinding) { s.Revision++ }},
		{"forged-provider", func(s *WiringSetFinding) { s.Findings[0].Selected.ProviderID = "forged" }},
		{"missing-target", func(s *WiringSetFinding) { s.Findings[0].VerificationTargets = nil }},
		{"forged-target", func(s *WiringSetFinding) { s.Findings[0].VerificationTargets[0].Epoch++ }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := r.InspectWiringSet([]WiringRequirement{{Scope: "tenant-a", Contract: c}})
			tc.edit(&got)
			if r.ProposalCurrent(got) {
				t.Fatalf("forged proposal accepted: %+v", got)
			}
		})
	}
	if (*Registry)(nil).ProposalCurrent(orig) {
		t.Fatal("nil Registry accepted proposal")
	}
}

func TestProposalCurrentConcurrentRegistryUpdatesRemainReadOnly(t *testing.T) {
	r := NewRegistry()
	contract := Contract{ID: "test.fresh.concurrent", Version: "1", Shape: "shape"}
	initial, err := r.Apply(0, []Provider{{ID: "source", Scope: "tenant-a", Offers: contract}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	proposal := r.InspectWiringSet([]WiringRequirement{{Contract: contract, Scope: "tenant-a"}})
	if !r.ProposalCurrent(proposal) {
		t.Fatal("initial proposal unexpectedly stale")
	}
	done := make(chan error, 1)
	go func() {
		revision := initial.Revision
		for i := 0; i < 100; i++ {
			state, err := r.Apply(revision, []Provider{{
				ID: "unrelated", Scope: "tenant-b",
				Offers: Contract{ID: "test.other", Version: "1", Shape: "other"},
			}}, nil)
			if err != nil {
				done <- err
				return
			}
			revision = state.Revision
		}
		done <- nil
	}()
	for i := 0; i < 100; i++ {
		if !r.ProposalCurrent(proposal) {
			t.Fatal("independent registry change invalidated stable proposal")
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
