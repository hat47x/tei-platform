package bindings

import (
	"context"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

type SEIDecisionRecord struct{}

func (SEIDecisionRecord) Invoke(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	fields, err := inputObject("sei_decision_record", input)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	decisionID, err := requireString("sei_decision_record", fields, "decisionId")
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	return capabilityinvoke.BusinessResult{
		Capability: id, Mode: capabilityinvoke.ResultOutput,
		Payload: map[string]any{"decisionId": decisionID, "provisional": true},
	}, nil
}
