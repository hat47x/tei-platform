package composition

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// This deliberately joins the declarative registry, dependency impact plan and
// a host-driven cutover; the production eligibility/authority handshake remains
// separate and is NOT implied by an eligible Assessment.
func TestRegistryPlanStagesAndPublishesGeneration(t *testing.T) {
	registry := NewRegistry()
	before, err := registry.Apply(0, []Provider{
		provider("db-v1", "tenant-A", storage),
		provider("consumer", "tenant-A", app, required(storage)),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := registry.Apply(before.Revision, []Provider{provider("db-v2", "tenant-A", storage)}, []string{"db-v1"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanReconciliation(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.StartOrder, []string{"db-v2", "consumer"}) || !reflect.DeepEqual(plan.StopOrder, []string{"consumer", "db-v1"}) {
		t.Fatalf("unexpected dependency plan: %+v", plan)
	}
	oldBinding := Binding{ContractID: storage.ID, ContractVersion: storage.Version, ProviderID: "db-v1", ArtifactDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1}
	router, err := NewRouter(oldBinding, NewScope())
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewCoordinator(router, allow)
	if err != nil {
		t.Fatal(err)
	}
	old, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	prepared, ready := false, false
	newBinding := oldBinding
	newBinding.ProviderID, newBinding.Generation = "db-v2", 2
	hooks := Lifecycle{
		Prepare: func(_ context.Context, scope *Scope) error {
			prepared = true
			return scope.Track(func() error { return nil })
		},
		Ready: func(_ context.Context, b Binding) error {
			a, ok := after.Assessment(b.ProviderID)
			if !ok || !a.Eligible {
				return errors.New("not contract-eligible")
			}
			ready = true
			return nil
		},
	}
	if err := c.Replace(context.Background(), 1, newBinding, hooks); err != nil {
		t.Fatal(err)
	}
	if !prepared || !ready {
		t.Fatal("provider not prepared/ready")
	}
	newCall, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if old.Binding.ProviderID != "db-v1" || newCall.Binding.ProviderID != "db-v2" {
		t.Fatal("generation boundary broken")
	}
	newCall.Release()
	old.Release()
	if err := c.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}
