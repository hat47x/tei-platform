package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Opt-in scoping must never expose a route belonging to the other
// Application. Missing own runners still return 503 (not fake success).
func TestApplicationScopedRoutesAreDisjoint(t *testing.T) {
	for _, tc := range []struct {
		application string
		own         []string
		foreign     []string
	}{
		{
			application: "order_entry",
			own: []string{"/scenario/read", "/scenario/submit", "/custom-submit"},
			foreign: []string{"/scenario/customer-lookup", "/customer-lookup"},
		},
		{
			application: "repair_intake",
			own: []string{"/scenario/customer-lookup", "/customer-lookup"},
			foreign: []string{"/scenario/read", "/scenario/submit", "/custom-submit"},
		},
	} {
		t.Run(tc.application, func(t *testing.T) {
			server, err := newWebTargetServer(webTargetProjection{ApplicationID: tc.application}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			scoped, err := server.ScopedHandler()
			if err != nil {
				t.Fatal(err)
			}
			for _, route := range tc.foreign {
				for _, method := range []string{http.MethodGet, http.MethodPost} {
					response := httptest.NewRecorder()
					scoped.ServeHTTP(response, httptest.NewRequest(method, route, nil))
					if response.Code != http.StatusNotFound {
						t.Errorf("foreign route %s %s exposed HTTP %d", method, route, response.Code)
					}
				}
			}
			for _, route := range tc.own {
				response := httptest.NewRecorder()
				method := http.MethodGet
				if route == "/scenario/submit" || route == "/custom-submit" {
					method = http.MethodPost
				}
				scoped.ServeHTTP(response, httptest.NewRequest(method, route, nil))
				if response.Code != http.StatusServiceUnavailable {
					t.Errorf("own route %s %s returned %d without runner, expected 503", method, route, response.Code)
				}
			}
			for _, route := range []string{"/manifest.json", "/healthz"} {
				response := httptest.NewRecorder()
				scoped.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
				if response.Code != http.StatusOK {
					t.Errorf("common route %s returned %d, expected 200", route, response.Code)
				}
			}
		})
	}
}

func TestApplicationScopedRoutesRejectUnknownApplication(t *testing.T) {
	if handler, err := (*webTargetServer)(nil).ScopedHandler(); err == nil || handler != nil {
		t.Fatalf("nil server accepted: handler=%v err=%v", handler, err)
	}
	for _, app := range []string{"", "unknown", "ORDER_ENTRY", "../order_entry"} {
		server, err := newWebTargetServer(webTargetProjection{ApplicationID: app}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if handler, err := server.ScopedHandler(); err == nil || handler != nil {
			t.Errorf("unknown Application %q accepted: handler=%v err=%v", app, handler, err)
		}
	}
}

func TestScopedRoutingDoesNotMutateLegacyGateCSurface(t *testing.T) {
	server, err := newWebTargetServer(webTargetProjection{ApplicationID: "order_entry"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.ScopedHandler(); err != nil {
		t.Fatal(err)
	}
	// Legacy Handler is deliberately unchanged for Gate C compatibility.
	for _, route := range []string{"/scenario/customer-lookup", "/scenario/read"} {
		result := httptest.NewRecorder()
		server.Handler().ServeHTTP(result, httptest.NewRequest(http.MethodGet, route, nil))
		if result.Code != http.StatusServiceUnavailable {
			t.Fatalf("historical multi-App route %s changed: got %d", route, result.Code)
		}
	}
}
