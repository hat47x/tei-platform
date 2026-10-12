package consumerprobe

import (
	"context"
	"errors"
	"strings"
	"testing"

	composition "tei-core-composition-probe"
	coreeffect "tei/core/effect"
)

func TestExternalConsumerUsesSelectedProviderAndHostAuthority(t *testing.T) {
	contract := composition.Contract{ID: "std.test.clock", Version: "1.0.0", Shape: "clock-request-v1"}
	registry := composition.NewRegistry()
	initialSnapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "clock-a", Scope: "tenant-A", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("clock-a", "tenant-A")
	if err != nil {
		t.Fatal(err)
	}
	initial := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: "clock-a", ArtifactDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1,
	}
	oldScope := composition.NewScope()
	oldClosed := 0
	if err := oldScope.Track(func() error { oldClosed++; return nil }); err != nil {
		t.Fatal(err)
	}
	denied := errors.New("host denied invocation")
	allow := true
	hostGuard := func(binding composition.Binding) error {
		if !allow || binding.ContractID != contract.ID {
			return denied
		}
		return nil
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selection, initial, oldScope, hostGuard)
	if err != nil {
		t.Fatal(err)
	}
	oldLease, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Apply(initialSnapshot.Revision, []composition.Provider{{
		ID: "clock-b", Scope: "tenant-A", Offers: contract,
	}}, []string{"clock-a"}); err != nil {
		t.Fatal(err)
	}
	if lease, err := coordinator.Acquire(); !errors.Is(err, composition.ErrStaleRegistry) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("stale selected provider served: %v", err)
	}
	newSelection, err := registry.Select("clock-b", "tenant-A")
	if err != nil {
		t.Fatal(err)
	}
	next := initial
	next.ProviderID = "clock-b"
	next.Generation = 2
	next.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	newClosed := 0
	lifecycle := composition.Lifecycle{
		Prepare: func(_ context.Context, scope *composition.Scope) error {
			return scope.Track(func() error { newClosed++; return nil })
		},
		Ready: func(_ context.Context, actual composition.Binding) error {
			if actual != next {
				return errors.New("unexpected staged binding")
			}
			return nil
		},
	}
	if err := coordinator.ReplaceSelected(context.Background(), registry, newSelection, 1, next, lifecycle); err != nil {
		t.Fatal(err)
	}
	newLease, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if oldLease.Binding.ProviderID != "clock-a" || newLease.Binding.ProviderID != "clock-b" || oldClosed != 0 {
		t.Fatal("in-flight provider was replaced or cleaned early")
	}
	oldLease.Release()
	newLease.Release()
	if err := coordinator.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if oldClosed != 1 {
		t.Fatalf("previous provider cleanup count: %d", oldClosed)
	}
	allow = false
	if lease, err := coordinator.Acquire(); !errors.Is(err, denied) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("host authority bypassed: %v", err)
	}
	if err := coordinator.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if newClosed != 1 {
		t.Fatalf("replacement provider cleanup count: %d", newClosed)
	}
	if lease, err := coordinator.Acquire(); !errors.Is(err, composition.ErrUnbound) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("withdrawn provider accepted new invocation: %v", err)
	}
}

func TestExternalConsumerScopeIsCanonicalCoreType(t *testing.T) {
	var direct *coreeffect.Scope = composition.NewScope()
	var legacy *composition.Scope = coreeffect.NewScope()
	if direct == nil || legacy == nil {
		t.Fatal("the aliased Scope is nil")
	}
	if err := direct.Track(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := direct.Close(); err != nil {
		t.Fatal(err)
	}
	if err := direct.Track(func() error { return nil }); !errors.Is(err, composition.ErrScopeClosed) ||
		!errors.Is(err, coreeffect.ErrScopeClosed) {
		t.Fatalf("legacy and canonical errors differ: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
}
