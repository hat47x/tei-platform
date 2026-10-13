//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIBoundCommandChecksSameRequestBeforeRunner(t *testing.T) {
	server := &webTargetServer{projection: webTargetProjection{
		ApplicationID:              "order_entry",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: strings.Repeat("a", 40),
		TargetBuildRevision:        strings.Repeat("b", 40),
	}}
	goodHeaders := map[string]string{
		apiBoundApplicationHeader: "order_entry",
		apiBoundBaselineHeader:    "B",
		apiBoundRevisionHeader:    strings.Repeat("a", 40),
		apiBoundTargetHeader:      strings.Repeat("b", 40),
	}
	call := func(path, method string, headers map[string]string, extra func(*http.Request)) int {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader("{}"))
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		if extra != nil {
			extra(r)
		}
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if got := call(apiBoundCommandPath, http.MethodPost, goodHeaders, nil); got != http.StatusServiceUnavailable {
		t.Fatalf("correct B/R/T should reach missing Runner (503), got %d", got)
	}
	for name := range goodHeaders {
		t.Run("missing-"+name, func(t *testing.T) {
			bad := map[string]string{}
			for key, value := range goodHeaders {
				if key != name {
					bad[key] = value
				}
			}
			if got := call(apiBoundCommandPath, http.MethodPost, bad, nil); got != http.StatusConflict {
				t.Fatalf("missing binding %s: got %d, want 409", name, got)
			}
		})
		t.Run("different-"+name, func(t *testing.T) {
			bad := map[string]string{}
			for key, value := range goodHeaders {
				bad[key] = value
			}
			bad[name] = "other-target"
			if got := call(apiBoundCommandPath, http.MethodPost, bad, nil); got != http.StatusConflict {
				t.Fatalf("different binding %s: got %d, want 409", name, got)
			}
		})
	}
	if got := call(apiBoundCommandPath, http.MethodPost, goodHeaders, func(r *http.Request) {
		r.Header.Add(apiBoundTargetHeader, strings.Repeat("b", 40))
	}); got != http.StatusConflict {
		t.Fatalf("ambiguous duplicate target header: got %d, want 409", got)
	}
	if got := call(apiBoundCommandPath, http.MethodGet, goodHeaders, nil); got != http.StatusMethodNotAllowed {
		t.Fatalf("GET bound command: got %d, want 405", got)
	}
	if got := call("/scenario/submit", http.MethodPost, nil, nil); got != http.StatusServiceUnavailable {
		t.Fatalf("existing direct API route should stay accessible, got %d", got)
	}
}

func TestAPIBoundReadChecksSameRequestBeforeRunner(t *testing.T) {
	server := &webTargetServer{projection: webTargetProjection{
		ApplicationID:              "order_entry",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: strings.Repeat("a", 40),
		TargetBuildRevision:        strings.Repeat("b", 40),
	}}
	good := map[string]string{
		apiBoundApplicationHeader: "order_entry",
		apiBoundBaselineHeader:    "B",
		apiBoundRevisionHeader:    strings.Repeat("a", 40),
		apiBoundTargetHeader:      strings.Repeat("b", 40),
	}
	call := func(method, path string, headers map[string]string, duplicate string) int {
		t.Helper()
		r := httptest.NewRequest(method, path, nil)
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		if duplicate != "" {
			r.Header.Add(duplicate, headers[duplicate])
		}
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, r)
		return w.Code
	}
	// A missing Runner is reached only after the in-process projection check.
	if got := call(http.MethodGet, apiBoundReadPath+"?row=0", good, ""); got != http.StatusServiceUnavailable {
		t.Fatalf("matching bound read should reach missing Runner (503), got %d", got)
	}
	for name := range good {
		t.Run("missing-"+name, func(t *testing.T) {
			bad := map[string]string{}
			for k, v := range good {
				if k != name {
					bad[k] = v
				}
			}
			if got := call(http.MethodGet, apiBoundReadPath, bad, ""); got != http.StatusConflict {
				t.Fatalf("missing %s: got %d, want 409", name, got)
			}
		})
		t.Run("different-"+name, func(t *testing.T) {
			bad := map[string]string{}
			for k, v := range good {
				bad[k] = v
			}
			bad[name] = "unrelated"
			if got := call(http.MethodGet, apiBoundReadPath, bad, ""); got != http.StatusConflict {
				t.Fatalf("different %s: got %d, want 409", name, got)
			}
		})
	}
	if got := call(http.MethodGet, apiBoundReadPath, good, apiBoundTargetHeader); got != http.StatusConflict {
		t.Fatalf("duplicate Target T header: got %d, want 409", got)
	}
	if got := call(http.MethodPost, apiBoundReadPath, good, ""); got != http.StatusMethodNotAllowed {
		t.Fatalf("POST bound read should return 405, got %d", got)
	}
	if got := call(http.MethodGet, "/scenario/read", nil, ""); got != http.StatusServiceUnavailable {
		t.Fatalf("existing direct read route must remain compatible, got %d", got)
	}
}
