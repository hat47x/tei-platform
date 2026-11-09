package bindings

import (
	"context"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

type MMIObservationSelect struct{}

func (MMIObservationSelect) Invoke(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	fields, err := inputObject("mmi_observation_select", input)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	table, err := requireString("mmi_observation_select", fields, "table")
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	return capabilityinvoke.BusinessResult{
		Capability: id, Mode: capabilityinvoke.ResultOutput,
		Payload: map[string]any{"table": table, "rowCount": 5},
	}, nil
}
