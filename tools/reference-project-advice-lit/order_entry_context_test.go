package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrderEntryContextUsesOnlyReadReference(t *testing.T) {
	var count int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if r.Method != http.MethodGet || r.URL.Path != "/scenario/read" ||
			r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Errorf("unsafe upstream request: %#v", r)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"targetBuildRevision":"synthetic-T","result":{"applicationId":"order_entry","scenarioId":"read-reference","selectedRowIndex":2,"selectedOrder":{"secret":"never-forward"}}}`))
	}))
	defer upstream.Close()
	h, err := newOrderEntryContextHandler(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/project/order-entry-context", nil)
	req.Header.Set("Authorization", "Bearer synthetic-secret")
	req.Header.Set("Cookie", "session=synthetic-secret")
	resp := httptest.NewRecorder()
	handlerWithContext(h).ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resp.Code, resp.Body.String())
	}
	if count != 1 || strings.Contains(resp.Body.String(), "never-forward") {
		t.Fatalf("read reference forwarded unexpected data: %d %s", count, resp.Body.String())
	}
	var result orderEntryContext
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil ||
		result.ApplicationID != "order_entry" || result.ScenarioID != "read-reference" ||
		result.SelectedRowIndex != 2 || result.TargetBuildRevision != "synthetic-T" {
		t.Fatalf("unexpected result %#v: %v", result, err)
	}
	if resp.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("read result must not be cached")
	}
}

func TestOrderEntryContextRejectsInvalidSources(t *testing.T) {
	for _, origin := range []string{
		"https://127.0.0.1:8888", "http://localhost:8888", "http://127.0.0.1",
		"http://127.0.0.1:0", "http://127.0.0.1:65536",
		"http://127.0.0.1:8888/path", "http://127.0.0.1:8888?x=1",
		"http://127.0.0.1:8888@evil.example:80",
	} {
		if _, err := newOrderEntryContextHandler(origin); err == nil {
			t.Errorf("accepted invalid upstream: %q", origin)
		}
	}
}

func TestOrderEntryContextRejectsUnexpectedReadResponses(t *testing.T) {
	for _, tc := range []struct {
		name, data, contentType string
		status                  int
	}{
		{"wrong-application", `{"targetBuildRevision":"T","result":{"applicationId":"other","scenarioId":"s"}}`, "application/json", 200},
		{"missing-revision", `{"result":{"applicationId":"order_entry","scenarioId":"s"}}`, "application/json", 200},
		{"missing-scenario", `{"targetBuildRevision":"T","result":{"applicationId":"order_entry"}}`, "application/json", 200},
		{"trailing", `{"targetBuildRevision":"T","result":{"applicationId":"order_entry","scenarioId":"s"}}{}`, "application/json", 200},
		{"non-json", "<html>ok</html>", "text/html", 200},
		{"upstream-error", "unavailable", "application/json", 503},
		{"redirect", "", "application/json", 302},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				if tc.status == 302 {
					w.Header().Set("Location", "https://example.invalid")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.data))
			}))
			defer upstream.Close()
			h, err := newOrderEntryContextHandler(upstream.URL)
			if err != nil {
				t.Fatal(err)
			}
			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/project/order-entry-context", nil))
			if resp.Code != http.StatusBadGateway {
				t.Fatalf("status=%d want=502", resp.Code)
			}
		})
	}
}

func TestOrderEntryContextDoesNotCreateAWriteRoute(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("upstream must not be invoked")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()
	h, err := newOrderEntryContextHandler(upstream.URL)
	if err != nil { t.Fatal(err) }
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, httptest.NewRequest(http.MethodPost, "/project/order-entry-context", strings.NewReader("{}")))
	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", resp.Code)
	}
	none := httptest.NewRecorder()
	handler().ServeHTTP(none, httptest.NewRequest(http.MethodGet, "/project/order-entry-context", nil))
	if none.Code != http.StatusNotFound {
		t.Fatalf("unconfigured route=%d", none.Code)
	}
}
