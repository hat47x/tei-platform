package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The original multi-Application Target still exposes all its diagnostic
// routes. Moving the registration beside each Application implementation
// must not silently change the legacy Gate C HTTP status contract.
func TestCombinedReferenceRouteRegistrationKeepsGateCStatuses(t *testing.T) {
	for _, app := range []string{"order_entry", "repair_intake"} {
		t.Run(app, func(t *testing.T) {
			server, err := newWebTargetServer(webTargetProjection{ApplicationID: app}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			handler := server.Handler()
			for _, tc := range []struct {
				method, path string
				status int
			}{
				{http.MethodGet, "/scenario/read", http.StatusServiceUnavailable},
				{http.MethodPost, "/scenario/read", http.StatusMethodNotAllowed},
				{http.MethodPost, "/scenario/submit", http.StatusServiceUnavailable},
				{http.MethodGet, "/scenario/submit", http.StatusMethodNotAllowed},
				{http.MethodGet, "/scenario/customer-lookup", http.StatusServiceUnavailable},
				{http.MethodPost, "/scenario/customer-lookup", http.StatusMethodNotAllowed},
				{http.MethodGet, "/customer-lookup", http.StatusServiceUnavailable},
				{http.MethodGet, "/manifest.json", http.StatusOK},
				{http.MethodGet, "/healthz", http.StatusOK},
				{http.MethodGet, "/invalid-business-path", http.StatusNotFound},
			} {
				received := httptest.NewRecorder()
				handler.ServeHTTP(received, httptest.NewRequest(tc.method, tc.path, nil))
				if received.Code != tc.status {
					t.Errorf("%s %s: got HTTP %d, expected %d", tc.method, tc.path, received.Code, tc.status)
				}
			}
		})
	}
}
