package composition

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

// evaluateWithProviderScan preserves the previous matching algorithm as a
// test-only oracle. The indexed evaluator must produce the same outcome.
func evaluateWithProviderScan(providers map[string]Provider) (Snapshot, error) {
	var out Snapshot
	memo := map[string]Assessment{}
	visiting := map[string]bool{}
	var resolve func(string) (Assessment, error)
	resolve = func(id string) (Assessment, error) {
		if visiting[id] {
			return Assessment{}, fmt.Errorf("coeffect dependency cycle involving %s", id)
		}
		if existing, ok := memo[id]; ok {
			return existing, nil
		}
		p := providers[id]
		visiting[id] = true
		a := Assessment{ProviderID: id, Eligible: true}
		for _, req := range p.Requires {
			var matches []string
			for _, otherID := range sortedKeys(providers) {
				other := providers[otherID]
				if other.Scope == p.Scope && other.Offers == req.Contract &&
					(req.ProviderID == "" || req.ProviderID == otherID) {
					matches = append(matches, otherID)
				}
			}
			if len(matches) == 0 {
				if a.Eligible {
					a.Eligible, a.Reason = false, fmt.Sprintf("missing %s in scope %s", req.Contract.ID, p.Scope)
				}
				continue
			}
			if len(matches) > 1 {
				if a.Eligible {
					a.Eligible, a.Reason = false, fmt.Sprintf("ambiguous %s in scope %s", req.Contract.ID, p.Scope)
				}
				continue
			}
			dependency, err := resolve(matches[0])
			if err != nil {
				return Assessment{}, err
			}
			if !dependency.Eligible {
				if a.Eligible {
					a.Eligible, a.Reason = false, fmt.Sprintf("dependency %s ineligible: %s", matches[0], dependency.Reason)
				}
				continue
			}
			a.Bindings = append(a.Bindings, Edge{req.Contract.ID, matches[0]})
		}
		visiting[id] = false
		memo[id] = a
		if a.Eligible {
			out.ActivationOrder = append(out.ActivationOrder, id)
		}
		return a, nil
	}
	for _, id := range sortedKeys(providers) {
		if _, err := resolve(id); err != nil {
			return Snapshot{}, err
		}
	}
	for _, id := range sortedKeys(memo) {
		out.Providers = append(out.Providers, memo[id])
	}
	return out, nil
}

func TestIndexedOffersPreservePreviousDependencyResolution(t *testing.T) {
	// Keep a fixed seed. Include duplicated offers, cross-scope providers,
	// explicitly pinned providers, incompatible shapes, missing dependencies,
	// and cyclic graphs. Error texts and ordered snapshots must stay identical.
	random := rand.New(rand.NewSource(20261020))
	scopes := []string{"tenant-a", "tenant-b", "host-local"}
	contracts := []Contract{
		{ID: "read", Version: "1", Shape: "shape-a"},
		{ID: "read", Version: "1", Shape: "shape-b"},
		{ID: "read", Version: "2", Shape: "shape-a"},
		{ID: "write", Version: "1", Shape: "shape-a"},
		{ID: "observe", Version: "1", Shape: "shape-a"},
	}
	for scenario := 0; scenario < 400; scenario++ {
		count := 1 + random.Intn(17)
		providers := make(map[string]Provider, count)
		for i := 0; i < count; i++ {
			id := fmt.Sprintf("plugin-%03d", i)
			p := Provider{
				ID: id,
				Scope: scopes[random.Intn(len(scopes))],
				Offers: contracts[random.Intn(len(contracts))],
			}
			seenContracts := map[string]bool{}
			for n := random.Intn(5); n > 0; n-- {
				contract := contracts[random.Intn(len(contracts))]
				if seenContracts[contract.ID] {
					continue
				}
				seenContracts[contract.ID] = true
				dep := Dependency{Contract: contract}
				switch random.Intn(4) {
				case 0:
					dep.ProviderID = fmt.Sprintf("plugin-%03d", random.Intn(count))
				case 1:
					dep.ProviderID = "missing-provider"
				}
				p.Requires = append(p.Requires, dep)
			}
			providers[id] = p
		}

		want, wantErr := evaluateWithProviderScan(providers)
		got, gotErr := evaluate(providers)
		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("scenario %d: old error %v, indexed error %v", scenario, wantErr, gotErr)
		}
		if wantErr != nil {
			if wantErr.Error() != gotErr.Error() {
				t.Fatalf("scenario %d: old error %q, indexed error %q", scenario, wantErr, gotErr)
			}
			continue
		}
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("scenario %d: old=%+v indexed=%+v", scenario, want, got)
		}
	}
}

