package composition

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestTypedServiceRestoreRequiresExplicitAdmissionAndCleansFailedCandidate(t *testing.T) {
	r, first, next := serviceRegistry(t)
	var denied atomic.Bool
	guard := func(Binding) error {
		if denied.Load() {
			return errors.New("host admission denied")
		}
		return nil
	}
	s, err := NewService(context.Background(), r, first, selectedBinding(1, "v1"), guard, func(context.Context, int) (int, error) { return 1, nil }, validLifecycle())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	invoke := func(ctx context.Context, _ int) (int, error) {
		b, _ := InvocationBinding(ctx)
		if b.Generation != 2 {
			t.Fatal("restored identity lost")
		}
		return 2, nil
	}
	cleaned := 0
	lifecycle := validLifecycle()
	lifecycle.Prepare = func(_ context.Context, scope *Scope) error {
		return scope.Track(func() error { cleaned++; return nil })
	}
	bad := lifecycle
	bad.Ready = func(context.Context, Binding) error { return errors.New("not ready") }
	if err := s.Restore(context.Background(), next, 1, selectedBinding(2, "v2"), invoke, bad); err == nil || cleaned != 1 {
		t.Fatal("unready candidate retained")
	}
	denied.Store(true)
	if err := s.Restore(context.Background(), next, 1, selectedBinding(2, "v2"), invoke, lifecycle); err == nil || cleaned != 2 {
		t.Fatal("denied restoration published or leaked")
	}
	if _, _, err := s.Invoke(context.Background(), 0); !errors.Is(err, ErrUnbound) {
		t.Fatal("failed candidate became callable")
	}
	denied.Store(false)
	if err := s.Restore(context.Background(), next, 1, selectedBinding(1, "v2"), invoke, lifecycle); err == nil {
		t.Fatal("reused generation restored")
	}
	if err := s.Restore(context.Background(), next, 1, selectedBinding(2, "v2"), invoke, lifecycle); err != nil {
		t.Fatal(err)
	}
	value, binding, err := s.Invoke(context.Background(), 0)
	if err != nil || value != 2 || binding.ProviderID != "v2" || binding.Generation != 2 {
		t.Fatalf("restoration: %v %v %v", value, binding, err)
	}
	if err := s.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.instances) != 0 {
		t.Fatal("typed instances retained after final cleanup")
	}
}

func TestServiceInvocationContextBindsHostAdapterWithoutChangingCaller(t *testing.T) {
	r, first, _ := serviceRegistry(t)
	parent := context.Background()
	if _, ok := InvocationBinding(parent); ok {
		t.Fatal("plain context has instance identity")
	}
	s, err := NewService(parent, r, first, selectedBinding(1, "v1"), allow, func(ctx context.Context, input int) (int, error) {
		binding, ok := InvocationBinding(ctx)
		if !ok || binding != selectedBinding(1, "v1") {
			t.Fatal("adapter did not receive its pinned instance")
		}
		binding.ProviderID = "forged"
		again, _ := InvocationBinding(ctx)
		if again.ProviderID != "v1" {
			t.Fatal("adapter mutated host identity")
		}
		return input, nil
	}, validLifecycle())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Invoke(parent, 1); err != nil {
		t.Fatal(err)
	}
	if _, ok := InvocationBinding(parent); ok {
		t.Fatal("instance identity escaped into caller context")
	}
	if err := s.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Reap(parent); err != nil {
		t.Fatal(err)
	}
}

