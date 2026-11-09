package composition

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func selectedBinding(gen uint64, id string) Binding {
	return Binding{ContractID: storage.ID, ContractVersion: storage.Version, ProviderID: id, ArtifactDigest: "sha256:" + strings.Repeat("a", 64), Generation: gen}
}

func selectedFixture(t *testing.T) (*Registry, Selection, *Coordinator) {
	t.Helper()
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{
		provider("v1", "tenant-A", storage),
		provider("v2", "tenant-A", storage),
	}, nil); err != nil {
		t.Fatal(err)
	}
	s, err := r.Select("v2", "tenant-A")
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(selectedBinding(1, "v1"), NewScope())
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewCoordinator(router, allow)
	if err != nil {
		t.Fatal(err)
	}
	return r, s, c
}

func validLifecycle() Lifecycle {
	return Lifecycle{
		Prepare: func(_ context.Context, _ *Scope) error { return nil },
		Ready:   func(_ context.Context, _ Binding) error { return nil },
	}
}

func TestSelectionRejectsWrongScopeAndIneligibleProvider(t *testing.T) {
	r, _, _ := selectedFixture(t)
	if _, err := r.Select("v2", "tenant-B"); !errors.Is(err, ErrSelectionMismatch) {
		t.Fatalf("cross-scope selection: %v", err)
	}
	if _, err := r.Select("missing", "tenant-A"); !errors.Is(err, ErrSelectionMismatch) {
		t.Fatalf("missing provider selected: %v", err)
	}
	_, err := r.Apply(1, []Provider{
		provider("broken", "tenant-A", app, required(Contract{ID: "absent", Version: "1", Shape: "x"})),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Select("broken", "tenant-A"); !errors.Is(err, ErrProviderIneligible) {
		t.Fatalf("ineligible provider selected: %v", err)
	}
}

func TestSelectedCutoverRechecksIdentityAndGatesInvocations(t *testing.T) {
	registry, selected, c := selectedFixture(t)
	mismatch := selectedBinding(2, "different")
	if err := c.ReplaceSelected(context.Background(), registry, selected, 1, mismatch, validLifecycle()); !errors.Is(err, ErrSelectionMismatch) {
		t.Fatalf("forged provider identity accepted: %v", err)
	}
	if err := c.ReplaceSelected(context.Background(), nil, selected, 1, selectedBinding(2, "v2"), validLifecycle()); !errors.Is(err, ErrSelectionRequired) {
		t.Fatalf("missing registry accepted: %v", err)
	}
	if err := c.ReplaceSelected(context.Background(), registry, selected, 1, selectedBinding(2, "v2"), validLifecycle()); err != nil {
		t.Fatal(err)
	}
	call, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if call.Binding.ProviderID != "v2" {
		t.Fatalf("wrong cutover provider: %s", call.Binding.ProviderID)
	}
	call.Release()
	if err := c.Replace(context.Background(), 2, selectedBinding(3, "v3"), validLifecycle()); !errors.Is(err, ErrSelectionRequired) {
		t.Fatalf("raw API bypass: %v", err)
	}
	// A change in declarations invalidates the registry-bound view until
	// a host explicitly reselects and republishes a fresh generation.
	if _, err := registry.Apply(selected.Revision, nil, []string{"v2"}); err != nil {
		t.Fatal(err)
	}
	if lease, err := c.Acquire(); !errors.Is(err, ErrStaleRegistry) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("stale selected provider served: %v", err)
	}
	if err := c.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryChangeDuringPrepareRejectsStalePublication(t *testing.T) {
	r, selected, c := selectedFixture(t)
	started := make(chan struct{})
	continuePrepare := make(chan struct{})
	var mu sync.Mutex
	cleaned := 0
	outcome := make(chan error, 1)
	go func() {
		outcome <- c.ReplaceSelected(context.Background(), r, selected, 1, selectedBinding(2, "v2"), Lifecycle{
			Prepare: func(_ context.Context, s *Scope) error {
				if err := s.Track(func() error {
					mu.Lock()
					cleaned++
					mu.Unlock()
					return nil
				}); err != nil {
					return err
				}
				close(started)
				<-continuePrepare
				return nil
			},
			Ready: func(_ context.Context, _ Binding) error { return nil },
		})
	}()
	<-started
	// The provider can disappear before a slow stage reports Ready.
	if _, err := r.Apply(selected.Revision, nil, []string{"v2"}); err != nil {
		t.Fatal(err)
	}
	close(continuePrepare)
	if err := <-outcome; !errors.Is(err, ErrStaleRegistry) {
		t.Fatalf("stale publication: %v", err)
	}
	mu.Lock()
	got := cleaned
	mu.Unlock()
	if got != 1 {
		t.Fatalf("failed stage cleanup count %d", got)
	}
	old, err := c.Acquire()
	if err != nil {
		t.Fatalf("previous live generation lost: %v", err)
	}
	if old.Binding.ProviderID != "v1" {
		t.Fatal("new generation published after registry edit")
	}
	old.Release()
}

func TestSelectionRejectsAlteredDependencyBindings(t *testing.T) {
	r := NewRegistry()
	snap, err := r.Apply(0, []Provider{
		provider("storage", "tenant-A", storage),
		provider("consumer", "tenant-A", app, required(storage)),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := r.Select("consumer", "tenant-A")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Dependencies) != 1 {
		t.Fatal("dependency was not selected")
	}
	selected.Dependencies[0].ProviderID = "wrong"
	r.mu.Lock()
	err = r.validateSelectionLocked(selected, Binding{
		ContractID: app.ID, ContractVersion: app.Version, ProviderID: "consumer",
	})
	r.mu.Unlock()
	if !errors.Is(err, ErrSelectionMismatch) {
		t.Fatalf("forged dependency accepted: %v", err)
	}
	if snap.Revision != 1 {
		t.Fatal("bad initial revision")
	}
}

func TestRestoreSelectedAfterRegistryRemoval(t *testing.T) {
	r, selected, c := selectedFixture(t)
	if err := c.ReplaceSelected(context.Background(), r, selected, 1, selectedBinding(2, "v2"), validLifecycle()); err != nil {
		t.Fatal(err)
	}
	if err := c.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	next, err := r.Apply(selected.Revision, []Provider{provider("v3", "tenant-A", storage)}, []string{"v2"})
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := r.Select("v3", "tenant-A")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Revision != next.Revision {
		t.Fatal("wrong new selection revision")
	}
	if err := c.RestoreSelected(context.Background(), r, fresh, 2, selectedBinding(3, "v3"), validLifecycle()); err != nil {
		t.Fatal(err)
	}
	lease, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if lease.Binding.ProviderID != "v3" {
		t.Fatal("failed to restore selected service")
	}
	lease.Release()
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestUnrelatedRegistryUpdateDoesNotDisruptSelectedService(t *testing.T) {
	r, selected, c := selectedFixture(t)
	if err := c.ReplaceSelected(context.Background(), r, selected, 1, selectedBinding(2, "v2"), validLifecycle()); err != nil {
		t.Fatal(err)
	}
	next, err := r.Apply(selected.Revision, []Provider{provider("tenantB", "tenant-B", app)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next.Revision == selected.Revision {
		t.Fatal("revision did not advance")
	}
	lease, err := c.Acquire()
	if err != nil {
		t.Fatalf("unrelated edit caused avoidable service outage: %v", err)
	}
	if lease.Binding.ProviderID != "v2" {
		t.Fatal("binding changed without cutover")
	}
	lease.Release()
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestProviderReRegistrationCannotReviveOldSelection(t *testing.T) {
	r, selected, c := selectedFixture(t)
	if err := c.ReplaceSelected(context.Background(), r, selected, 1, selectedBinding(2, "v2"), validLifecycle()); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Apply(selected.Revision, nil, []string{"v2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Apply(selected.Revision+1, []Provider{provider("v2", "tenant-A", storage)}, nil); err != nil {
		t.Fatal(err)
	}
	if lease, err := c.Acquire(); !errors.Is(err, ErrStaleRegistry) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("removed and re-registered provider silently resurrected: %v", err)
	}
	fresh, err := r.Select("v2", "tenant-A")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Epoch <= selected.Epoch {
		t.Fatal("provider epoch did not advance")
	}
}

func TestSelectedStageRejectsRevokedHostAuthority(t *testing.T) {
	r, selected, c := selectedFixture(t)
	denied := false
	c.guard = func(b Binding) error {
		if b.ProviderID == "v2" && denied {
			return errors.New("fencing revoked")
		}
		return nil
	}
	cleaned := 0
	lc := Lifecycle{
		Prepare: func(_ context.Context, s *Scope) error { return s.Track(func() error { cleaned++; return nil }) },
		Ready:   func(_ context.Context, _ Binding) error { denied = true; return nil },
	}
	if err := c.ReplaceSelected(context.Background(), r, selected, 1, selectedBinding(2, "v2"), lc); err == nil {
		t.Fatal("revoked guard allowed cutover")
	}
	if cleaned != 1 {
		t.Fatalf("staged effect not cleaned: %d", cleaned)
	}
	lease, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if lease.Binding.ProviderID != "v1" {
		t.Fatal("previous provider not retained")
	}
	lease.Release()
}
