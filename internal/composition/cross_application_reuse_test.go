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

	// A shared artifact content digest does not confer a shared Application
	// permission or prove that both hosts use the same running instance.
	sharedArtifactDigest := "sha256:" + strings.Repeat("a", 64)
	bind := func(c Contract, providerID string) Binding {
		return Binding{
			ContractID: c.ID, ContractVersion: c.Version, ProviderID: providerID,
			ArtifactDigest: sharedArtifactDigest, Generation: 1,
		}
	}
	allowedByApplication := map[string]bool{orderScope: true, repairScope: true}
	artifactTrusted := true
	errApplicationDenied := errors.New("Application access revoked")
	errArtifactUntrusted := errors.New("shared artifact trust revoked")
	// These are explicitly synthetic Host-owned guards; no Production
	// permission, attestation or invocation context is modeled here.
	guardFor := func(scope string) func(Binding) error {
		return func(b Binding) error {
			if !artifactTrusted || b.ArtifactDigest != sharedArtifactDigest {
				return errArtifactUntrusted
			}
			if !allowedByApplication[scope] {
				return errApplicationDenied
			}
			return nil
		}
	}
	orderCoordinator, err := NewSelectedCoordinator(
		registry, orderSelection, bind(orderView, "order.reference"), NewScope(), guardFor(orderScope),
	)
	if err != nil {
		t.Fatal(err)
	}
	repairCoordinator, err := NewSelectedCoordinator(
		registry, repairSelection, bind(repairView, "repair.reference"), NewScope(), guardFor(repairScope),
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
	if inFlightOrder.Binding.ArtifactDigest != inFlightRepair.Binding.ArtifactDigest ||
		inFlightOrder.Binding.ProviderID == inFlightRepair.Binding.ProviderID {
		t.Fatal("physical artifact identity and logical Provider identity were conflated")
	}

	// A single Application loses access without revoking the shared artifact
	// or interrupting already-acquired calls from either Application.
	allowedByApplication[orderScope] = false
	if lease, err := orderCoordinator.Acquire(); !errors.Is(err, errApplicationDenied) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("order access was not denied independently: %v", err)
	}
	validRepair, err := repairCoordinator.Acquire()
	if err != nil {
		t.Fatalf("unrelated Application lost access to shared artifact: %v", err)
	}
	validRepair.Release()
	allowedByApplication[orderScope] = true

	// Revoking the shared artifact denies NEW calls for both Applications,
	// while the previously acquired leases retain their original bindings.
	artifactTrusted = false
	for _, c := range []*Coordinator{orderCoordinator, repairCoordinator} {
		if lease, err := c.Acquire(); !errors.Is(err, errArtifactUntrusted) {
			if lease != nil {
				lease.Release()
			}
			t.Fatalf("shared artifact revocation was not enforced: %v", err)
		}
	}
	artifactTrusted = true

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
