package bindings

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveEKITaskExecuteHTTP runs the T1 EKI Native path against a live
// eki-control and submits a task. It requires TEI_EKI_LIVE_WRITE=1 as well as
// EKI_CONTROL_URL and EKI_CONTROL_TOKEN.
func TestLiveEKITaskExecuteHTTP(t *testing.T) {
	if os.Getenv("TEI_EKI_LIVE_WRITE") != "1" {
		t.Skip("live submission requires TEI_EKI_LIVE_WRITE=1")
	}
	baseURL := os.Getenv("EKI_CONTROL_URL")
	token := os.Getenv("EKI_CONTROL_TOKEN")
	if baseURL == "" || token == "" {
		t.Skip("EKI_CONTROL_URL and EKI_CONTROL_TOKEN are not set")
	}
	request := map[string]any{
		"schema": "eki.task-request/v1",
		"executor": map[string]any{
			"id":                 "tei-pure-compute",
			"version_constraint": "^1",
		},
		"inputs": []map[string]any{
			{"name": "input.txt", "artifact_id": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		},
		"parameters": map[string]any{
			"tei": map[string]any{
				"operation":         "pure-compute-v1",
				"source_revision":   "test-rev",
				"executable_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				"toolchain":         "go1.25.0",
				"command":           "test-bin",
			},
		},
		"resources": map[string]any{
			"cpu_millis":        1000,
			"memory_bytes":      1073741824,
			"wall_time_seconds": 60,
			"output_bytes_max":  1048576,
			"network_bytes_max": 0,
		},
		"reliability": map[string]any{
			"retryable":      true,
			"preemptible":    true,
			"checkpointable": false,
			"deterministic":  true,
			"max_attempts":   2,
			"effect_mode":    "none",
		},
		"deadline":       time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339),
		"priority":       "normal",
		"result_profile": "default",
	}
	binding := EKITaskExecute{
		BaseURL: baseURL,
		Client:  &http.Client{Transport: bearerTransport{token: token}, Timeout: 30 * time.Second},
		Timeout: 10 * time.Second,
	}
	result, err := binding.Invoke(context.Background(), "product_contracts.eki_task_execute", map[string]any{
		"idempotencyKey": fmt.Sprintf("tei-live-%d", time.Now().UnixNano()),
		"request":        request,
	})
	if err != nil {
		t.Fatalf("live execute: %v", err)
	}
	payload, ok := result.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload is not an object: %T", result.Payload)
	}
	t.Logf("live execute payload: %+v", payload)
}

// TestLiveEKITaskExecuteInsufficientAuthority sends a request that should be
// rejected by admission. It is skipped unless EKI_CONTROL_URL and
// EKI_CONTROL_TOKEN are set.
func TestLiveEKITaskExecuteInsufficientAuthority(t *testing.T) {
	if os.Getenv("TEI_EKI_LIVE_WRITE") != "1" {
		t.Skip("live submission requires TEI_EKI_LIVE_WRITE=1")
	}
	baseURL := os.Getenv("EKI_CONTROL_URL")
	token := os.Getenv("EKI_CONTROL_TOKEN")
	if baseURL == "" || token == "" {
		t.Skip("EKI_CONTROL_URL and EKI_CONTROL_TOKEN are not set")
	}
	request := map[string]any{
		"schema": "eki.task-request/v1",
		"executor": map[string]any{
			"id":                 "unknown-executor",
			"version_constraint": "^1",
		},
		"inputs": []map[string]any{},
		"parameters": map[string]any{
			"tei": map[string]any{
				"operation": "unknown",
			},
		},
		"resources": map[string]any{
			"cpu_millis":        1000000,
			"memory_bytes":      1073741824,
			"wall_time_seconds": 60,
			"output_bytes_max":  1048576,
			"network_bytes_max": 0,
		},
		"reliability": map[string]any{
			"retryable":      true,
			"preemptible":    true,
			"checkpointable": false,
			"deterministic":  true,
			"max_attempts":   2,
			"effect_mode":    "none",
		},
		"deadline":       time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339),
		"priority":       "normal",
		"result_profile": "default",
	}
	binding := EKITaskExecute{
		BaseURL: baseURL,
		Client:  &http.Client{Transport: bearerTransport{token: token}, Timeout: 10 * time.Second},
		Timeout: 10 * time.Second,
	}
	_, err := binding.Invoke(context.Background(), "product_contracts.eki_task_execute", map[string]any{
		"idempotencyKey": fmt.Sprintf("tei-insufficient-%d", time.Now().UnixNano()),
		"request":        request,
	})
	if err == nil {
		t.Fatal("expected error for insufficient authority, got nil")
	}
	if !strings.Contains(err.Error(), "FORBIDDEN") && !strings.Contains(err.Error(), "UNAUTHORIZED") {
		t.Logf("insufficient authority error: %v", err)
	}
}
