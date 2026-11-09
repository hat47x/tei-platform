package bindings

import (
	"context"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

type EKIResultAccept struct{}

func (EKIResultAccept) Invoke(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	fields, err := inputObject("eki_result_accept", input)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	claimID, err := requireString("eki_result_accept", fields, "claimId")
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	return capabilityinvoke.BusinessResult{
		Capability: id, Mode: capabilityinvoke.ResultOutput,
		Payload: map[string]any{"resultId": "res_reference", "claimId": claimID, "outcome": "accepted"},
	}, nil
}
