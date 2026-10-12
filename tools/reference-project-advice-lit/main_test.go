package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	customeradvice "tei/plugins/reference-project-customer-advice"
)

func TestProjectAdvisoryHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8787/project/advice",
		strings.NewReader(`{"customerCode":"VIP-7","note":""}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resp.Code, resp.Body.String())
	}
	var result customeradvice.Result
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil ||
		result.Kind != "review_suggested" || !result.Advisory {
		t.Fatalf("unexpected result: %#v, err %v", result, err)
	}
	if got := resp.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache policy = %q", got)
	}
}

func TestProjectAdvisoryRejectsMalformedOrForbidden(t *testing.T) {
	tests := []struct {
		name, method, contentType, body, origin string
		want                                    int
	}{
		{"method", "GET", "application/json", "", "", 405},
		{"media", "POST", "text/plain", "{}", "", 415},
		{"cross-origin", "POST", "application/json", "{}", "https://example.invalid", 403},
		{"unknown", "POST", "application/json", `{"customerCode":"A","note":"","authority":"admin"}`, "", 400},
		{"trailing", "POST", "application/json", `{"customerCode":"A","note":""}{}`, "", 400},
		{"bad-code", "POST", "application/json", `{"customerCode":"<script>","note":""}`, "", 400},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://127.0.0.1:8787/project/advice", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			resp := httptest.NewRecorder()
			handler().ServeHTTP(resp, req)
			if resp.Code != tc.want {
				t.Fatalf("status=%d, want=%d", resp.Code, tc.want)
			}
		})
	}
}

func TestReferenceServesOnlyKnownAssets(t *testing.T) {
	for _, path := range []string{"/", "/advice-element.mjs", "/advice-action.mjs", "/order-entry-context.mjs"} {
		req := httptest.NewRequest("GET", path, nil)
		resp := httptest.NewRecorder()
		handler().ServeHTTP(resp, req)
		if resp.Code != 200 || resp.Body.Len() == 0 {
			t.Fatalf("path %s status %d size %d", path, resp.Code, resp.Body.Len())
		}
	}
	req := httptest.NewRequest("GET", "/secret.txt", nil)
	resp := httptest.NewRecorder()
	handler().ServeHTTP(resp, req)
	if resp.Code != 404 {
		t.Fatalf("unknown path returned %d", resp.Code)
	}
}
