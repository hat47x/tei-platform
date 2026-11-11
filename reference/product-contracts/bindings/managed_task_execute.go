package bindings

import (
	"context"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

// ManagedTaskExecute is a stub for T2 Managed Provider execution.
// It records that T2 is not yet implemented and returns a synthetic
// result for comparison purposes.
type ManagedTaskExecute struct{}

func (ManagedTaskExecute) Invoke(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	return capabilityinvoke.BusinessResult{
		Capability: id,
		Mode:       capabilityinvoke.ResultOutput,
		Payload: map[string]any{
			"path":      "T2 Managed Provider",
			"status":    "not_implemented",
			"note":      "T2 Managed Provider delegation is not yet implemented",
			"requestId": "managed-reference",
		},
	}, nil
}
