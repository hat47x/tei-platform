// Package bindings holds deterministic reference bindings for the product
// capabilities. They validate the capability input shape and return a synthetic
// output; they do not contact EKI or create a real Task.
package bindings

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

type EKITaskSubmit struct{}

func (EKITaskSubmit) Invoke(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	fields, ok := input.(map[string]any)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("eki_task_submit input must be an object")
	}
	taskID, _ := fields["taskId"].(string)
	executorID, _ := fields["executorId"].(string)
	if taskID == "" || executorID == "" {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("eki_task_submit requires taskId and executorId")
	}
	return capabilityinvoke.BusinessResult{
		Capability: id,
		Mode:       capabilityinvoke.ResultOutput,
		Payload: map[string]any{
			"attemptId": "atm_reference",
			"taskId":    taskID,
			"executor":  executorID,
			"phase":     "dispatched",
		},
	}, nil
}
