package composition

import (
	"context"
	"errors"
	"testing"
)

// This is an explicit host reconciliation reference, not automatic runtime activation.
func TestContractCoeffectLossWithdrawsLiveService(t *testing.T) {
	registry := NewRegistry()
	plan, err := registry.Apply(0, []Provider{
		provider("data-v1", "tenant-A", storage),
		provider("list", "tenant-A", app, required(storage)),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	consumer, ok := plan.Assessment("list")
	if !ok || !consumer.Eligible {
		t.Fatalf("consumer unavailable: %+v", consumer)
	}
	scope := NewScope()
	cleanup := 0
	if err := scope.Track(func() error { cleanup++; return nil }); err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(b(1, "data-v1"), scope)
	if err != nil {
		t.Fatal(err)
	}
	inFlight, err := router.Acquire(allow)
	if err != nil {
		t.Fatal(err)
	}
	// A dropped coeffect is a declarative loss, and the host must withdraw.
	plan, err = registry.Apply(plan.Revision, nil, []string{"data-v1"})
	if err != nil {
		t.Fatal(err)
	}
	consumer, _ = plan.Assessment("list")
	if consumer.Eligible {
		t.Fatal("consumer remained eligible without provider")
	}
	retired, err := router.Withdraw(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := router.Acquire(allow); !errors.Is(err, ErrUnbound) {
		t.Fatalf("new call reached retired provider: %v", err)
	}
	if cleanup != 0 {
		t.Fatal("disposed active invocation")
	}
	inFlight.Release()
	if err := retired.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cleanup != 1 {
		t.Fatalf("expected one cleanup, got %d", cleanup)
	}
	// Restoring the provider makes the consumer eligible, but does not silently
	// reactivate an instance or grant execution authority.
	plan, err = registry.Apply(plan.Revision, []Provider{provider("data-v2", "tenant-A", storage)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	consumer, _ = plan.Assessment("list")
	if !consumer.Eligible || consumer.Bindings[0].ProviderID != "data-v2" {
		t.Fatalf("not eligible after provider return: %+v", consumer)
	}
	if _, err := router.Acquire(allow); !errors.Is(err, ErrUnbound) {
		t.Fatal("registry state reactivated runtime without host")
	}
}
