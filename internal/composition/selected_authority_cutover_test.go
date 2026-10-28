package composition

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// These tests intentionally use a synthetic Host-owned guard. The Composition
// selection is only a declaration snapshot: it cannot mint authority or attest
// an artifact. No source/business-model acceptance or production Host is used.
var (
	cutoverReadContract  = Contract{ID: "reference.readonly.list", Version: "1.0.0", Shape: "list-shape-v1"}
	cutoverDataContract  = Contract{ID: "reference.readonly.source", Version: "1.0.0", Shape: "source-shape-v1"}
	errCutoverHostDenied = errors.New("synthetic host permission revoked")
)

func newSelectedCutoverFixture(t *testing.T) (*Registry, *Coordinator, *atomic.Bool, *Scope) {
	t.Helper()
	registry := NewRegistry()
	_, err := registry.Apply(0, []Provider{
		{ID: "read-data", Scope: "tenant-a", Offers: cutoverDataContract},
		{ID: "read-service-v2", Scope: "tenant-a", Offers: cutoverReadContract, Requires: []Dependency{{Contract: cutoverDataContract, ProviderID: "read-data"}}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldScope := NewScope()
	first := Binding{ContractID: cutoverReadContract.ID, ContractVersion: cutoverReadContract.Version, ProviderID: "read-service-v1", ArtifactDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1}
	router, err := NewRouter(first, oldScope)
	if err != nil {
		t.Fatal(err)
	}
	allowed := new(atomic.Bool)
	allowed.Store(true)
	coordinator, err := NewCoordinator(router, func(b Binding) error {
		if b.ProviderID == "read-service-v2" && !allowed.Load() {
			return errCutoverHostDenied
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return registry, coordinator, allowed, oldScope
}

func selectedCutoverBinding() Binding {
	return Binding{ContractID: cutoverReadContract.ID, ContractVersion: cutoverReadContract.Version, ProviderID: "read-service-v2", ArtifactDigest: "sha256:" + strings.Repeat("b", 64), Generation: 2}
}

func TestSelectedCutoverRejectsHostRevocationAfterPrepare(t *testing.T) {
	registry, coordinator, allowed, _ := newSelectedCutoverFixture(t)
	selected, err := registry.Select("read-service-v2", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	entered, proceed := make(chan struct{}), make(chan struct{})
	finished := make(chan error, 1)
	var cleaned atomic.Int32
	go func() {
		finished <- coordinator.ReplaceSelected(context.Background(), registry, selected, 1, selectedCutoverBinding(), Lifecycle{
			Prepare: func(_ context.Context, scope *Scope) error {
				if err := scope.Track(func() error { cleaned.Add(1); return nil }); err != nil {
					return err
				}
				close(entered)
				<-proceed
				return nil
			},
			Ready: func(context.Context, Binding) error { return nil },
		})
	}()
	<-entered
	allowed.Store(false) // The revocation occurs after staging, before Publish.
	close(proceed)
	if err := <-finished; !errors.Is(err, errCutoverHostDenied) {
		t.Fatalf("revoked Host still published next generation: %v", err)
	}
	if cleaned.Load() != 1 || coordinator.PendingRetirements() != 0 {
		t.Fatalf("failed stage did not clean exactly once: cleanup=%d retirements=%d", cleaned.Load(), coordinator.PendingRetirements())
	}
	lease, err := coordinator.Acquire()
	if err != nil || lease.Binding.ProviderID != "read-service-v1" {
		t.Fatalf("rejected stage displaced the old generation: lease=%+v err=%v", lease, err)
	}
	lease.Release()
}

func TestSelectedCutoverRejectsTransitiveDependencyChangeDuringPrepare(t *testing.T) {
	registry, coordinator, _, _ := newSelectedCutoverFixture(t)
	selected, err := registry.Select("read-service-v2", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	entered, proceed := make(chan struct{}), make(chan struct{})
	finished := make(chan error, 1)
	var cleaned atomic.Int32
	go func() {
		finished <- coordinator.ReplaceSelected(context.Background(), registry, selected, 1, selectedCutoverBinding(), Lifecycle{
			Prepare: func(_ context.Context, scope *Scope) error {
				if err := scope.Track(func() error { cleaned.Add(1); return nil }); err != nil {
					return err
				}
				close(entered)
				<-proceed
				return nil
			},
			Ready: func(context.Context, Binding) error { return nil },
		})
	}()
	<-entered
	// The direct consumer declaration is unchanged. Updating the selected
	// dependency's own dependency must invalidate the captured transitive epoch.
	aux := Contract{ID: "reference.readonly.auxiliary", Version: "1.0.0", Shape: "aux-shape-v1"}
	_, err = registry.Apply(selected.Revision, []Provider{
		{ID: "auxiliary", Scope: "tenant-a", Offers: aux},
		{ID: "read-data", Scope: "tenant-a", Offers: cutoverDataContract, Requires: []Dependency{{Contract: aux, ProviderID: "auxiliary"}}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	close(proceed)
	if err := <-finished; !errors.Is(err, ErrStaleRegistry) {
		t.Fatalf("stale transitive binding published: %v", err)
	}
	if cleaned.Load() != 1 || coordinator.PendingRetirements() != 0 {
		t.Fatalf("failed stage leaked effects or retired live service: cleanup=%d retirements=%d", cleaned.Load(), coordinator.PendingRetirements())
	}
	lease, err := coordinator.Acquire()
	if err != nil || lease.Binding.ProviderID != "read-service-v1" {
		t.Fatalf("stale stage displaced v1: lease=%+v err=%v", lease, err)
	}
	lease.Release()
}

func TestSelectedCutoverPinsInFlightAndKeepsTimedOutDrainPending(t *testing.T) {
	registry, coordinator, allowed, oldScope := newSelectedCutoverFixture(t)
	selected, err := registry.Select("read-service-v2", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	var oldCleaned, newCleaned atomic.Int32
	if err := oldScope.Track(func() error { oldCleaned.Add(1); return nil }); err != nil {
		t.Fatal(err)
	}
	v1a, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	v1b, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	err = coordinator.ReplaceSelected(context.Background(), registry, selected, 1, selectedCutoverBinding(), Lifecycle{
		Prepare: func(_ context.Context, scope *Scope) error {
			return scope.Track(func() error { newCleaned.Add(1); return nil })
		},
		Ready: func(_ context.Context, b Binding) error {
			if b.ProviderID != "read-service-v2" {
				return errors.New("wrong provider prepared")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	v2, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if v1a.Binding.ProviderID != "read-service-v1" || v1b.Binding.ProviderID != "read-service-v1" || v2.Binding.ProviderID != "read-service-v2" {
		t.Fatalf("requests switched generation in flight: v1a=%+v v1b=%+v v2=%+v", v1a.Binding, v1b.Binding, v2.Binding)
	}
	allowed.Store(false)
	if blocked, err := coordinator.Acquire(); !errors.Is(err, errCutoverHostDenied) {
		if blocked != nil {
			blocked.Release()
		}
		t.Fatalf("revoked v2 admitted new request: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := coordinator.Reap(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("active v1 was prematurely drained: %v", err)
	}
	if coordinator.PendingRetirements() != 1 || oldCleaned.Load() != 0 {
		t.Fatalf("timed-out drain was reported clean: pending=%d cleaned=%d", coordinator.PendingRetirements(), oldCleaned.Load())
	}
	v1a.Release()
	v1b.Release()
	if err := coordinator.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if coordinator.PendingRetirements() != 0 || oldCleaned.Load() != 1 || newCleaned.Load() != 0 {
		t.Fatalf("incorrect cleanup boundary: pending=%d old=%d new=%d", coordinator.PendingRetirements(), oldCleaned.Load(), newCleaned.Load())
	}
	v2.Release()
}
