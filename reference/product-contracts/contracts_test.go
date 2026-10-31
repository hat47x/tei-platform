package productcontracts_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
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
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{
			Capability: id,
			Mode:       capabilityinvoke.ResultOutput,
			Payload:    map[string]any{"binding": "test"},
		}, nil
	})
	for _, id := range ids {
		capability, ok := byID[id]
		if !ok {
			t.Fatalf("%s capability was not loaded", id)
		}
		result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, map[string]any{"input": "test"})
		if err != nil {
			t.Fatalf("invoke %s: %v", id, err)
		}
		if result.Mode != capabilityinvoke.ResultOutput {
			t.Fatalf("%s result mode = %q, want output", id, result.Mode)
		}
	}
}
