package bindings

import (
	"context"
	"testing"
)

func TestEKITaskSubmitReturnsAttempt(t *testing.T) {
	result, err := EKITaskSubmit{}.Invoke(context.Background(), "product_contracts.eki_task_submit", map[string]any{
		"taskId": "tsk_reference", "executorId": "dev.example.transform",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Payload.(map[string]any)["phase"] != "dispatched" {
		t.Fatalf("payload = %+v", result.Payload)
	}
}

func TestEKITaskSubmitRejectsMissingInput(t *testing.T) {
	if _, err := (EKITaskSubmit{}).Invoke(context.Background(), "product_contracts.eki_task_submit", map[string]any{"taskId": "tsk_reference"}); err == nil {
		t.Fatal("missing executorId was accepted")
	}
}
