//go:build !tei_repair_only

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// This table needs no Workspace or synthetic identity fixture at runtime.
// It checks the precise HTTP classification of the Session Runner's final
// business stages and protects against leaking Adapter errors to a browser.
func TestProtectedSessionResultNeverTreatsUnconfirmedCommandAsSuccess(t *testing.T) {
	internal := errors.New("private adapter credential or provider exception")
	const securityFailure = "trusted identity or authorization resolution unavailable"
	const effectUnknown = "action completion unconfirmed; inspect authoritative state before retrying"
	for _, tc := range []struct {
		name        string
		stage       string
		command     bool
		observation bool
		err         error
		status      int
		errorText   string
	}{
		{"completed with command and observation", "completed", true, true, nil, http.StatusOK, ""},
		{"validation business rejection", "validation-not-satisfied", false, false, nil, http.StatusUnprocessableEntity, ""},
		{"identity business rejection", "identity-link-denied", false, false, nil, http.StatusForbidden, ""},
		{"authorization business rejection", "authorization-denied", false, false, nil, http.StatusForbidden, ""},
		{"identity backend outage", "identity-link", false, false, internal, http.StatusServiceUnavailable, securityFailure},
		{"first authorization outage", "authorization", false, false, internal, http.StatusServiceUnavailable, securityFailure},
		{"effect authorization outage", "effect-authorization", false, false, internal, http.StatusServiceUnavailable, securityFailure},
		{"unconfirmed command effect", "command", false, false, internal, http.StatusInternalServerError, effectUnknown},
		{"observation error after command", "observation", true, false, internal, http.StatusInternalServerError, effectUnknown},
		{"read error", "read", false, false, internal, http.StatusInternalServerError, effectUnknown},
		{"nonterminal stage without error", "command", false, false, nil, http.StatusInternalServerError, effectUnknown},
		{"completed without command", "completed", false, true, nil, http.StatusInternalServerError, effectUnknown},
		{"completed without observation", "completed", true, false, nil, http.StatusInternalServerError, effectUnknown},
		{"completed but execution error", "completed", true, true, internal, http.StatusInternalServerError, effectUnknown},
		{"validation stage plus execution error", "validation-not-satisfied", false, false, internal, http.StatusInternalServerError, effectUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := commandScenarioResult{ApplicationID: "order_entry", Stage: tc.stage}
			if tc.command {
				result.Command = &businessResultProjection{Capability: "sales.order.submit"}
			}
			if tc.observation {
				result.Observation = &observationProjection{}
			}
			w := httptest.NewRecorder()
			(&webTargetServer{}).writeProtectedSessionCommandHTTPResult(w, result, tc.err)
			if w.Code != tc.status {
				t.Fatalf("HTTP=%d, want=%d, body=%s", w.Code, tc.status, w.Body.String())
			}
			if strings.Contains(w.Body.String(), internal.Error()) {
				t.Fatal("internal Adapter error leaked into a Session HTTP response")
			}
			if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("unexpected response content type: %s", w.Header().Get("Content-Type"))
			}
			if tc.errorText == "" {
				var response webCommandScenarioResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Result.Stage != tc.stage {
					t.Fatalf("lost Business Result: %+v", response)
				}
				return
			}
			var response webCommandScenarioErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Result.Stage != tc.stage || response.ExecutionError != tc.errorText {
				t.Fatalf("error classification=%+v, want stage %s and text %q", response, tc.stage, tc.errorText)
			}
		})
	}
}
