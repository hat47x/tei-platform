package webcomposition

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// TestHDG21HostDecisionPathWitness checks a bounded set of real Go HTTP
// handler branches. Callback order is independent expected data. It is not
// proof that the E1 declarative DAG is bound to this Host, that all possible
// paths are covered, or that any business effect has been independently read.
func TestHDG21HostDecisionPathWitness(t *testing.T) {
	valid := Principal{
		ProviderID:  "corporate",
		Issuer:      "https://issuer.example.invalid/tenant1",
		Subject:     "subject-1",
		PrincipalID: "employee-1",
	}

	cases := []struct {
		id           string
		path         string
		method       string
		subject      Principal
		sessionError bool
		csrf         bool
		allowed      bool
		authzError   bool
		invokeStatus int
		panicAfter   bool
		wantStatus   int
		wantStage    string
		wantCalls    []string
	}{
		{"HDG-H01-route", "/other", "POST", valid, false, true, true, false, 202, false, 404, "rejected", []string{"audit"}},
		{"HDG-H02-method", "/scenario/submit", "GET", valid, false, true, true, false, 202, false, 405, "rejected", []string{"audit"}},
		{"HDG-H03-session", "/scenario/submit", "POST", valid, true, true, true, false, 202, false, 401, "rejected", []string{"session", "audit"}},
		{"HDG-H04-issuer", "/scenario/submit", "POST", Principal{ProviderID: "corporate", Issuer: "other", Subject: "subject-1", PrincipalID: "employee-1"}, false, true, true, false, 202, false, 401, "rejected", []string{"session", "audit"}},
		{"HDG-H05-csrf", "/scenario/submit", "POST", valid, false, false, true, false, 202, false, 403, "rejected", []string{"session", "csrf", "audit"}},
		{"HDG-H06-authority", "/scenario/submit", "POST", valid, false, true, false, false, 202, false, 403, "rejected", []string{"session", "csrf", "authorize", "audit"}},
		{"HDG-H07-authority-unavailable", "/scenario/submit", "POST", valid, false, true, true, true, 202, false, 503, "error", []string{"session", "csrf", "authorize", "audit"}},
		{"HDG-H08-accepted-not-completed", "/scenario/submit", "POST", valid, false, true, true, false, 202, false, 202, "completed", []string{"session", "csrf", "authorize", "invoke", "audit"}},
		{"HDG-H09-rejected-by-handler", "/scenario/submit", "POST", valid, false, true, true, false, 422, false, 422, "rejected", []string{"session", "csrf", "authorize", "invoke", "audit"}},
		{"HDG-H10-panic-after-headers", "/scenario/submit", "POST", valid, false, true, true, false, 202, true, 202, "error", []string{"session", "csrf", "authorize", "invoke", "audit"}},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			calls := make([]string, 0)
			records := make([]AuditRecord, 0)
			h, err := NewHost(Policy{
				Path: "/scenario/submit", Method: http.MethodPost,
				ActionID:  "order_entry.submit_order",
				Providers: map[string]string{"corporate": valid.Issuer},
			}, Dependencies{
				VerifySession: func(*http.Request) (Principal, error) {
					calls = append(calls, "session")
					if tc.sessionError {
						return Principal{}, errors.New("synthetic session failure")
					}
					return tc.subject, nil
				},
				VerifyCSRF: func(*http.Request, Principal) bool {
					calls = append(calls, "csrf")
					return tc.csrf
				},
				Authorize: func(context.Context, Principal, string) (bool, error) {
					calls = append(calls, "authorize")
					if tc.authzError {
						return false, errors.New("synthetic authorization unavailable")
					}
					return tc.allowed, nil
				},
				Invoke: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					calls = append(calls, "invoke")
					w.WriteHeader(tc.invokeStatus)
					if tc.panicAfter {
						panic("synthetic panic after response began")
					}
				}),
				Audit: func(record AuditRecord) {
					calls = append(calls, "audit")
					records = append(records, record)
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(tc.method, "http://example.test"+tc.path, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.wantStatus || !reflect.DeepEqual(calls, tc.wantCalls) {
				t.Fatalf("status=%d want=%d, observed callbacks=%v want=%v",
					w.Code, tc.wantStatus, calls, tc.wantCalls)
			}
			if len(records) != 1 {
				t.Fatalf("expected one audit record for this reference request, got %d", len(records))
			}
			if got := records[0]; got.Status != tc.wantStatus || got.Stage != tc.wantStage ||
				got.Route != "POST /scenario/submit" || got.Action != "order_entry.submit_order" {
				t.Fatalf("audit=%+v, expected status=%d stage=%s", got, tc.wantStatus, tc.wantStage)
			}
		})
	}
}

// These two observed values must not be promoted into business-effect claims:
// - HTTP 202 with AuditRecord.Stage=completed is a finished Host invocation,
//   not evidence of a committed business operation.
// - A handler may write 202 and then panic; the client-visible status stays
//   202 while the Host records an error. Neither proves what the Action did.
