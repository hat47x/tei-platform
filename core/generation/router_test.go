package generation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/core/effect"
)

func fixtureBinding(g uint64) Binding {
	return Binding{ContractID: "service", ContractVersion: "1", ProviderID: "provider", ArtifactDigest: "sha256:" + strings.Repeat("a", 64), Generation: g}
}

func TestLeasePinsOldGenerationUntilDrain(t *testing.T) {
	firstScope := effect.NewScope()
	cleaned := false
	if err := firstScope.Track(func() error { cleaned = true; return nil }); err != nil {
		t.Fatal(err)
	}
	r, err := NewRouter(fixtureBinding(1), firstScope)
	if err != nil {
		t.Fatal(err)
	}
	guard := func(Binding) error { return nil }
	lease, err := r.Acquire(guard)
	if err != nil {
		t.Fatal(err)
	}
	nextScope := effect.NewScope()
	retired, err := r.Publish(1, fixtureBinding(2), nextScope, guard)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Binding.Generation != 1 {
		t.Fatalf("lease generation=%d", lease.Binding.Generation)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := retired.Drain(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("premature drain=%v", err)
	}
	if cleaned {
		t.Fatal("retired scope cleaned while lease active")
	}
	lease.Release()
	lease.Release()
	if err := retired.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cleaned {
		t.Fatal("retired scope was not cleaned")
	}
	newLease, err := r.Acquire(guard)
	if err != nil {
		t.Fatal(err)
	}
	if newLease.Binding.Generation != 2 {
		t.Fatalf("new lease generation=%d", newLease.Binding.Generation)
	}
	newLease.Release()
	if _, err := r.Publish(1, fixtureBinding(3), effect.NewScope(), guard); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("stale publish=%v", err)
	}
	if err := nextScope.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHostGuardAndWithdrawRestore(t *testing.T) {
	s := effect.NewScope()
	r, err := NewRouter(fixtureBinding(1), s)
	if err != nil {
		t.Fatal(err)
	}
	denied := errors.New("host-denied")
	if _, err := r.Acquire(func(Binding) error { return denied }); !errors.Is(err, denied) {
		t.Fatalf("guard=%v", err)
	}
	if _, err := r.Acquire(nil); !errors.Is(err, ErrGuardRequired) {
		t.Fatalf("missing guard=%v", err)
	}
	old, err := r.Withdraw(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Acquire(func(Binding) error { return nil }); !errors.Is(err, ErrUnbound) {
		t.Fatalf("withdraw=%v", err)
	}
	if err := old.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.Activate(1, fixtureBinding(2), effect.NewScope(), func(Binding) error { return denied }); !errors.Is(err, denied) {
		t.Fatalf("restore guard=%v", err)
	}
	restored := effect.NewScope()
	if err := r.Activate(1, fixtureBinding(2), restored, func(Binding) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := r.Activate(1, fixtureBinding(3), effect.NewScope(), func(Binding) error { return nil }); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("duplicate restore=%v", err)
	}
	if err := restored.Close(); err != nil {
		t.Fatal(err)
	}
}
