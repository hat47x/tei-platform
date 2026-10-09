package webcomposition

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type observation struct {
	status  int
	calls   int
	authz   int
	records []AuditRecord
}

func fixture(t *testing.T, principal Principal, sessionErr error, authorizeErr error, allowed bool) (http.Handler, *observation) {
	t.Helper()
	obs := &observation{}
	h, err := NewHost(Policy{
		Path: "/scenario/submit", Method: http.MethodPost,
		ActionID: "order_entry.submit_order",
		Providers: map[string]string{
			"corporate": "https://issuer.example.invalid/tenant1",
			"legacy":    "legacy:order_entry",
		},
	}, Dependencies{
		VerifySession: func(_ *http.Request) (Principal, error) { return principal, sessionErr },
		VerifyCSRF:    func(r *http.Request, _ Principal) bool { return r.Header.Get("X-CSRF-Fixture") == "matching" },
		Authorize: func(_ context.Context, _ Principal, action string) (bool, error) {
			obs.authz++
			if action != "order_entry.submit_order" {
				t.Errorf("unexpected action %q", action)
			}
			return allowed, authorizeErr
		},
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { obs.calls++; w.WriteHeader(http.StatusAccepted) }),
		Audit:  func(record AuditRecord) { obs.records = append(obs.records, record) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return h, obs
}

func request(h http.Handler, path, method, csrf string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.test"+path, nil)
	req.Header.Set("X-CSRF-Fixture", csrf)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestProtectedActionOnlyAfterAllHostOwnedChecks(t *testing.T) {
	for _, provider := range []Principal{
		{ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1", Subject: "s1", PrincipalID: "employee-1"},
		{ProviderID: "legacy", Issuer: "legacy:order_entry", Subject: "legacy-user-1", PrincipalID: "employee-1"},
	} {
		h, obs := fixture(t, provider, nil, nil, true)
		w := request(h, "/scenario/submit", http.MethodPost, "matching")
		if w.Code != http.StatusAccepted || obs.calls != 1 || obs.authz != 1 || len(obs.records) != 1 || obs.records[0].Stage != "completed" {
			t.Fatalf("unexpected success result: status=%d observation=%+v", w.Code, obs)
		}
	}
}
func TestDeniedRoutesNeverInvokeActionAndAreAudited(t *testing.T) {
	p := Principal{ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1", Subject: "s1", PrincipalID: "employee-1"}
	cases := []struct {
		name, path, method, csrf string
		subject                  Principal
		sessionErr, authzErr     error
		allowed                  bool
		want                     int
		wantAuthz                int
	}{
		{name: "unauthenticated", path: "/scenario/submit", method: "POST", csrf: "matching", sessionErr: errors.New("no session"), allowed: true, want: 401},
		{name: "unknown provider", path: "/scenario/submit", method: "POST", csrf: "matching", subject: Principal{ProviderID: "unknown", Issuer: "other", Subject: "a", PrincipalID: "employee-1"}, allowed: true, want: 401},
		{name: "issuer confusion", path: "/scenario/submit", method: "POST", csrf: "matching", subject: Principal{ProviderID: "corporate", Issuer: "https://attacker.invalid", Subject: "s1", PrincipalID: "employee-1"}, allowed: true, want: 401},
		{name: "CSRF failed", path: "/scenario/submit", method: "POST", subject: p, allowed: true, want: 403},
		{name: "business permission denied", path: "/scenario/submit", method: "POST", csrf: "matching", subject: p, want: 403, wantAuthz: 1},
		{name: "authorization provider unavailable", path: "/scenario/submit", method: "POST", csrf: "matching", subject: p, authzErr: errors.New("down"), want: 503, wantAuthz: 1},
		{name: "wrong method", path: "/scenario/submit", method: "GET", csrf: "matching", subject: p, allowed: true, want: 405},
		{name: "different path", path: "/different", method: "POST", csrf: "matching", subject: p, allowed: true, want: 404},
		{name: "encoded alias", path: "/%73cenario/submit", method: "POST", csrf: "matching", subject: p, allowed: true, want: 404},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			h, obs := fixture(t, tt.subject, tt.sessionErr, tt.authzErr, tt.allowed)
			w := request(h, tt.path, tt.method, tt.csrf)
			if w.Code != tt.want || obs.calls != 0 || obs.authz != tt.wantAuthz || len(obs.records) != 1 {
				t.Fatalf("status=%d, actionCalls=%d, authzCalls=%d, audit=%+v", w.Code, obs.calls, obs.authz, obs.records)
			}
			if obs.records[0].Status != tt.want {
				t.Fatalf("missing audit status: %+v", obs.records[0])
			}
			if tt.want >= 500 && obs.records[0].Stage != "error" {
				t.Fatalf("server errors must be audited as errors: %+v", obs.records[0])
			}
		})
	}
}
func TestPolicyConstructionMustFailClosed(t *testing.T) {
	h, obs := fixture(t, Principal{ProviderID: "legacy", Issuer: "legacy:order_entry", Subject: "s", PrincipalID: "p"}, nil, nil, true)
	if h == nil || obs == nil {
		t.Fatal("fixture failed")
	}
	p := Policy{Path: "/scenario/submit", Method: "POST", ActionID: "order_entry.submit_order", Providers: map[string]string{"legacy": "legacy:order_entry"}}
	d := Dependencies{}
	if _, err := NewHost(p, d); err == nil {
		t.Fatal("accepted missing guard dependencies")
	}
	p.Providers = map[string]string{"legacy": ""}
	if _, err := NewHost(p, d); err == nil {
		t.Fatal("accepted unspecified issuer")
	}
	p.Providers = map[string]string{"legacy": "legacy:order_entry"}
	p.Method = "GET"
	if _, err := NewHost(p, d); err == nil {
		t.Fatal("accepted GET state-changing Action")
	}
}
func TestAuthorizationNeverChecksClientSuppliedRole(t *testing.T) {
	p := Principal{ProviderID: "legacy", Issuer: "legacy:order_entry", Subject: "s", PrincipalID: "employee-1"}
	h, obs := fixture(t, p, nil, nil, false)
	req := httptest.NewRequest(http.MethodPost, "http://example.test/scenario/submit", strings.NewReader(`{"role":"admin"}`))
	req.Header.Set("X-CSRF-Fixture", "matching")
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden || obs.calls != 0 {
		t.Fatalf("header bypassed action permission: code=%d calls=%d", w.Code, obs.calls)
	}
}

// Changes to a caller-owned map after construction cannot bypass the frozen policy.
func TestProviderPolicyIsFrozenAtConstruction(t *testing.T) {
	providers := map[string]string{"corporate": "https://issuer.example.invalid/tenant1"}
	p := Policy{Path: "/scenario/submit", Method: "POST", ActionID: "order_entry.submit_order", Providers: providers}
	principal := Principal{ProviderID: "corporate", Issuer: "https://attacker.invalid", Subject: "s1", PrincipalID: "employee-1"}
	called := 0
	h, err := NewHost(p, Dependencies{
		VerifySession: func(*http.Request) (Principal, error) { return principal, nil },
		VerifyCSRF:    func(*http.Request, Principal) bool { return true },
		Authorize:     func(context.Context, Principal, string) (bool, error) { return true, nil },
		Invoke:        http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called++ }),
		Audit:         func(AuditRecord) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	providers["corporate"] = "https://attacker.invalid"
	w := request(h, "/scenario/submit", http.MethodPost, "matching")
	if w.Code != http.StatusUnauthorized || called != 0 {
		t.Fatalf("mutable policy bypass: status=%d called=%d", w.Code, called)
	}
}

// Exercise the actual net/http transport boundary on loopback, still with test-only providers.
func TestLoopbackProtectedAction(t *testing.T) {
	p := Principal{ProviderID: "legacy", Issuer: "legacy:order_entry", Subject: "s1", PrincipalID: "employee-1"}
	h, obs := fixture(t, p, nil, nil, true)
	server := httptest.NewServer(h)
	defer server.Close()
	for _, tc := range []struct {
		csrf   string
		status int
	}{{"", 403}, {"matching", 202}} {
		req, err := http.NewRequest(http.MethodPost, server.URL+"/scenario/submit", nil)
		if err != nil {
			t.Fatal(err)
		}
		if tc.csrf != "" {
			req.Header.Set("X-CSRF-Fixture", tc.csrf)
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Fatalf("loopback HTTP status %d, want %d", resp.StatusCode, tc.status)
		}
	}
	if obs.calls != 1 || len(obs.records) != 2 {
		t.Fatalf("action calls=%d, audit entries=%d", obs.calls, len(obs.records))
	}
}
