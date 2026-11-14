package bindings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"tei/internal/core"
	"tei/internal/scenariotest"
	"tei/internal/semantic/capabilityinvoke"
)

var ekiTaskIDPattern = regexp.MustCompile(`^tsk_[0-9a-fA-F]{32}$`)
var ekiDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

const ekiResultLimit = 1 << 20

// EKIResultError retains the accepted task ID when result retrieval fails.
// The caller can reconcile that task rather than submitting a second one.
type EKIResultError struct {
	TaskID string
	Err    error
}

func (e *EKIResultError) Error() string {
	return fmt.Sprintf("eki task %s: result unavailable: %v", e.TaskID, e.Err)
}
func (e *EKIResultError) Unwrap() error { return e.Err }

// EKITaskExecute submits a request and reads accepted result metadata.
// A result digest is not proof that output bytes were retrieved or verified.
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
		return capabilityinvoke.BusinessResult{}, errors.New("eki_task_execute requires a request object")
	}
	key, err := requireString("eki_task_execute", fields, "idempotencyKey")
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	if b.Poll < 0 || b.Timeout < 0 {
		return capabilityinvoke.BusinessResult{}, errors.New("poll and timeout must not be negative")
	}
	poll, timeout := b.Poll, b.Timeout
	if poll == 0 {
		poll = 500 * time.Millisecond
	}
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := b.Client
	if client == nil {
		client = http.DefaultClient
	}
	safeClient := *client
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	submitted, err := (EKITaskSubmitHTTP{BaseURL: b.BaseURL, Client: &safeClient}).Invoke(ctx, id, map[string]any{
		"idempotencyKey": key, "request": request,
	})
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("submit task (outcome may be unknown): %w", err)
	}
	payload, ok := submitted.Payload.(map[string]any)
	if !ok {
		return capabilityinvoke.BusinessResult{}, errors.New("submit payload is not an object")
	}
	taskID, _ := payload["task_id"].(string)
	if !ekiTaskIDPattern.MatchString(taskID) {
		return capabilityinvoke.BusinessResult{}, errors.New("submit response has invalid task_id; reconcile by idempotency key")
	}
	fail := func(err error) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{}, &EKIResultError{TaskID: taskID, Err: err}
	}
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(b.BaseURL, "/")+"/v1/tasks/"+taskID+"/result", nil)
		if err != nil {
			return fail(err)
		}
		resp, err := safeClient.Do(req)
		if err != nil {
			return fail(err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, ekiResultLimit+1))
		closeErr := resp.Body.Close()
		if readErr != nil {
			return fail(fmt.Errorf("read result: %w", readErr))
		}
		if closeErr != nil {
			return fail(fmt.Errorf("close result: %w", closeErr))
		}
		if len(body) > ekiResultLimit {
			return fail(errors.New("result response exceeds limit"))
		}
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict {
			if err := scenariotest.RejectDuplicateJSONMembers(body); err != nil {
				return fail(fmt.Errorf("invalid result JSON: %w", err))
			}
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var result map[string]any
			if err := json.Unmarshal(body, &result); err != nil {
				return fail(fmt.Errorf("decode result: %w", err))
			}
			assurance, aok := result["assurance"].(map[string]any)
			digest, dok := result["output_digest"].(map[string]any)
			_, artifactsOK := result["artifacts"].([]any)
			value, _ := digest["value"].(string)
			profile, _ := assurance["profile"].(string)
			if result["schema"] != "eki.task-result/v1" || result["task_id"] != taskID || !aok || assurance["status"] != "satisfied" ||
				(profile != "RA-1" && profile != "RA-2") || !dok || digest["algorithm"] != "sha256" || !ekiDigestPattern.MatchString(value) || !artifactsOK {
				return fail(errors.New("result metadata does not match the accepted-result contract"))
			}
			return capabilityinvoke.BusinessResult{Capability: id, Mode: capabilityinvoke.ResultOutput, Payload: result}, nil
		case http.StatusConflict:
			var problem struct {
				Code      string `json:"code"`
				Retryable bool   `json:"retryable"`
				TaskID    string `json:"task_id"`
			}
			if err := json.Unmarshal(body, &problem); err != nil {
				return fail(fmt.Errorf("decode result problem: %w", err))
			}
			if problem.Code != "RESULT_NOT_READY" || !problem.Retryable || problem.TaskID != taskID {
				return fail(fmt.Errorf("result status %d is not a retryable pending result", resp.StatusCode))
			}
			timer := time.NewTimer(poll)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fail(ctx.Err())
			case <-timer.C:
			}
		default:
			return fail(fmt.Errorf("unexpected result status %d", resp.StatusCode))
		}
	}
}
