package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tei/internal/composition"
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
	for _, path := range []string{"/", "/advice-element.mjs", "/advice-action.mjs", "/order-entry-context.mjs", "/lit-core.min.js", "/lit-license.txt"} {
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

func TestDiagnosticHostRejectsDNSRebinding(t *testing.T) {
	readCalls := 0
	readHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		readCalls++
		w.WriteHeader(http.StatusOK)
	})
	secured := diagnosticHostOnly(handlerWithContext(readHandler))
	for _, tc := range []struct {
		name, method, path, host, body string
	}{
		{"static-rebinding", "GET", "/", "attacker.invalid:8787", ""},
		{"read-rebinding", "GET", "/project/order-entry-context", "attacker.invalid:8787", ""},
		{"advisory-rebinding", "POST", "/project/advice", "attacker.invalid:8787", `{"customerCode":"VIP-7","note":""}`},
		{"suffix-host", "GET", "/", "127.0.0.1.attacker.invalid:8787", ""},
		{"alternate-host", "GET", "/", "localhost:8787", ""},
		{"wrong-port", "GET", "/", "127.0.0.1:8788", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://127.0.0.1:8787"+tc.path, strings.NewReader(tc.body))
			req.Host = tc.host
			if tc.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
				// A DNS-rebound origin matches the malicious Host but is never trusted.
				req.Header.Set("Origin", "http://"+tc.host)
			}
			resp := httptest.NewRecorder()
			secured.ServeHTTP(resp, req)
			if resp.Code != http.StatusMisdirectedRequest {
				t.Fatalf("host %q: status=%d, want=421", tc.host, resp.Code)
			}
			if resp.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("rejected host response must not be cached")
			}
		})
	}
	if readCalls != 0 {
		t.Fatalf("DNS-rebinding request reached upstream: %d", readCalls)
	}
	for _, tc := range []struct {
		method, path, body string
	}{
		{"GET", "/", ""},
		{"GET", "/project/order-entry-context", ""},
		{"POST", "/project/advice", `{"customerCode":"VIP-7","note":""}`},
	} {
		req := httptest.NewRequest(tc.method, "http://127.0.0.1:8787"+tc.path, strings.NewReader(tc.body))
		if tc.method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "http://127.0.0.1:8787")
		}
		resp := httptest.NewRecorder()
		secured.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("canonical Host %s %s: status=%d body=%s", tc.method, tc.path, resp.Code, resp.Body.String())
		}
	}
	if readCalls != 1 {
		t.Fatalf("expected exactly one canonical read, got %d", readCalls)
	}
}

func TestProjectAdvisoryHTTPServiceUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name string
		invalidate func(*testing.T, *projectAdvisoryBridge)
	}{
		{"host-withdraw", func(t *testing.T, bridge *projectAdvisoryBridge) {
			if err := bridge.coordinator.Withdraw(1); err != nil {
				t.Fatal(err)
			}
		}},
		{"provider-removed", func(t *testing.T, bridge *projectAdvisoryBridge) {
			if _, err := bridge.registry.Apply(bridge.registry.Snapshot().Revision, nil, []string{projectAdvisoryProviderID}); err != nil {
				t.Fatal(err)
			}
		}},
		{"contract-changed", func(t *testing.T, bridge *projectAdvisoryBridge) {
			contract := ProjectAdvisoryContract
			contract.Version = "v1"
			if _, err := bridge.registry.Apply(bridge.registry.Snapshot().Revision, []composition.Provider{{
				ID: projectAdvisoryProviderID, Scope: projectAdvisoryScope, Offers: contract,
			}}, nil); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bridge, err := newProjectAdvisoryBridge()
			if err != nil {
				t.Fatal(err)
			}
			tc.invalidate(t, bridge)
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8787/project/advice",
				strings.NewReader(`{"customerCode":"VIP-7","note":""}`))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			diagnosticHostOnly(handlerWithProjectBridge(nil, bridge)).ServeHTTP(resp, req)
			if resp.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected unavailable 503, got %d: %s", resp.Code, resp.Body.String())
			}
			if resp.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("Host withdrawal response must not be cached")
			}
			if strings.Contains(resp.Body.String(), "メモの記入を推奨") {
				t.Fatal("unavailable provider must not produce advisory")
			}
		})
	}
}

func TestProjectAdvisoryHTTPInvalidInputRemains400(t *testing.T) {
	bridge, err := newProjectAdvisoryBridge()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8787/project/advice",
		strings.NewReader(`{"customerCode":"<script>","note":""}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	diagnosticHostOnly(handlerWithProjectBridge(nil, bridge)).ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("invalid input must stay 400, got %d", resp.Code)
	}
}
