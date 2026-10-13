package main

import (
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
