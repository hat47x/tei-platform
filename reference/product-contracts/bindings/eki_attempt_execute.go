package bindings

import (
	"context"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

type EKIAttemptExecute struct{}

func (EKIAttemptExecute) Invoke(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	fields, err := inputObject("eki_attempt_execute", input)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	attemptID, err := requireString("eki_attempt_execute", fields, "attemptId")
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	return capabilityinvoke.BusinessResult{
		Capability: id, Mode: capabilityinvoke.ResultOutput,
		Payload: map[string]any{"claimId": "clm_reference", "attemptId": attemptID, "status": "completed"},
	}, nil
}
