package bindings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEKITaskSubmitHTTPPostsTaskRequest(t *testing.T) {
	var gotKey, gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"taskId":"tsk_reference","phase":"admitted"}`))
	}))
	defer server.Close()

	binding := EKITaskSubmitHTTP{BaseURL: server.URL, Client: server.Client()}
	result, err := binding.Invoke(context.Background(), "product_contracts.eki_task_submit", map[string]any{
		"idempotencyKey": "idem-reference-1",
		"request":        map[string]any{"taskId": "tsk_reference"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/tasks" || gotKey != "idem-reference-1" {
		t.Fatalf("path=%q key=%q", gotPath, gotKey)
	}
	if gotBody["taskId"] != "tsk_reference" {
		t.Fatalf("body=%+v", gotBody)
	}
	if result.Payload.(map[string]any)["taskId"] != "tsk_reference" {
		t.Fatalf("payload=%+v", result.Payload)
	}
}

func TestEKITaskSubmitHTTPRejectsNonAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"title":"not admitted"}`))
	}))
	defer server.Close()

	binding := EKITaskSubmitHTTP{BaseURL: server.URL, Client: server.Client()}
	_, err := binding.Invoke(context.Background(), "product_contracts.eki_task_submit", map[string]any{
		"idempotencyKey": "idem-reference-2",
		"request":        map[string]any{"taskId": "tsk_reference"},
	})
	if err == nil {
		t.Fatal("non-202 response was accepted")
	}
}
