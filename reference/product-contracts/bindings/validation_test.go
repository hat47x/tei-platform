package bindings

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

func TestAllBindingsRejectMissingRequiredInput(t *testing.T) {
	cases := []struct {
		id      core.CanonicalID
		adapter capabilityinvoke.Adapter
	}{
		{"product_contracts.eki_task_submit", EKITaskSubmit{}},
		{"product_contracts.eki_attempt_execute", EKIAttemptExecute{}},
		{"product_contracts.eki_result_accept", EKIResultAccept{}},
		{"product_contracts.sui_document_put", SUIDocumentPut{}},
		{"product_contracts.sei_decision_record", SEIDecisionRecord{}},
		{"product_contracts.mmi_observation_select", MMIObservationSelect{}},
	}
	for _, tc := range cases {
		if _, err := tc.adapter.Invoke(context.Background(), tc.id, map[string]any{}); err == nil {
			t.Errorf("%s accepted empty input", tc.id)
		}
	}
}
