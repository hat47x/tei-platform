package composition

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// This reference is deliberately synthetic. Two Application scopes consume
// one logical service contract, but do not share a Provider selection, runtime
// authority, or an accepted business relation. No shared Plugin is inferred.
func TestTwoApplicationScopesReuseContractWithoutBorrowingProvider(t *testing.T) {
	const (
		orderScope  = "order_entry"
		repairScope = "repair_intake"
	)
	customer := Contract{ID: "master_data.customer.find_by_id", Version: "2.0.0", Shape: "customer-id-query-v2"}
	orderView := Contract{ID: "order_entry.customer_reference", Version: "1.0.0", Shape: "order-view-v1"}
	repairView := Contract{ID: "repair_intake.customer_reference", Version: "1.0.0", Shape: "repair-view-v1"}

	registry := NewRegistry()
	snapshot, err := registry.Apply(0, []Provider{
		{ID: "order.customer", Scope: orderScope, Offers: customer},
		{ID: "repair.customer", Scope: repairScope, Offers: customer},
		{ID: "order.reference", Scope: orderScope, Offers: orderView, Requires: []Dependency{{Contract: customer}}},
		{ID: "repair.reference", Scope: repairScope, Offers: repairView, Requires: []Dependency{{Contract: customer}}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	orderSelection, err := registry.Select("order.reference", orderScope)
	if err != nil {
		t.Fatal(err)
	}
	repairSelection, err := registry.Select("repair.reference", repairScope)
	if err != nil {
		t.Fatal(err)
	}
	assertDependency := func(selected Selection, want string) {
		t.Helper()
		if len(selected.Dependencies) != 1 ||
			selected.Dependencies[0].ContractID != customer.ID ||
			selected.Dependencies[0].ProviderID != want {
			t.Fatalf("%s: wrong scoped dependency: %+v", selected.ProviderID, selected.Dependencies)
		}
	}
	assertDependency(orderSelection, "order.customer")
	assertDependency(repairSelection, "repair.customer")

	for _, tc := range []struct{ provider, wrongScope string }{
		{"order.reference", repairScope},
		{"repair.reference", orderScope},
	} {
		if _, err := registry.Select(tc.provider, tc.wrongScope); !errors.Is(err, ErrSelectionMismatch) {
			t.Fatalf("cross-Application provider selection %s: %v", tc.provider, err)
		}
	}

	// The permissive guard only lets this synthetic reference exercise the
	// composition mechanics. It is NOT evidence of actual Host authority.
	testGuard := func(Binding) error { return nil }
	bind := func(c Contract, providerID, artifactByte string) Binding {
		return Binding{
			ContractID: c.ID, ContractVersion: c.Version, ProviderID: providerID,
			ArtifactDigest: "sha256:" + strings.Repeat(artifactByte, 64), Generation: 1,
		}
	}
	orderCoordinator, err := NewSelectedCoordinator(
		registry, orderSelection, bind(orderView, "order.reference", "a"), NewScope(), testGuard,
	)
	if err != nil {
		t.Fatal(err)
	}
	repairCoordinator, err := NewSelectedCoordinator(
		registry, repairSelection, bind(repairView, "repair.reference", "b"), NewScope(), testGuard,
	)
	if err != nil {
		t.Fatal(err)
	}

	inFlightOrder, err := orderCoordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	inFlightRepair, err := repairCoordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}

	// Changing only the order-side data contract makes that Application's
	// dependency ineligible; the other Application must not silently inherit
	// the first scope's provider, lose its selection, or change its binding.
	changed := customer
	changed.Shape = "incompatible-customer-query-shape"
	updated, err := registry.Apply(snapshot.Revision, []Provider{
		{ID: "order.customer", Scope: orderScope, Offers: changed},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	orderAssessment, _ := updated.Assessment("order.reference")
	repairAssessment, _ := updated.Assessment("repair.reference")
	if orderAssessment.Eligible || !repairAssessment.Eligible {
		t.Fatalf("cross-Application dependency state leaked: order=%+v repair=%+v", orderAssessment, repairAssessment)
	}
	if lease, err := orderCoordinator.Acquire(); !errors.Is(err, ErrStaleRegistry) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("stale order Application served a new call: %v", err)
	}
	unaffected, err := repairCoordinator.Acquire()
	if err != nil {
		t.Fatalf("unrelated Application lost valid selection: %v", err)
	}
	if unaffected.Binding.ProviderID != "repair.reference" ||
		inFlightOrder.Binding.ProviderID != "order.reference" ||
		inFlightRepair.Binding.ProviderID != "repair.reference" {
		t.Fatal("already acquired calls or unaffected Application changed Provider")
	}
	unaffected.Release()
	inFlightOrder.Release()
	inFlightRepair.Release()
	for _, c := range []*Coordinator{orderCoordinator, repairCoordinator} {
		if err := c.Withdraw(1); err != nil {
			t.Fatal(err)
		}
		if err := c.Reap(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}
