package bindings

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestEKITaskExecuteSynthetic(t *testing.T) {
	binding := EKITaskExecute{
		BaseURL: "http://127.0.0.1:1",
		Client:  &http.Client{Timeout: 100 * time.Millisecond},
		Timeout: 200 * time.Millisecond,
	}
	_, err := binding.Invoke(context.Background(), "product_contracts.eki_task_execute", map[string]any{
		"idempotencyKey": "test-key",
		"request": map[string]any{
			"schema": "eki.task-request/v1",
		},
	})
	if err == nil {
		t.Fatal("expected error for unreachable eki-control, got nil")
	}
}
