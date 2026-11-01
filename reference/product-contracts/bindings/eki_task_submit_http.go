package bindings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

// EKITaskSubmitHTTP submits a TaskRequest to an EKI Control Plane over HTTP.
// The caller supplies the request body and idempotency key, because the TaskIR
// shape is owned by EKI, not by this binding. Transport authentication (mTLS or
// a bearer token) is supplied through Client.
type EKITaskSubmitHTTP struct {
	BaseURL string
	Client  *http.Client
}

func (b EKITaskSubmitHTTP) Invoke(ctx context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	fields, err := inputObject("eki_task_submit_http", input)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	request, ok := fields["request"].(map[string]any)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("eki_task_submit_http requires a request object")
	}
	key, err := requireString("eki_task_submit_http", fields, "idempotencyKey")
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("encode task request: %w", err)
	}
	url := strings.TrimRight(b.BaseURL, "/") + "/v1/tasks"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Idempotency-Key", key)
	client := b.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("submit task: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("submit task: status %d: %s", response.StatusCode, strings.TrimSpace(string(detail)))
	}
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("decode task acceptance: %w", err)
	}
	return capabilityinvoke.BusinessResult{Capability: id, Mode: capabilityinvoke.ResultOutput, Payload: payload}, nil
}
