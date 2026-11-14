package bindings

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testEKITaskID = "tsk_00000000000000000000000000000001"

func executeInput() map[string]any {
	return map[string]any{"idempotencyKey": "test-key", "request": map[string]any{"schema": "eki.task-request/v1"}}
}
func acceptedMetadata(taskID string) string {
	return fmt.Sprintf(`{"schema":"eki.task-result/v1","task_id":%q,"artifacts":[],"assurance":{"profile":"RA-2","status":"satisfied"},"output_digest":{"algorithm":"sha256","value":%q}}`, taskID, strings.Repeat("a", 64))
}

func TestEKITaskExecutePollsPendingThenAcceptedMetadata(t *testing.T) {
	var gets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/tasks" {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, `{"task_id":%q}`, testEKITaskID)
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/v1/tasks/"+testEKITaskID+"/result" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if gets.Add(1) == 1 {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprintf(w, `{"code":"RESULT_NOT_READY","retryable":true,"task_id":%q}`, testEKITaskID)
			return
		}
		fmt.Fprint(w, acceptedMetadata(testEKITaskID))
	}))
	defer server.Close()
	b := EKITaskExecute{BaseURL: server.URL, Client: server.Client(), Poll: time.Millisecond, Timeout: time.Second}
	result, err := b.Invoke(context.Background(), "product_contracts.eki_task_execute", executeInput())
	if err != nil {
		t.Fatal(err)
	}
	payload, ok := result.Payload.(map[string]any)
	if !ok || payload["task_id"] != testEKITaskID {
		t.Fatalf("result = %#v", result.Payload)
	}
	if gets.Load() != 2 {
		t.Fatalf("GET count = %d", gets.Load())
	}
}

func TestEKITaskExecuteRejectsUnqualifiedResponses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		submit  string
		status  int
		body    string
		wantGet bool
	}{
		{"invalid task ID", `{"task_id":"../other"}`, 200, "", false},
		{"duplicate acceptance", fmt.Sprintf(`{"task_id":%q,"task_id":%q}`, testEKITaskID, testEKITaskID), 200, "", false},
		{"different task", "", 200, acceptedMetadata("tsk_00000000000000000000000000000002"), true},
		{"duplicate schema", "", 200, fmt.Sprintf(`{"schema":"bad","schema":"eki.task-result/v1","task_id":%q}`, testEKITaskID), true},
		{"bad digest", "", 200, strings.Replace(acceptedMetadata(testEKITaskID), strings.Repeat("a", 64), "short", 1), true},
		{"unknown assurance", "", 200, strings.Replace(acceptedMetadata(testEKITaskID), "RA-2", "unverified", 1), true},
		{"missing artifacts", "", 200, strings.Replace(acceptedMetadata(testEKITaskID), `"artifacts":[],`, "", 1), true},
		{"terminal result", "", 409, fmt.Sprintf(`{"code":"RESULT_NOT_AVAILABLE","retryable":false,"task_id":%q}`, testEKITaskID), true},
		{"pending not retryable", "", 409, fmt.Sprintf(`{"code":"RESULT_NOT_READY","retryable":false,"task_id":%q}`, testEKITaskID), true},
		{"pending for other task", "", 409, `{"code":"RESULT_NOT_READY","retryable":true,"task_id":"tsk_00000000000000000000000000000002"}`, true},
		{"unimplemented result", "", 501, "", true},
		{"oversized result", "", 200, strings.Repeat("x", ekiResultLimit+1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gets atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusAccepted)
					body := tc.submit
					if body == "" {
						body = fmt.Sprintf(`{"task_id":%q}`, testEKITaskID)
					}
					fmt.Fprint(w, body)
					return
				}
				gets.Add(1)
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			_, err := (EKITaskExecute{BaseURL: server.URL, Client: server.Client(), Poll: time.Millisecond, Timeout: time.Second}).Invoke(context.Background(), "test", executeInput())
			if err == nil {
				t.Fatal("unqualified response was accepted")
			}
			if tc.wantGet {
				var resultErr *EKIResultError
				if !errors.As(err, &resultErr) || resultErr.TaskID != testEKITaskID {
					t.Fatalf("task ID unavailable in error: %v", err)
				}
				if gets.Load() != 1 {
					t.Fatalf("GET count = %d", gets.Load())
				}
			} else if gets.Load() != 0 {
				t.Fatalf("unexpected GET count = %d", gets.Load())
			}
		})
	}
}

