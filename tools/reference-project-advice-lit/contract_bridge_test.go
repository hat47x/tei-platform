package main

import (
	"context"
	"testing"

	"tei/internal/composition"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

func TestProjectAdvisoryUsesExistingContractSelection(t *testing.T) {
	bridge, err := newProjectAdvisoryBridge()
	if err != nil {
		t.Fatal(err)
	}
	if bridge.selection.Contract != ProjectAdvisoryContract ||
		bridge.selection.ProviderID != projectAdvisoryProviderID ||
		bridge.selection.Scope != projectAdvisoryScope {
		t.Fatalf("unexpected provider selection: %+v", bridge.selection)
	}
	result, err := bridge.Evaluate(customeradvice.Input{CustomerCode: "VIP-123"})
	if err != nil || !result.Advisory || result.Kind != "review_suggested" {
		t.Fatalf("unexpected advisory: %+v, %v", result, err)
	}
}

func TestProjectAdvisoryFailsClosedOnRemovedProvider(t *testing.T) {
	bridge, err := newProjectAdvisoryBridge()
	if err != nil {
		t.Fatal(err)
	}
	_, err = bridge.registry.Apply(bridge.registry.Snapshot().Revision, nil, []string{projectAdvisoryProviderID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = bridge.Evaluate(customeradvice.Input{CustomerCode: "VIP-123"})
	if err == nil {
		t.Fatal("removed provider must not be invoked")
	}
}

func TestProjectAdvisoryCannotSelectAcrossScope(t *testing.T) {
	bridge, err := newProjectAdvisoryBridge()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.registry.Select(projectAdvisoryProviderID, "other-project"); err == nil {
		t.Fatal("cross-project provider selection must fail")
	}
	_, err = bridge.registry.Apply(bridge.registry.Snapshot().Revision, []composition.Provider{{
		ID: "other-project-provider", Scope: "other-project", Offers: ProjectAdvisoryContract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.registry.Select("other-project-provider", projectAdvisoryScope); err == nil {
		t.Fatal("cross-scope selection must fail")
	}
}

func TestProjectAdvisoryHostWithdrawDeniesEvenEligibleProvider(t *testing.T) {
	bridge, err := newProjectAdvisoryBridge()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.registry.Select(projectAdvisoryProviderID, projectAdvisoryScope); err != nil {
		t.Fatalf("fixture must start with eligible provider: %v", err)
	}
	if err := bridge.coordinator.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.registry.Select(projectAdvisoryProviderID, projectAdvisoryScope); err != nil {
		t.Fatalf("registry eligibility is not revoked with Host withdrawal: %v", err)
	}
	if lease, err := bridge.coordinator.Acquire(); err == nil {
		lease.Release()
		t.Fatal("withdrawn Host must reject new leases")
	}
	if result, err := bridge.Evaluate(customeradvice.Input{CustomerCode: "VIP-123"}); err == nil {
		t.Fatalf("withdrawn Host incorrectly produced advisory: %+v", result)
	}
}

func TestProjectAdvisoryChangedContractDeniesStaleSelection(t *testing.T) {
	bridge, err := newProjectAdvisoryBridge()
	if err != nil {
		t.Fatal(err)
	}
	replacement := ProjectAdvisoryContract
	replacement.Version = "v1"
	_, err = bridge.registry.Apply(bridge.registry.Snapshot().Revision, []composition.Provider{{
		ID:     projectAdvisoryProviderID,
		Scope:  projectAdvisoryScope,
		Offers: replacement,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if lease, err := bridge.coordinator.Acquire(); err == nil {
		lease.Release()
		t.Fatal("contract replacement must invalidate existing selection lease")
	}
	if result, err := bridge.Evaluate(customeradvice.Input{CustomerCode: "VIP-123"}); err == nil {
		t.Fatalf("stale selected contract was invoked: %+v", result)
	}
}

func TestProjectAdvisoryWithdrawReapThenRejects(t *testing.T) {
	bridge, err := newProjectAdvisoryBridge()
	if err != nil {
		t.Fatal(err)
	}
	if err := bridge.coordinator.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := bridge.coordinator.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if bridge.coordinator.PendingRetirements() != 0 {
		t.Fatal("retired reference generation was not drained")
	}
	if _, err := bridge.Evaluate(customeradvice.Input{CustomerCode: "VIP-123"}); err == nil {
		t.Fatal("reaped generation must not be resurrected by eligible registry declaration")
	}
}
