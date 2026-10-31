package productcontracts_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
	"tei/reference/product-contracts/bindings"
)

type adapterFunc func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)

func (f adapterFunc) Invoke(ctx context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, id, input)
}

func TestProductCapabilitiesCanBeInvokedByGoBinding(t *testing.T) {
	loaded, err := workspace.Load(".")
	if err != nil {
		t.Fatalf("product-contracts workspace: %v", err)
	}
	byID := map[core.CanonicalID]semantic.Capability{}
	for _, namespace := range loaded.World.Semantics {
		for _, capability := range namespace.Capabilities {
			byID[capability.ID] = capability
		}
	}
	ids := []core.CanonicalID{
		"product_contracts.eki_task_submit",
		"product_contracts.eki_attempt_execute",
		"product_contracts.eki_result_accept",
		"product_contracts.sui_document_put",
		"product_contracts.sei_decision_record",
		"product_contracts.mmi_observation_select",
	}
	adapters := map[core.CanonicalID]capabilityinvoke.Adapter{
		"product_contracts.eki_task_submit":        bindings.EKITaskSubmit{},
		"product_contracts.eki_attempt_execute":    bindings.EKIAttemptExecute{},
		"product_contracts.eki_result_accept":      bindings.EKIResultAccept{},
		"product_contracts.sui_document_put":       bindings.SUIDocumentPut{},
		"product_contracts.sei_decision_record":    bindings.SEIDecisionRecord{},
		"product_contracts.mmi_observation_select": bindings.MMIObservationSelect{},
	}
	adapter := adapterFunc(func(ctx context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
		bound, ok := adapters[id]
		if !ok {
			t.Fatalf("no reference binding for %s", id)
		}
		return bound.Invoke(ctx, id, input)
	})
	for _, id := range ids {
		capability, ok := byID[id]
		if !ok {
			t.Fatalf("%s capability was not loaded", id)
		}
		inputs := map[core.CanonicalID]map[string]any{
			"product_contracts.eki_task_submit":        {"taskId": "tsk_reference", "executorId": "dev.example.transform"},
			"product_contracts.eki_attempt_execute":    {"attemptId": "atm_reference"},
			"product_contracts.eki_result_accept":      {"claimId": "clm_reference"},
			"product_contracts.sui_document_put":       {"documentId": "doc_reference"},
			"product_contracts.sei_decision_record":    {"decisionId": "dec_reference"},
			"product_contracts.mmi_observation_select": {"table": "Sheet1!A1:E6"},
		}
		result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, inputs[id])
		if err != nil {
			t.Fatalf("invoke %s: %v", id, err)
		}
		if result.Mode != capabilityinvoke.ResultOutput {
			t.Fatalf("%s result mode = %q, want output", id, result.Mode)
		}
	}
}
