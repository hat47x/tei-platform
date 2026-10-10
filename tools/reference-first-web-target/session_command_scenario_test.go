package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"tei/internal/realization"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
	webcomposition "tei/reference/web-composition/go-host"
)

func sessionReferenceRunner(t *testing.T) *referenceCommandScenarioRunner {
	t.Helper()
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(loaded.World,
		legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	return runner
}

func sessionReferenceAdapters(runner *referenceCommandScenarioRunner) bindingAdapterResolverFunc {
	authorization := legacyauthorization.Adapter{Store: runner.store, Rule: runner.authorization}
	return bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		if binding.Identity() != "order_entry/security.authorize" {
			return nil, errors.New("unexpected authorization Binding")
		}
		return authorization, nil
	})
}

func TestSessionCommandUsesExistingBusinessExecutionWithoutCredentials(t *testing.T) {
	runner := sessionReferenceRunner(t)
	corporate := webcomposition.Principal{ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1", Subject: "oidc-editor", PrincipalID: "employee-1"}
	legacy := webcomposition.Principal{ProviderID: "legacy", Issuer: "legacy:order_entry", Subject: "legacy-editor", PrincipalID: "employee-1"}
	current := corporate
	linkCalls := 0
	links := webcomposition.SubjectLinkLookup(func(_ context.Context, origin webcomposition.ExternalSubject) (webcomposition.SubjectLink, bool, error) {
		linkCalls++
		if origin.ProviderID == current.ProviderID && origin.Issuer == current.Issuer && origin.Subject == current.Subject {
			return webcomposition.SubjectLink{PrincipalID: current.PrincipalID, SubjectID: "subject-order-editor"}, true, nil
		}
		return webcomposition.SubjectLink{}, false, nil
	})
	authorize, err := webcomposition.NewCanonicalActionAuthorizer(
		runner.resolver, "order_entry", "order_entry.submit_order", links, sessionReferenceAdapters(runner))
	if err != nil {
		t.Fatal(err)
	}
	var actual commandScenarioResult
	var actualErr error
	host, err := webcomposition.NewHost(webcomposition.Policy{
		Path: "/scenario/session-submit", Method: http.MethodPost, ActionID: "order_entry.submit_order",
		Providers: map[string]string{"corporate": corporate.Issuer, "legacy": legacy.Issuer},
	}, webcomposition.Dependencies{
		VerifySession: func(*http.Request) (webcomposition.Principal, error) { return current, nil },
		VerifyCSRF: func(r *http.Request, _ webcomposition.Principal) bool {
			return r.Header.Get("X-CSRF-Fixture") == "matching"
		},
		Authorize: authorize,
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actual, actualErr = runner.RunVerifiedSessionCommand(r.Context(), links, "2026-09-18T08:30:00Z", 0)
			if actualErr != nil {
				http.Error(w, "command failed", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		}),
		Audit: func(webcomposition.AuditRecord) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, identity := range []webcomposition.Principal{corporate, legacy} {
		current = identity
		req := httptest.NewRequest(http.MethodPost, "/scenario/session-submit", nil)
		req.Header.Set("X-CSRF-Fixture", "matching")
		recorder := httptest.NewRecorder()
		host.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusAccepted || actualErr != nil || actual.Stage != "completed" {
			t.Fatalf("%s: code=%d stage=%s err=%v", identity.ProviderID, recorder.Code, actual.Stage, actualErr)
		}
		if actual.Authentication != nil {
			t.Fatalf("session authentication must not be presented as legacy credential authentication: %+v", actual.Authentication)
		}
		if actual.Authorization == nil || actual.Authorization.Capability != "security.authorize" ||
			actual.Authorization.Outcome != "allowed" || actual.Command == nil || actual.Observation == nil ||
			len(actual.Validations) != 3 {
			t.Fatalf("%s lost business stages: %+v", identity.ProviderID, actual)
		}
	}
	if linkCalls != 4 {
		t.Fatalf("expected fresh link lookup at HTTP and Command boundaries, got %d", linkCalls)
	}
}

func TestSessionCommandStillAuthorizesAtBusinessBoundary(t *testing.T) {
	runner := sessionReferenceRunner(t)
	cases := []struct {
		name       string
		subjectID  string
		principal  string
		linkExists bool
		linkError  bool
		wantStage  string
		wantError  bool
	}{
		{name: "viewer denied by canonical rule", subjectID: "subject-order-viewer", principal: "viewer", linkExists: true, wantStage: "authorization-denied"},
		{name: "unlinked identity denied", principal: "unknown", wantStage: "identity-link-denied"},
		{name: "forged PrincipalID fails", subjectID: "subject-order-editor", principal: "forged", linkExists: true, wantStage: "identity-link", wantError: true},
		{name: "link backend unavailable", principal: "missing", linkError: true, wantStage: "identity-link", wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			principal := webcomposition.Principal{ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1", Subject: "fixed-sub", PrincipalID: tc.principal}
			links := webcomposition.SubjectLinkLookup(func(context.Context, webcomposition.ExternalSubject) (webcomposition.SubjectLink, bool, error) {
				if tc.linkError {
					return webcomposition.SubjectLink{}, false, errors.New("link store down")
				}
				return webcomposition.SubjectLink{PrincipalID: "viewer", SubjectID: tc.subjectID}, tc.linkExists, nil
			})
			invocations := 0
			var result commandScenarioResult
			var runErr error
			host, err := webcomposition.NewHost(webcomposition.Policy{
				Path: "/scenario/session-submit", Method: http.MethodPost, ActionID: "order_entry.submit_order",
				Providers: map[string]string{"corporate": principal.Issuer},
			}, webcomposition.Dependencies{
				VerifySession: func(*http.Request) (webcomposition.Principal, error) { return principal, nil },
				VerifyCSRF:    func(*http.Request, webcomposition.Principal) bool { return true },
				// Deliberately permissive HTTP hook: the Runner must not treat it as
				// permission to skip the canonical business authorization.
				Authorize: func(context.Context, webcomposition.Principal, string) (bool, error) { return true, nil },
				Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					invocations++
					result, runErr = runner.RunVerifiedSessionCommand(r.Context(), links, "2026-09-18T08:30:00Z", 0)
					w.WriteHeader(http.StatusAccepted)
				}),
				Audit: func(webcomposition.AuditRecord) {},
			})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/scenario/session-submit", nil)
			rec := httptest.NewRecorder()
			host.ServeHTTP(rec, req)
			if rec.Code != http.StatusAccepted || invocations != 1 || result.Stage != tc.wantStage || (runErr != nil) != tc.wantError {
				t.Fatalf("HTTP=%d calls=%d stage=%s err=%v", rec.Code, invocations, result.Stage, runErr)
			}
			if result.Command != nil || result.Observation != nil {
				t.Fatal("untrusted or unauthorized identity reached a command")
			}
			if tc.wantStage == "authorization-denied" && (result.Authorization == nil || result.Authorization.Outcome != "denied") {
				t.Fatalf("canonical business denial lost: %+v", result.Authorization)
			}
		})
	}
	if _, err := runner.RunVerifiedSessionCommand(context.Background(),
		func(context.Context, webcomposition.ExternalSubject) (webcomposition.SubjectLink, bool, error) {
			return webcomposition.SubjectLink{PrincipalID: "someone", SubjectID: "subject-order-editor"}, true, nil
		}, "2026-09-18T08:30:00Z", 0); err == nil {
		t.Fatal("plain context bypassed protected HTTP Host")
	}
}
