//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIOnlyExcludesServerSideHTMLAndRoutes(t *testing.T) {
	server, err := newWebTargetServer(webTargetProjection{ApplicationID: "order_entry"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, method string
		want         int
	}{
		{"/", http.MethodGet, http.StatusNotFound},
		{"/custom-submit", http.MethodGet, http.StatusNotFound},
		{"/custom-submit", http.MethodPost, http.StatusNotFound},
		{"/customer-lookup", http.MethodGet, http.StatusNotFound},
		{"/manifest.json", http.MethodGet, http.StatusOK},
		{"/healthz", http.MethodGet, http.StatusOK},
		{"/scenario/read", http.MethodGet, http.StatusServiceUnavailable},
		{"/scenario/submit", http.MethodPost, http.StatusServiceUnavailable},
	} {
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		if response.Code != tc.want {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, response.Code, tc.want)
		}
	}
}
