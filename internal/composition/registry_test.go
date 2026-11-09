package composition

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

var storage = Contract{ID: "std.data.read", Version: "1.0.0", Shape: "shape-1"}
var app = Contract{ID: "std.app.list", Version: "1.0.0", Shape: "shape-2"}

func provider(id, scope string, c Contract, needs ...Dependency) Provider {
	return Provider{ID: id, Scope: scope, Offers: c, Requires: needs}
}

func required(c Contract) Dependency { return Dependency{Contract: c} }

func TestCoeffectReactiveLossAndRecovery(t *testing.T) {
	r := NewRegistry()
	s, err := r.Apply(0, []Provider{provider("consumer", "tenant-A", app, required(storage))}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.Assessment("consumer")
	if a.Eligible || !strings.Contains(a.Reason, "missing") {
		t.Fatalf("missing coeffect not detected: %+v", a)
	}
	s, err = r.Apply(1, []Provider{provider("data", "tenant-A", storage)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.ActivationOrder, []string{"data", "consumer"}) {
		t.Fatalf("dependency activation order: %v", s.ActivationOrder)
	}
	a, _ = s.Assessment("consumer")
	if !a.Eligible || len(a.Bindings) != 1 || a.Bindings[0].ProviderID != "data" {
		t.Fatalf("dependency selection: %+v", a)
	}
	s, err = r.Apply(2, nil, []string{"data"})
	if err != nil {
		t.Fatal(err)
	}
	a, _ = s.Assessment("consumer")
	if a.Eligible || len(s.Transitions) != 2 {
		t.Fatalf("coeffect loss: %+v", s)
	}
	if _, err = r.Apply(2, []Provider{provider("data", "tenant-A", storage)}, nil); !errors.Is(err, ErrStaleRegistry) {
		t.Fatal("stale expected revision accepted")
	}
	s, err = r.Apply(3, []Provider{provider("data", "tenant-A", storage)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, _ = s.Assessment("consumer")
	if !a.Eligible {
		t.Fatalf("provider recovery not reflected: %+v", a)
	}
}

func TestCoeffectAmbiguityExplicitWiringAndScope(t *testing.T) {
	r := NewRegistry()
	providers := []Provider{
		provider("A", "tenant-A", storage), provider("B", "tenant-A", storage),
		provider("separate", "tenant-B", storage), provider("consumer", "tenant-A", app, required(storage)),
	}
	s, err := r.Apply(0, providers, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.Assessment("consumer")
	if a.Eligible || !strings.Contains(a.Reason, "ambiguous") {
		t.Fatalf("ambiguity: %+v", a)
	}
	consumer := provider("consumer", "tenant-A", app, Dependency{Contract: storage, ProviderID: "B"})
	s, err = r.Apply(1, []Provider{consumer}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, _ = s.Assessment("consumer")
	if !a.Eligible || a.Bindings[0].ProviderID != "B" {
		t.Fatalf("wiring: %+v", a)
	}
	consumer.Requires[0].ProviderID = "separate"
	s, err = r.Apply(2, []Provider{consumer}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, _ = s.Assessment("consumer")
	if a.Eligible || !strings.Contains(a.Reason, "missing") {
		t.Fatalf("cross tenant fallback: %+v", a)
	}
}

func TestCoeffectCycleRejectedAtomically(t *testing.T) {
	r := NewRegistry()
	_, err := r.Apply(0, []Provider{provider("A", "tenant-A", storage)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Apply(1, []Provider{
		provider("A", "tenant-A", storage, required(app)),
		provider("B", "tenant-A", app, required(storage)),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("missed cycle: %v", err)
	}
	s := r.Snapshot()
	if s.Revision != 1 || len(s.Providers) != 1 || !s.Providers[0].Eligible {
		t.Fatalf("invalid mutation committed: %+v", s)
	}
}

func TestRegistrySnapshotDefensiveCopyAndConcurrentCAS(t *testing.T) {
	r := NewRegistry()
	_, err := r.Apply(0, []Provider{provider("A", "tenant-A", storage)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	success := make(chan bool, 2)
	for _, id := range []string{"B", "C"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, err := r.Apply(1, []Provider{provider(id, "tenant-A", app)}, nil)
			success <- err == nil
		}(id)
	}
	wg.Wait()
	close(success)
	n := 0
	for ok := range success {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("CAS winners %d, expected 1", n)
	}
	s := r.Snapshot()
	s.Providers[0].ProviderID = "modified"
	if r.Snapshot().Providers[0].ProviderID == "modified" {
		t.Fatal("snapshot aliases internal state")
	}
}

func TestContractShapeAndDuplicatesRejected(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{provider("A", "tenant-A", Contract{ID: "x", Version: "1"})}, nil); err == nil {
		t.Fatal("missing contract shape accepted")
	}
	if _, err := r.Apply(0, []Provider{provider("A", "tenant-A", storage), provider("A", "tenant-A", storage)}, nil); err == nil {
		t.Fatal("duplicate update accepted")
	}
	s, err := r.Apply(0, []Provider{provider("A", "tenant-A", storage)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	consumer := provider("consumer", "tenant-A", app, required(Contract{ID: storage.ID, Version: storage.Version, Shape: "different"}))
	s, err = r.Apply(s.Revision, []Provider{consumer}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.Assessment("consumer")
	if a.Eligible {
		t.Fatalf("shape drift accepted: %+v", a)
	}
}

func TestRebindingToAnotherProviderEmitsTransition(t *testing.T) {
	r := NewRegistry()
	s, err := r.Apply(0, []Provider{provider("A", "tenant-A", storage), provider("consumer", "tenant-A", app, required(storage))}, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, err = r.Apply(s.Revision, []Provider{provider("B", "tenant-A", storage)}, []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	a, ok := s.Assessment("consumer")
	if !ok || !a.Eligible || a.Bindings[0].ProviderID != "B" {
		t.Fatalf("rebinding: %+v", a)
	}
	found := false
	for _, transition := range s.Transitions {
		if transition.ProviderID == "consumer" && transition.From != transition.To {
			found = true
		}
	}
	if !found {
		t.Fatalf("silent rebinding: %+v", s.Transitions)
	}
}

func TestCycleDetectedEvenWithUnrelatedMissingDependency(t *testing.T) {
	r := NewRegistry()
	c := Contract{ID: "std.missing", Version: "1.0.0", Shape: "shape-missing"}
	_, err := r.Apply(0, []Provider{
		provider("A", "tenant-A", storage, required(c), required(app)),
		provider("B", "tenant-A", app, required(storage)),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle suppressed by missing dependency: %v", err)
	}
	if r.Snapshot().Revision != 0 {
		t.Fatal("cycle mutation committed")
	}
}

func TestChangedProviderDefinitionTriggersTransition(t *testing.T) {
	r := NewRegistry()
	s, err := r.Apply(0, []Provider{provider("A", "tenant-A", storage)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	changed := storage
	changed.Shape = "shape-2"
	s, err = r.Apply(s.Revision, []Provider{provider("A", "tenant-A", changed)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Transitions) != 1 || s.Transitions[0].From == s.Transitions[0].To {
		t.Fatalf("silent identity change: %+v", s.Transitions)
	}
}
