package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The same fixture-backed Business Result, Execution Error, Observation and
// denial must be preserved by opt-in route scoping. This is NOT package or
// binary isolation, and NOT evidence of production authorization.
func TestScopedBusinessRoutesMatchExistingCombinedTarget(t *testing.T) {
	world := loadReferenceWorld(t)
	commandRunner, err := newReferenceCommandScenarioRunner(
		world,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	order, err := newWebTargetServer(
		mustOrderEntryProjection(t, world),
		newReferenceReadScenarioRunner(world),
		commandRunner,
	)
	if err != nil {
		t.Fatal(err)
	}
	repair, err := newWebTargetServer(
		repairIntakeProjection(t, world),
		nil,
		nil,
		withCustomerLookup(newReferenceCustomerLookupScenarioRunner(world)),
	)
	if err != nil {
		t.Fatal(err)
	}
	type requestCase struct {
		method, path, body string
		status             int
	}
	for _, tc := range []struct {
		name   string
		server *webTargetServer
		cases  []requestCase
	}{
		{
			name: "order_entry",
			server: order,
			cases: []requestCase{
				{http.MethodGet, "/manifest.json", "", http.StatusOK},
				{http.MethodGet, "/healthz", "", http.StatusOK},
				{http.MethodGet, "/scenario/read?from=2026-09-18T08:30:00Z&row=0", "", http.StatusOK},
				{http.MethodGet, "/scenario/read?row=-1", "", http.StatusBadRequest},
				{http.MethodPost, "/scenario/submit", `{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`, http.StatusOK},
				{http.MethodPost, "/scenario/submit", `{"loginId":"legacy-order-viewer","credential":"viewer-secret","from":"2026-09-18T08:30:00Z","row":0}`, http.StatusOK},
				{http.MethodPost, "/scenario/submit", `{"loginId":"legacy-order-editor","credential":"wrong-secret","from":"2026-09-18T08:30:00Z","row":0}`, http.StatusOK},
				{http.MethodPost, "/scenario/submit", `{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T07:00:00Z","row":2}`, http.StatusUnprocessableEntity},
				{http.MethodPost, "/scenario/submit", `{"unknown":"field"}`, http.StatusBadRequest},
			},
		},
		{
			name: "repair_intake",
			server: repair,
			cases: []requestCase{
				{http.MethodGet, "/manifest.json", "", http.StatusOK},
				{http.MethodGet, "/healthz", "", http.StatusOK},
				{http.MethodGet, "/scenario/customer-lookup?customerId=11", "", http.StatusOK},
				{http.MethodGet, "/scenario/customer-lookup?customerId=abc", "", http.StatusBadRequest},
				{http.MethodGet, "/scenario/customer-lookup", "", http.StatusBadRequest},
				{http.MethodPost, "/scenario/customer-lookup?customerId=11", "", http.StatusMethodNotAllowed},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scoped, err := tc.server.ScopedHandler()
			if err != nil {
				t.Fatal(err)
			}
			combined := tc.server.Handler()
			for _, input := range tc.cases {
				t.Run(input.method+" "+input.path+" "+input.body, func(t *testing.T) {
					invoke := func(handler http.Handler) *httptest.ResponseRecorder {
						req := httptest.NewRequest(input.method, input.path, strings.NewReader(input.body))
						if input.body != "" {
							req.Header.Set("Content-Type", "application/json")
						}
						res := httptest.NewRecorder()
						handler.ServeHTTP(res, req)
						return res
					}
					before := invoke(combined)
					after := invoke(scoped)
					if before.Code != input.status || after.Code != input.status {
						t.Fatalf("%s %s: historical HTTP %d, scoped HTTP %d, expected %d",
							input.method, input.path, before.Code, after.Code, input.status)
					}
					if before.Code != after.Code ||
						before.Body.String() != after.Body.String() ||
						before.Header().Get("Content-Type") != after.Header().Get("Content-Type") {
						t.Fatalf("route parity lost: combined HTTP %d %q; scoped HTTP %d %q",
							before.Code, before.Body.String(), after.Code, after.Body.String())
					}
					for _, credential := range []string{"editor-secret", "viewer-secret", "wrong-secret"} {
						if strings.Contains(after.Body.String(), credential) {
							t.Fatal("credential leaked in HTTP response")
						}
					}
				})
			}
		})
	}
}