func TestIndexedOffersPreserveExplicitSelectionInExactScope(t *testing.T) {
	read := Contract{ID: "read", Version: "1", Shape: "exact"}
	otherShape := Contract{ID: "read", Version: "1", Shape: "other"}
	providers := map[string]Provider{
		"a": {ID: "a", Scope: "tenant-a", Offers: read},
		"b": {ID: "b", Scope: "tenant-a", Offers: read},
		"c": {ID: "c", Scope: "tenant-b", Offers: read},
		"d": {ID: "d", Scope: "tenant-a", Offers: otherShape},
		"client": {ID: "client", Scope: "tenant-a",
			Offers: Contract{ID: "client", Version: "1", Shape: "exact"},
			Requires: []Dependency{{Contract: read, ProviderID: "b"}}},
	}
	result, err := evaluate(providers)
	if err != nil {
		t.Fatal(err)
	}
	client, ok := result.Assessment("client")
	if !ok || !client.Eligible ||
		!reflect.DeepEqual(client.Bindings, []Edge{{ContractID: "read", ProviderID: "b"}}) {
		t.Fatalf("explicit binding selected the wrong offer: %+v", client)
	}
	// A different scope or an equal ID with a different shape must not match.
	for _, bad := range []string{"c", "d", "missing-provider"} {
		modified := make(map[string]Provider, len(providers))
		for id, p := range providers {
			modified[id] = p
		}
		clientProvider := modified["client"]
		clientProvider.Requires = []Dependency{{Contract: read, ProviderID: bad}}
		modified["client"] = clientProvider
		out, err := evaluate(modified)
		if err != nil {
			t.Fatal(err)
		}
		found, ok := out.Assessment("client")
		if !ok || found.Eligible {
			t.Fatalf("explicit provider %s unexpectedly eligible: %+v", bad, found)
		}
	}
}

func benchmarkDependencyRegistry(n int) map[string]Provider {
	providers := make(map[string]Provider, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("provider-%04d", i)
		contract := Contract{ID: id, Version: "1", Shape: "exact"}
		p := Provider{ID: id, Scope: "tenant-a", Offers: contract}
		if i != 0 {
			previous := fmt.Sprintf("provider-%04d", i-1)
			p.Requires = []Dependency{{Contract: Contract{ID: previous, Version: "1", Shape: "exact"}}}
		}
		providers[id] = p
	}
	return providers
}

func BenchmarkDependencyOfferLookup(b *testing.B) {
	for _, count := range []int{64, 256} {
		providers := benchmarkDependencyRegistry(count)
		b.Run(fmt.Sprintf("indexed-%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := evaluate(providers); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("previous-scan-%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := evaluateWithProviderScan(providers); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// The offer index must not depend on insertion order, even though map
// iteration order changes across calls.
func TestIndexedOffersActivationOrderIsDeterministic(t *testing.T) {
	providers := benchmarkDependencyRegistry(40)
	a, err := evaluate(providers)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		b, err := evaluate(providers)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(a, b) {
			t.Fatal("dependency resolution depends on Go map iteration")
		}
	}
	if !sort.StringsAreSorted(a.ActivationOrder) {
		t.Fatalf("chain activation order must be stable: %+v", a.ActivationOrder)
	}
}
