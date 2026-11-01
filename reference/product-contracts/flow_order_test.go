package productcontracts_test

import (
	"reflect"
	"testing"

	"tei/internal/composition"
)

func productContract(id string) composition.Contract {
	return composition.Contract{ID: id, Version: "1.0.0", Shape: "product-view-v1"}
}

// TestProductFlowActivationFollowsDependencies uses the composition registry,
// not a product-specific keyword, to derive the order of the three EKI steps.
// The order comes from the Attempt and ResultClaim coeffects, so a missing or
// reordered dependency changes ActivationOrder.
func TestProductFlowActivationFollowsDependencies(t *testing.T) {
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{
		{
			ID:     "eki_task_submit",
			Scope:  "product_contracts",
			Offers: productContract("product_contracts.attempt"),
		},
		{
			ID:     "eki_attempt_execute",
			Scope:  "product_contracts",
			Offers: productContract("product_contracts.result_claim"),
			Requires: []composition.Dependency{
				{Contract: productContract("product_contracts.attempt")},
			},
		},
		{
			ID:     "eki_result_accept",
			Scope:  "product_contracts",
			Offers: productContract("product_contracts.accepted_result"),
			Requires: []composition.Dependency{
				{Contract: productContract("product_contracts.result_claim")},
			},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"eki_task_submit", "eki_attempt_execute", "eki_result_accept"}
	if !reflect.DeepEqual(snapshot.ActivationOrder, want) {
		t.Fatalf("activation order = %v, want %v", snapshot.ActivationOrder, want)
	}
	for _, id := range want {
		assessment, ok := snapshot.Assessment(id)
		if !ok || !assessment.Eligible {
			t.Fatalf("%s is not eligible: %+v", id, assessment)
		}
	}
}

// TestProductFlowOrderDependsOnCoeffects removes the Attempt/ResultClaim
// dependencies. The registry then falls back to its deterministic name order,
// which differs from the EKI chain, so the earlier assertion is not vacuous.
func TestProductFlowOrderDependsOnCoeffects(t *testing.T) {
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{
		{ID: "eki_task_submit", Scope: "product_contracts", Offers: productContract("product_contracts.attempt")},
		{ID: "eki_attempt_execute", Scope: "product_contracts", Offers: productContract("product_contracts.result_claim")},
		{ID: "eki_result_accept", Scope: "product_contracts", Offers: productContract("product_contracts.accepted_result")},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"eki_attempt_execute", "eki_result_accept", "eki_task_submit"}
	if !reflect.DeepEqual(snapshot.ActivationOrder, want) {
		t.Fatalf("activation order without coeffects = %v, want %v", snapshot.ActivationOrder, want)
	}
}
