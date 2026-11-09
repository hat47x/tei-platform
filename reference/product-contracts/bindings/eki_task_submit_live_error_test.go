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

// TestLiveEKIProblemJSON checks the error path against a running EKI Control
// Plane. It is skipped unless the live environment variables are set.
func TestLiveEKIProblemJSON(t *testing.T) {
	baseURL := os.Getenv("EKI_CONTROL_URL")
	token := os.Getenv("EKI_CONTROL_TOKEN")
	if baseURL == "" || token == "" {
		t.Skip("EKI_CONTROL_URL and EKI_CONTROL_TOKEN are not set")
	}
	binding := EKITaskSubmitHTTP{
		BaseURL: baseURL,
		Client:  &http.Client{Transport: bearerTransport{token: token}, Timeout: 10 * time.Second},
	}
	_, err := binding.Invoke(context.Background(), "product_contracts.eki_task_submit", map[string]any{
		"idempotencyKey": fmt.Sprintf("tai-live-error-%d", time.Now().UnixNano()),
		"request":        map[string]any{},
	})
	if err == nil {
		t.Fatal("an empty TaskRequest was accepted")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("error does not carry status 400: %v", err)
	}
	t.Logf("live problem: %v", err)
}
