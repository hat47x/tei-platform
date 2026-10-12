package composition

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func setupCoordinator(t *testing.T) (*Coordinator, *Router, *Scope) {
	t.Helper()
	scope := NewScope()
	r, err := NewRouter(b(1, "v1"), scope)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewCoordinator(r, allow)
	if err != nil {
		t.Fatal(err)
	}
	return c, r, scope
}

func lifecycle(prepare func(*Scope) error, ready func(Binding) error) Lifecycle {
	return Lifecycle{
		Prepare: func(_ context.Context, s *Scope) error { return prepare(s) },
		Ready:   func(_ context.Context, b Binding) error { return ready(b) },
	}
}

func TestCoordinatorStagesBeforeCutoverAndDrains(t *testing.T) {
	c, _, oldScope := setupCoordinator(t)
	var mu sync.Mutex
	var order []string
	if err := oldScope.Track(func() error { mu.Lock(); order = append(order, "old close"); mu.Unlock(); return nil }); err != nil {
		t.Fatal(err)
	}
	old, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	lc := lifecycle(func(s *Scope) error {
		mu.Lock()
		order = append(order, "prepare")
		mu.Unlock()
		return s.Track(func() error { return nil })
	}, func(b Binding) error {
		mu.Lock()
		order = append(order, "ready")
		mu.Unlock()
		if b.ProviderID != "v2" {
			t.Fatal(b)
		}
		return nil
	})
	if err := c.Replace(context.Background(), 1, b(2, "v2"), lc); err != nil {
		t.Fatal(err)
	}
	if c.PendingRetirements() != 1 {
		t.Fatal("old scope lost")
	}
	newLease, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if old.Binding.ProviderID != "v1" || newLease.Binding.ProviderID != "v2" {
		t.Fatal("in-flight changed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Reap(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel should retain retirement: %v", err)
	}
	if c.PendingRetirements() != 1 {
		t.Fatal("retirement lost")
	}
	newLease.Release()
	old.Release()
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	got := strings.Join(order, ",")
	mu.Unlock()
	if got != "prepare,ready,old close" {
		t.Fatalf("wrong order: %s", got)
	}
}

func TestCoordinatorFailedStagingLeavesOldLive(t *testing.T) {
	for _, failure := range []string{"prepare", "ready", "guard", "panic"} {
		t.Run(failure, func(t *testing.T) {
			c, r, _ := setupCoordinator(t)
			cleaned := 0
			if failure == "guard" {
				c.guard = func(b Binding) error {
					if b.ProviderID == "v2" {
						return errors.New("denied")
					}
					return nil
				}
			}
			lc := lifecycle(func(s *Scope) error {
				if err := s.Track(func() error { cleaned++; return nil }); err != nil {
					return err
				}
				if failure == "prepare" {
					return errors.New("prepare failed")
				}
				if failure == "panic" {
					panic("prepare panic")
				}
				return nil
			}, func(Binding) error {
				if failure == "ready" {
					return errors.New("not ready")
				}
				return nil
			})
			if err := c.Replace(context.Background(), 1, b(2, "v2"), lc); err == nil {
				t.Fatal("expected error")
			}
			if cleaned != 1 {
				t.Fatalf("failed scope leaked or double closed: %d", cleaned)
			}
			lease, err := r.Acquire(allow)
			if err != nil {
				t.Fatal(err)
			}
			if lease.Binding.ProviderID != "v1" {
				t.Fatal("old instance replaced on error")
			}
			lease.Release()
			if c.PendingRetirements() != 0 {
				t.Fatal("failed stage retired old")
			}
		})
	}
}

func TestCoordinatorWithdrawRestoreAndFencing(t *testing.T) {
	c, _, _ := setupCoordinator(t)
	lease, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Acquire(); !errors.Is(err, ErrUnbound) {
		t.Fatalf("new call after withdraw: %v", err)
	}
	if err := c.Restore(context.Background(), 0, b(2, "v2"), lifecycle(func(*Scope) error { return nil }, func(Binding) error { return nil })); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("restored with stale epoch: %v", err)
	}
	if err := c.Restore(context.Background(), 1, b(2, "v2"), lifecycle(func(*Scope) error { return nil }, func(Binding) error { return nil })); err != nil {
		t.Fatal(err)
	}
	newLease, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if newLease.Binding.ProviderID != "v2" || lease.Binding.ProviderID != "v1" {
		t.Fatal("lost pinned generation")
	}
	newLease.Release()
	lease.Release()
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCoordinatorCleanupErrorNotRetried(t *testing.T) {
	c, _, oldScope := setupCoordinator(t)
	count := 0
	if err := oldScope.Track(func() error { count++; return errors.New("cleanup failure") }); err != nil {
		t.Fatal(err)
	}
	lc := lifecycle(func(*Scope) error { return nil }, func(Binding) error { return nil })
	if err := c.Replace(context.Background(), 1, b(2, "v2"), lc); err != nil {
		t.Fatal(err)
	}
	if err := c.Reap(context.Background()); err == nil {
		t.Fatal("cleanup error suppressed")
	}
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if count != 1 || c.PendingRetirements() != 0 {
		t.Fatal("retried cleanup")
	}
}

func TestWithdrawalPreemptsBlockedStage(t *testing.T) {
	c, _, _ := setupCoordinator(t)
	started := make(chan struct{})
	unblock := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- c.Replace(context.Background(), 1, b(2, "v2"), Lifecycle{
			Prepare: func(ctx context.Context, s *Scope) error { close(started); <-unblock; return nil },
			Ready:   func(context.Context, Binding) error { return nil },
		})
	}()
	<-started
	if err := c.Withdraw(1); err != nil {
		t.Fatalf("withdraw must preempt blocked stage: %v", err)
	}
	close(unblock)
	if err := <-result; !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("stale stage published: %v", err)
	}
	if _, err := c.Acquire(); !errors.Is(err, ErrUnbound) {
		t.Fatalf("withdrawal bypassed: %v", err)
	}
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestEmergencyWithdrawalWhileReaperWaits(t *testing.T) {
	c, _, _ := setupCoordinator(t)
	old, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	lc := lifecycle(func(*Scope) error { return nil }, func(Binding) error { return nil })
	if err := c.Replace(context.Background(), 1, b(2, "v2"), lc); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() { close(started); done <- c.Reap(ctx) }()
	<-started
	// Even if Drain is blocked, emergency withdrawal of v2 must work.
	if err := c.Withdraw(2); err != nil {
		t.Fatalf("withdraw blocked by Drain: %v", err)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	old.Release()
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c.PendingRetirements() != 0 {
		t.Fatal("retirements leaked")
	}
}
