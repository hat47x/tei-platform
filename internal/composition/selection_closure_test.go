package composition

import (
	"errors"
	"reflect"
	"testing"
)

var proposalDeepContract = Contract{ID: "test.proposal.deep", Version: "1", Shape: "deep-1"}
var proposalCacheContract = Contract{ID: "test.proposal.cache", Version: "1", Shape: "cache-1"}

func TestDependencyClosureInvalidatesTransitiveChange(t *testing.T) {
	r := NewRegistry()
	s, err := r.Apply(0, []Provider{
		{ID: "db", Scope: "tenant-a", Offers: probeRead},
		{ID: "consumer", Scope: "tenant-a", Offers: probeApp, Requires: []Dependency{{Contract: probeRead}}},
		{ID: "top", Scope: "tenant-a", Offers: proposalDeepContract, Requires: []Dependency{{Contract: probeApp}}},
		{ID: "cache", Scope: "tenant-a", Offers: proposalCacheContract},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.Select("top", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]string{first.DependencyEpochs[0].ProviderID, first.DependencyEpochs[1].ProviderID}, []string{"consumer", "db"}) {
		t.Fatalf("bad closure %+v", first.DependencyEpochs)
	}
	unchanged, err := r.Apply(s.Revision, []Provider{{ID: "unrelated", Scope: "tenant-b", Offers: probeRead}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := Binding{ContractID: proposalDeepContract.ID, ContractVersion: proposalDeepContract.Version, ProviderID: "top"}
	r.mu.Lock()
	err = r.validateSelectionLocked(first, b)
	r.mu.Unlock()
	if err != nil {
		t.Fatalf("unrelated edit invalidated closure: %v", err)
	}
	_, err = r.Apply(unchanged.Revision, []Provider{{ID: "db", Scope: "tenant-a", Offers: probeRead, Requires: []Dependency{{Contract: proposalCacheContract}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	still, err := r.Select("top", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if still.Epoch != first.Epoch {
		t.Fatal("root declaration unexpectedly changed")
	}
	r.mu.Lock()
	err = r.validateSelectionLocked(first, b)
	r.mu.Unlock()
	if !errors.Is(err, ErrStaleRegistry) {
		t.Fatalf("transitive change not detected %v", err)
	}
	r.mu.Lock()
	err = r.validateSelectionLocked(still, b)
	r.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if len(still.DependencyEpochs) != 3 {
		t.Fatalf("nested dependency not captured: %+v", still.DependencyEpochs)
	}
}

func TestDependencyClosureDefensiveCopyAndForgedEpoch(t *testing.T) {
	r := NewRegistry()
	_, err := r.Apply(0, []Provider{{ID: "db", Scope: "tenant-a", Offers: probeRead}, {ID: "consumer", Scope: "tenant-a", Offers: probeApp, Requires: []Dependency{{Contract: probeRead}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.Select("consumer", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.DependencyEpochs) != 1 {
		t.Fatal("missing pinned dependency")
	}
	first.DependencyEpochs[0].Epoch++
	b := Binding{ContractID: probeApp.ID, ContractVersion: probeApp.Version, ProviderID: "consumer"}
	r.mu.Lock()
	err = r.validateSelectionLocked(first, b)
	r.mu.Unlock()
	if !errors.Is(err, ErrStaleRegistry) {
		t.Fatalf("forged epoch accepted %v", err)
	}
	second, err := r.Select("consumer", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if second.DependencyEpochs[0].Epoch == first.DependencyEpochs[0].Epoch {
		t.Fatal("internal slice changed")
	}
	f := r.InspectWiringSet([]WiringRequirement{{Contract: probeApp, Scope: "tenant-a"}})
	if f.Findings[0].Selected == nil || len(f.Findings[0].Selected.DependencyEpochs) != 1 {
		t.Fatalf("no proposal closure %+v", f)
	}
}
