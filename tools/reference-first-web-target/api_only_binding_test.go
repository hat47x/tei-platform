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
		ApplicationID: "order_entry",
		HandoverBaselineID: "B",
		BaselineRepositoryRevision: strings.Repeat("a", 40),
		TargetBuildRevision: strings.Repeat("b", 40),
	}}
	goodHeaders := map[string]string{
		apiBoundApplicationHeader: "order_entry",
		apiBoundBaselineHeader: "B",
		apiBoundRevisionHeader: strings.Repeat("a", 40),
		apiBoundTargetHeader: strings.Repeat("b", 40),
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
