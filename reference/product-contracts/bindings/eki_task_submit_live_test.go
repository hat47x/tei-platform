package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const liveTaskRequest = `{
  "schema": "eki.task-request/v1",
  "executor": {"id": "dev.example.transform", "version_constraint": "^1"},
  "inputs": [],
  "parameters": {"mode": "normalize"},
  "resources": {"cpu_millis": 1000, "memory_bytes": 268435456, "wall_time_seconds": 300, "output_bytes_max": 16777216, "network_bytes_max": 134217728},
  "reliability": {"retryable": true, "preemptible": true, "checkpointable": false, "deterministic": true, "max_attempts": 3, "effect_mode": "none"},
  "priority": "normal"
}`

type bearerTransport struct {
	token string
}

func (b bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(clone)
}

// TestLiveEKITaskSubmitHTTP calls a running EKI Control Plane. It is skipped
// unless EKI_CONTROL_URL and EKI_CONTROL_TOKEN are set, so the default test
// run never depends on a live service.
func TestLiveEKITaskSubmitHTTP(t *testing.T) {
	baseURL := os.Getenv("EKI_CONTROL_URL")
	token := os.Getenv("EKI_CONTROL_TOKEN")
	if baseURL == "" || token == "" {
		t.Skip("EKI_CONTROL_URL and EKI_CONTROL_TOKEN are not set")
	}
	request := map[string]any{}
	if err := json.Unmarshal([]byte(liveTaskRequest), &request); err != nil {
		t.Fatal(err)
	}
	binding := EKITaskSubmitHTTP{
		BaseURL: baseURL,
		Client:  &http.Client{Transport: bearerTransport{token: token}, Timeout: 10 * time.Second},
	}
	result, err := binding.Invoke(context.Background(), "product_contracts.eki_task_submit", map[string]any{
		"idempotencyKey": fmt.Sprintf("tai-live-%d", time.Now().UnixNano()),
		"request":        request,
	})
	if err != nil {
		t.Fatalf("live submit: %v", err)
	}
	payload, ok := result.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload is not an object: %T", result.Payload)
	}
	taskID, _ := payload["task_id"].(string)
	if !strings.HasPrefix(taskID, "tsk_") {
		t.Fatalf("task_id = %q", taskID)
	}
	t.Logf("live task_id=%s", taskID)
}
