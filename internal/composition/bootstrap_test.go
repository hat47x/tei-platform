package composition

import (
	"context"
	"errors"
	"testing"
)

func TestSelectedBootstrapFailClosed(t *testing.T) {
	registry, selected, _ := selectedFixture(t)
	initial := selectedBinding(1, "v2")
	deny := errors.New("no authority")
	if _, err := NewSelectedCoordinator(registry, selected, initial, NewScope(), func(Binding) error { return deny }); !errors.Is(err, deny) {
		t.Fatalf("revoked authority permitted bootstrap: %v", err)
	}
	bad := initial
	bad.ProviderID = "different"
	if _, err := NewSelectedCoordinator(registry, selected, bad, NewScope(), allow); !errors.Is(err, ErrSelectionMismatch) {
		t.Fatalf("incorrect provider permitted bootstrap: %v", err)
	}
	if _, err := NewSelectedCoordinator(nil, selected, initial, NewScope(), allow); !errors.Is(err, ErrCoordinatorInvalid) {
		t.Fatalf("missing registry: %v", err)
	}
	c, err := NewSelectedCoordinator(registry, selected, initial, NewScope(), allow)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	lease.Release()
	if _, err := registry.Apply(selected.Revision, nil, []string{"v2"}); err != nil {
		t.Fatal(err)
	}
	if lease, err := c.Acquire(); !errors.Is(err, ErrStaleRegistry) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("removed provider accepted: %v", err)
	}
	if err := c.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSelectedBootstrapRejectsStaleSelection(t *testing.T) {
	r, selected, _ := selectedFixture(t)
	if _, err := r.Apply(selected.Revision, nil, []string{"v2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewSelectedCoordinator(r, selected, selectedBinding(1, "v2"), NewScope(), allow); !errors.Is(err, ErrStaleRegistry) {
		t.Fatalf("stale bootstrap accepted: %v", err)
	}
}
