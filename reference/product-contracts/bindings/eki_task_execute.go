package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

// EKITaskExecute submits a TaskRequest to EKI and polls the result route.
// It is a synchronous wrapper for the T1 EKI Native path.
type EKITaskExecute struct {
	BaseURL string
	Client  *http.Client
	Poll    time.Duration
	Timeout time.Duration
}

func (b EKITaskExecute) Invoke(ctx context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	fields, err := inputObject("eki_task_execute", input)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	request, ok := fields["request"].(map[string]any)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("eki_task_execute requires a request object")
	}
	key, err := requireString("eki_task_execute", fields, "idempotencyKey")
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	submitResult, err := EKITaskSubmitHTTP{BaseURL: b.BaseURL, Client: b.Client}.Invoke(ctx, id, map[string]any{
		"idempotencyKey": key,
		"request":        request,
	})
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("submit: %w", err)
	}
	payload, ok := submitResult.Payload.(map[string]any)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("submit payload is not an object: %T", submitResult.Payload)
	}
	taskID, _ := payload["task_id"].(string)
	if taskID == "" {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("submit response missing task_id")
	}

	client := b.Client
	if client == nil {
		client = http.DefaultClient
	}
	poll := b.Poll
	if poll == 0 {
		poll = 500 * time.Millisecond
	}
	timeout := b.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("eki_task_execute: timeout waiting for result of task %s", taskID)
		}
		select {
		case <-ctx.Done():
			return capabilityinvoke.BusinessResult{}, ctx.Err()
		default:
		}
		url := strings.TrimRight(b.BaseURL, "/") + "/v1/tasks/" + taskID + "/result"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("result request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("result fetch: %w", err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			var result map[string]any
			if err := json.Unmarshal(body, &result); err != nil {
				return capabilityinvoke.BusinessResult{}, fmt.Errorf("decode result: %w", err)
			}
			return capabilityinvoke.BusinessResult{Capability: id, Mode: capabilityinvoke.ResultOutput, Payload: result}, nil
		case http.StatusNoContent, http.StatusAccepted:
			time.Sleep(poll)
			continue
		default:
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("result status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}
}
