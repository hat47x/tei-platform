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

func TestProductCapabilityCanBeInvokedByGoBinding(t *testing.T) {
	loaded, err := workspace.Load(".")
	if err != nil {
		t.Fatalf("product-contracts workspace: %v", err)
	}
	var capability semantic.Capability
	for _, namespace := range loaded.World.Semantics {
		for _, candidate := range namespace.Capabilities {
			if candidate.ID == "product_contracts.eki_task_submit" {
				capability = candidate
			}
		}
	}
	if capability.ID == "" {
		t.Fatal("eki_task_submit capability was not loaded")
	}
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{
			Capability: id,
			Mode:       capabilityinvoke.ResultOutput,
			Payload:    map[string]any{"attemptId": "atm_test"},
		}, nil
	})
	result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, map[string]any{"taskId": "tsk_test"})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if result.Mode != capabilityinvoke.ResultOutput {
		t.Fatalf("result mode = %q, want output", result.Mode)
	}
}