func TestServiceShutdownRetainsCancelledDrainAndReportsCleanupFailure(t *testing.T) {
	r, selected, _ := serviceRegistry(t)
	failure := errors.New("temporary resource cleanup failed")
	cleanups := 0
	lifecycle := validLifecycle()
	lifecycle.Prepare = func(_ context.Context, scope *Scope) error {
		return scope.Track(func() error { cleanups++; return failure })
	}
	s, err := NewService(context.Background(), r, selected, selectedBinding(1, "v1"), allow, func(context.Context, int) (int, error) { return 42, nil }, lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Shutdown(context.Background(), 99); !errors.Is(err, ErrStaleGeneration) || cleanups != 0 {
		t.Fatal("wrong generation silently shut down active service")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Shutdown(ctx, 1); !errors.Is(err, context.Canceled) || cleanups != 0 {
		t.Fatal("cancelled drain lost cleanup responsibility")
	}
	if _, _, err := s.Invoke(context.Background(), 0); !errors.Is(err, ErrUnbound) {
		t.Fatal("shutdown continued accepting calls")
	}
	if err := s.Reap(context.Background()); !errors.Is(err, failure) || cleanups != 1 {
		t.Fatal("cleanup error hidden")
	}
	if err := s.Reap(context.Background()); err != nil || cleanups != 1 {
		t.Fatal("failed inverse retried")
	}
}

func serviceRegistry(t *testing.T) (*Registry, Selection, Selection) {
	t.Helper()
	r := NewRegistry()
	if _, err := r.Apply(0, []Provider{provider("v1", "tenant-A", storage), provider("v2", "tenant-A", storage)}, nil); err != nil {
		t.Fatal(err)
	}
	a, err := r.Select("v1", "tenant-A")
	if err != nil {
		t.Fatal(err)
	}
	z, err := r.Select("v2", "tenant-A")
	if err != nil {
		t.Fatal(err)
	}
	return r, a, z
}

func TestServiceExecutesPinnedInstanceDuringReplacement(t *testing.T) {
	r, first, next := serviceRegistry(t)
	started, finish := make(chan struct{}), make(chan struct{})
	var cleaned atomic.Bool
	lifecycle := validLifecycle()
	lifecycle.Prepare = func(_ context.Context, scope *Scope) error {
		return scope.Track(func() error { cleaned.Store(true); return nil })
	}
	s, err := NewService(context.Background(), r, first, selectedBinding(1, "v1"), allow, func(_ context.Context, input int) (string, error) {
		close(started)
		<-finish
		if cleaned.Load() {
			return "", errors.New("instance disposed in-flight")
		}
		return "old", nil
	}, lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		value, binding, err := s.Invoke(context.Background(), 7)
		if err == nil && (value != "old" || binding.Generation != 1) {
			err = errors.New("old request changed implementation")
		}
		done <- err
	}()
	<-started
	if err := s.Replace(context.Background(), next, 1, selectedBinding(2, "v2"), func(context.Context, int) (string, error) { return "new", nil }, validLifecycle()); err != nil {
		t.Fatal(err)
	}
	value, binding, err := s.Invoke(context.Background(), 8)
	if err != nil || value != "new" || binding.Generation != 2 {
		t.Fatalf("new invocation: %q %+v %v", value, binding, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Reap(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("early drain: %v", err)
	}
	if cleaned.Load() {
		t.Fatal("in-flight instance cleaned")
	}
	close(finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := s.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cleaned.Load() {
		t.Fatal("old instance not cleaned")
	}
	if err := s.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Invoke(context.Background(), 9); !errors.Is(err, ErrUnbound) {
		t.Fatalf("withdraw: %v", err)
	}
	if err := s.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.instances) != 0 {
		t.Fatal("retired executable retained")
	}
}

func TestServiceRejectsRevocationDependencyLossAndFailedReady(t *testing.T) {
	r, first, next := serviceRegistry(t)
	var denied atomic.Bool
	calls := 0
	s, err := NewService(context.Background(), r, first, selectedBinding(1, "v1"), func(Binding) error {
		if denied.Load() {
			return errors.New("revoked")
		}
		return nil
	}, func(context.Context, int) (int, error) { calls++; return 42, nil }, validLifecycle())
	if err != nil {
		t.Fatal(err)
	}
	bad := validLifecycle()
	bad.Ready = func(context.Context, Binding) error { return errors.New("not ready") }
	if err := s.Replace(context.Background(), next, 1, selectedBinding(2, "v2"), func(context.Context, int) (int, error) { t.Fatal("failed candidate called"); return 0, nil }, bad); err == nil {
		t.Fatal("accepted unready")
	}
	value, binding, err := s.Invoke(context.Background(), 1)
	if err != nil || value != 42 || binding.Generation != 1 {
		t.Fatalf("old service lost: %v", err)
	}
	denied.Store(true)
	if _, _, err := s.Invoke(context.Background(), 1); err == nil || calls != 1 {
		t.Fatal("revoked invocation executed")
	}
	denied.Store(false)
	if _, err := r.Apply(1, nil, []string{"v1"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Invoke(context.Background(), 1); err == nil || calls != 1 {
		t.Fatal("removed provider executed")
	}
	if err := s.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestServiceStartupPanicCleansAndInvocationPanicReleases(t *testing.T) {
	r, first, _ := serviceRegistry(t)
	cleaned := false
	lifecycle := validLifecycle()
	lifecycle.Prepare = func(_ context.Context, scope *Scope) error {
		return scope.Track(func() error { cleaned = true; return nil })
	}
	lifecycle.Ready = func(context.Context, Binding) error { panic("ready") }
	if _, err := NewService(context.Background(), r, first, selectedBinding(1, "v1"), allow, func(context.Context, int) (int, error) { return 0, nil }, lifecycle); err == nil || !cleaned {
		t.Fatal("startup panic leaked")
	}
	s, err := NewService(context.Background(), r, first, selectedBinding(1, "v1"), allow, func(context.Context, int) (int, error) { panic("invoke") }, validLifecycle())
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected invocation panic")
			}
		}()
		_, _, _ = s.Invoke(context.Background(), 1)
	}()
	if err := s.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}
