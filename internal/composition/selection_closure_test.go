package composition

import (
	"errors"
	"fmt"
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

func TestDeepDependencyClosureAndUnrelatedScopeRemainStable(t *testing.T) {
	const count = 96
	r := NewRegistry()
	providers := make([]Provider, 0, count)
	for i := 0; i < count; i++ {
		p := Provider{
			ID: fmt.Sprintf("p%03d", i), Scope: "tenant-a",
			Offers: Contract{ID: fmt.Sprintf("c%03d", i), Version: "1", Shape: "v1"},
		}
		if i > 0 {
			p.Requires = []Dependency{{Contract: providers[i-1].Offers}}
		}
		providers = append(providers, p)
	}
	snapshot, err := r.Apply(0, providers, nil)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := r.Select("p095", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.DependencyEpochs) != count-1 ||
		selected.DependencyEpochs[0].ProviderID != "p000" ||
		selected.DependencyEpochs[count-2].ProviderID != "p094" {
		t.Fatalf("deep dependency closure is incomplete or unordered: %+v", selected.DependencyEpochs)
	}
	identity := Binding{
		ProviderID: selected.ProviderID,
		ContractID: selected.Contract.ID,
		ContractVersion: selected.Contract.Version,
	}
	check := func(selection Selection) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.validateSelectionLocked(selection, identity)
	}
	if err := check(selected); err != nil {
		t.Fatal(err)
	}
	snapshot, err = r.Apply(snapshot.Revision, []Provider{{
		ID: "unrelated", Scope: "tenant-b", Offers: providers[0].Offers,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := check(selected); err != nil {
		t.Fatalf("other-tenant update invalidated a deep selection: %v", err)
	}
	changedRoot := providers[0]
	changedRoot.Requires = []Dependency{{Contract: Contract{
		ID: "missing", Version: "1", Shape: "missing",
	}}}
	if _, err := r.Apply(snapshot.Revision, []Provider{changedRoot}, nil); err != nil {
		t.Fatal(err)
	}
	if err := check(selected); !errors.Is(err, ErrStaleRegistry) &&
		!errors.Is(err, ErrProviderIneligible) {
		t.Fatalf("changed deep dependency remained eligible: %v", err)
	}
}

func BenchmarkSelectDeepDependencyClosure(b *testing.B) {
	for _, count := range []int{16, 64, 128} {
		b.Run(fmt.Sprintf("providers-%d", count), func(b *testing.B) {
			r := NewRegistry()
			providers := make([]Provider, 0, count)
			for i := 0; i < count; i++ {
				p := Provider{
					ID: fmt.Sprintf("p%03d", i), Scope: "tenant-a",
					Offers: Contract{ID: fmt.Sprintf("c%03d", i), Version: "1", Shape: "v1"},
				}
				if i > 0 {
					p.Requires = []Dependency{{Contract: providers[i-1].Offers}}
				}
				providers = append(providers, p)
			}
			if _, err := r.Apply(0, providers, nil); err != nil {
				b.Fatal(err)
			}
			target := providers[count-1].ID
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				selected, err := r.Select(target, "tenant-a")
				if err != nil || len(selected.DependencyEpochs) != count-1 {
					b.Fatalf("deep selection not available: %+v %v", selected, err)
				}
			}
		})
	}
}

func TestDependencyFreeSelectionKeepsEmptyClosure(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{{
		ID: "leaf", Scope: "tenant-a",
		Offers: Contract{ID: "leaf.contract", Version: "1", Shape: "v1"},
	}}, nil); err != nil {
		t.Fatal(err)
	}
	selected, err := r.Select("leaf", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.DependencyEpochs) != 0 {
		t.Fatalf("dependency-free provider acquired unexpected pins: %+v", selected)
	}
}

func BenchmarkSelectDependencyFreeProvider(b *testing.B) {
	r := NewRegistry()
	providers := make([]Provider, 128)
	for i := range providers {
		providers[i] = Provider{
			ID: fmt.Sprintf("leaf%03d", i), Scope: "tenant-a",
			Offers: Contract{ID: fmt.Sprintf("leaf%03d", i), Version: "1", Shape: "v1"},
		}
	}
	if _, err := r.Apply(0, providers, nil); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selection, err := r.Select("leaf127", "tenant-a")
		if err != nil || len(selection.DependencyEpochs) != 0 {
			b.Fatalf("dependency-free selection failed: %+v %v", selection, err)
		}
	}
}
