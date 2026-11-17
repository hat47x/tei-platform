package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/studiohost"
)

const testInspectionToken = "test-only-0123456789abcdef0123456789abcdef"

func TestInspectionHTTPRequiresExplicitLocalCredential(t *testing.T) {
	if _, err := newInspectionHTTPService(nil, testInspectionToken); err == nil {
		t.Fatal("nil Studio Host must be rejected")
	}
	host, err := studiohost.NewReadOnly([]string{filepath.Join("..", "..", "internal", "workspace", "testdata", "tei-md-rule")})
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"", "short", " " + testInspectionToken} {
		if _, err := newInspectionHTTPService(host, token); err == nil {
			t.Fatalf("invalid bearer token %q was accepted", token)
		}
	}
}

func TestInspectionHTTPProjectsOnlyAuthorizedReadOnlyInspection(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "workspace", "testdata", "tei-md-rule")
	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := newInspectionHTTPService(host, testInspectionToken)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(svc)
	defer server.Close()
	const id = "sales.order_line.quantity_positive"
	query := func(workspace, kind, subject string) string {
		return inspectionHTTPRoute + "?" + url.Values{"workspace": {workspace}, "kind": {kind}, "subject": {subject}}.Encode()
	}
	request := func(method, path, authorization, origin string) (int, http.Header, []byte) {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, res.Header, body
	}
	validPath := query("workspace-1", "semantic-validation", id)
	for _, tc := range []struct {
		name, method, path, auth, origin string
		want                             int
	}{
		{"unknown-route", http.MethodGet, "/tei/read-only/v0/other", "Bearer " + testInspectionToken, "", http.StatusNotFound},
		{"no-token", http.MethodGet, validPath, "", "", http.StatusUnauthorized},
		{"bad-token", http.MethodGet, validPath, "Bearer wrong", "", http.StatusUnauthorized},
		{"cross-origin", http.MethodGet, validPath, "Bearer " + testInspectionToken, "https://attacker.example", http.StatusForbidden},
		{"no-mutation", http.MethodPost, validPath, "Bearer " + testInspectionToken, "", http.StatusMethodNotAllowed},
		{"unknown-query", http.MethodGet, validPath + "&extra=1", "Bearer " + testInspectionToken, "", http.StatusBadRequest},
		{"repeated-subject", http.MethodGet, validPath + "&subject=other", "Bearer " + testInspectionToken, "", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, headers, body := request(tc.method, tc.path, tc.auth, tc.origin)
			if status != tc.want {
				t.Fatalf("HTTP status=%d want=%d body=%s", status, tc.want, body)
			}
			if headers.Get("Cache-Control") != "no-store" || headers.Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("HTTP response caching or type sniffing not prohibited: %+v", headers)
			}
		})
	}

	status, headers, raw := request(http.MethodGet, validPath, "Bearer "+testInspectionToken, "")
	if status != http.StatusOK || !strings.HasPrefix(headers.Get("Content-Type"), "application/json") {
		t.Fatalf("authorized inspection status=%d Content-Type=%q body=%s", status, headers.Get("Content-Type"), raw)
	}
	var result studiohost.InspectionResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	direct := host.InspectSubject("workspace-1", "semantic-validation", id)
	if !result.OK || result.Failure != nil || result.Value == nil || !reflect.DeepEqual(result.Value, direct.Value) {
		t.Fatalf("HTTP projection differs from the typed Studio Operation: HTTP=%+v direct=%+v", result, direct)
	}

	for _, tc := range []struct {
		name, workspace, kind, id, code string
		status                          int
	}{
		{"unlisted-workspace", "workspace-2", "semantic-validation", id, "WORKSPACE_NOT_ALLOWED", http.StatusForbidden},
		{"unpublished-capability", "workspace-1", "semantic-capability", id, "SUBJECT_NOT_FOUND", http.StatusNotFound},
		{"not-a-rule", "workspace-1", "semantic-validation", "sales.order_line.unit_price_positive", "SUBJECT_NOT_FOUND", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, _, raw := request(http.MethodGet, query(tc.workspace, tc.kind, tc.id), "Bearer "+testInspectionToken, "")
			if status != tc.status {
				t.Fatalf("status=%d want=%d body=%s", status, tc.status, raw)
			}
			var result studiohost.InspectionResponse
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
			if result.OK || result.Value != nil || result.Failure == nil || result.Failure.Code != tc.code ||
				result.Failure.Kind != "business_rejection" || result.Failure.Retryable != "not_retryable" || result.Failure.EffectUnknown ||
				result.Failure.Message != "inspection rejected" {
				t.Fatalf("HTTP failure classification or redaction was lost: %+v", result)
			}
		})
	}
}