func TestEKITaskExecuteTimeoutKeepsAcceptedTaskID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, `{"task_id":%q}`, testEKITaskID)
			return
		}
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, `{"code":"RESULT_NOT_READY","retryable":true,"task_id":%q}`, testEKITaskID)
	}))
	defer server.Close()
	_, err := (EKITaskExecute{BaseURL: server.URL, Client: server.Client(), Poll: time.Second, Timeout: 200 * time.Millisecond}).Invoke(context.Background(), "test", executeInput())
	var resultErr *EKIResultError
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &resultErr) || resultErr.TaskID != testEKITaskID {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestEKITaskExecuteDoesNotEchoDuplicateMemberNames(t *testing.T) {
	for _, onSubmit := range []bool{true, false} {
		t.Run(fmt.Sprintf("submit=%t", onSubmit), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusAccepted)
					if onSubmit {
						fmt.Fprintf(w, `{"task_id":%q,"private-canary":1,"private-canary":2}`, testEKITaskID)
					} else {
						fmt.Fprintf(w, `{"task_id":%q}`, testEKITaskID)
					}
					return
				}
				fmt.Fprint(w, `{"private-canary":1,"private-canary":2}`)
			}))
			defer server.Close()
			_, err := (EKITaskExecute{BaseURL: server.URL, Client: server.Client(), Timeout: time.Second}).Invoke(context.Background(), "test", executeInput())
			if err == nil || strings.Contains(err.Error(), "private-canary") {
				t.Fatalf("duplicate member leaked or accepted: %v", err)
			}
		})
	}
}

func TestEKITaskExecuteDoesNotFollowSubmitRedirect(t *testing.T) {
	var reached atomic.Int32
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1); w.WriteHeader(202) }))
	defer sink.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, sink.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	_, err := (EKITaskExecute{BaseURL: source.URL, Client: source.Client(), Timeout: time.Second}).Invoke(context.Background(), "test", executeInput())
	if err == nil || reached.Load() != 0 {
		t.Fatalf("redirect was followed: reached=%d err=%v", reached.Load(), err)
	}
}

func TestEKITaskExecuteDoesNotFollowResultRedirect(t *testing.T) {
	var reached atomic.Int32
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1) }))
	defer sink.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, `{"task_id":%q}`, testEKITaskID)
			return
		}
		http.Redirect(w, r, sink.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	_, err := (EKITaskExecute{BaseURL: source.URL, Client: source.Client(), Timeout: time.Second}).Invoke(context.Background(), "test", executeInput())
	var resultErr *EKIResultError
	if !errors.As(err, &resultErr) || resultErr.TaskID != testEKITaskID || reached.Load() != 0 {
		t.Fatalf("redirect result: reached=%d err=%v", reached.Load(), err)
	}
}

func TestEKITaskExecuteRejectsInvalidTimeoutBeforeSubmission(t *testing.T) {
	var reached atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1) }))
	defer server.Close()
	_, err := (EKITaskExecute{BaseURL: server.URL, Client: server.Client(), Timeout: -time.Second}).Invoke(context.Background(), "test", executeInput())
	if err == nil || reached.Load() != 0 {
		t.Fatalf("invalid timeout submitted task: reached=%d err=%v", reached.Load(), err)
	}
}

func TestManagedTaskExecuteNeverReturnsSyntheticSuccess(t *testing.T) {
	result, err := (ManagedTaskExecute{}).Invoke(context.Background(), "test", nil)
	if err == nil || result.Payload != nil {
		t.Fatalf("managed execution = %#v, %v", result, err)
	}
}
