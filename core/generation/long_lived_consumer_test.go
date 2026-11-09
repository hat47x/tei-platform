package generation

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"tei/core/effect"
)

// This models a long-lived consumer owned by the test, not a product
// Subscription implementation. The lease must outlive Start and cover all
// deliveries until the consumer explicitly stops.
func TestLongLivedConsumerKeepsRetiringGenerationUntilStop(t *testing.T) {
	binding := func(number uint64, provider string) Binding {
		return Binding{
			ContractID:      "std.test.long-lived",
			ContractVersion: "1.0.0",
			ProviderID:      provider,
			ArtifactDigest:  "sha256:" + strings.Repeat("a", 64),
			Generation:      number,
		}
	}
	allow := func(Binding) error { return nil } // Test guard, not product authority.
	oldScope := effect.NewScope()
	cleanupCalls := 0
	if err := oldScope.Track(func() error {
		cleanupCalls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(binding(1, "v1"), oldScope)
	if err != nil {
		t.Fatal(err)
	}
	oldLease, err := router.Acquire(allow)
	if err != nil {
		t.Fatal(err)
	}

	type delivery struct {
		generation uint64
		value      string
	}
	input := make(chan string)
	output := make(chan delivery, 1)
	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer oldLease.Release()
		for {
			select {
			case <-stopCh:
				return
			case value := <-input:
				select {
				case output <- delivery{generation: oldLease.Binding.Generation, value: value}:
				case <-stopCh:
					return
				}
			}
		}
	}()
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() { close(stopCh) })
		<-done
	}
	t.Cleanup(stop)

	limit, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	expectDelivery := func(value string) {
		t.Helper()
		select {
		case input <- value:
		case <-limit.Done():
			t.Fatalf("subscription did not accept %q: %v", value, limit.Err())
		}
		select {
		case got := <-output:
			if got.generation != 1 || got.value != value {
				t.Fatalf("delivery after cutover changed provider: %+v", got)
			}
		case <-limit.Done():
			t.Fatalf("subscription did not deliver %q: %v", value, limit.Err())
		}
	}

	expectDelivery("before-cutover")
	nextScope := effect.NewScope()
	retired, err := router.Publish(1, binding(2, "v2"), nextScope, allow)
	if err != nil {
		t.Fatal(err)
	}
	newLease, err := router.Acquire(allow)
	if err != nil {
		t.Fatal(err)
	}
	if newLease.Binding.Generation != 2 || newLease.Binding.ProviderID != "v2" {
		t.Fatalf("new consumer did not receive v2: %+v", newLease.Binding)
	}
	newLease.Release()

	// Cutover alone neither stops the old consumer nor disposes its resources.
	expectDelivery("after-cutover")
	drainCtx, cancelDrain := context.WithCancel(context.Background())
	cancelDrain()
	if err := retired.Drain(drainCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("active consumer must prevent drain, got %v", err)
	}
	if cleanupCalls != 0 {
		t.Fatalf("retiring scope released during active deliveries: %d", cleanupCalls)
	}

	// The consumer, not the Router, owns its termination. Once it stops and
	// releases its lease, the Host can finish retirement and transient cleanup.
	stop()
	if err := retired.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cleanupCalls != 1 {
		t.Fatalf("old generation cleanup count=%d, want 1", cleanupCalls)
	}
	if err := retired.Drain(context.Background()); err != nil {
		t.Fatalf("repeated drain changed the cleanup result: %v", err)
	}
	if cleanupCalls != 1 {
		t.Fatalf("old generation cleanup was repeated: %d", cleanupCalls)
	}
	if err := nextScope.Close(); err != nil {
		t.Fatal(err)
	}
}
