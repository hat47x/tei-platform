package webcomposition

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
)

// This Adapter is a test double. Invoke is called through the real Driver
// and the real Reference Workspace Binding, but never changes a database.
type hdgSyntheticSubmitAdapter func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)

func (f hdgSyntheticSubmitAdapter) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, capability, input)
}

// HDG-I01..I07: one protected HTTP attempt, real Workspace -> Driver
// Binding -> Adapter resolver -> synthetic Adapter dispatch. This verifies
// actual callback order, not durable commit, Host-Provider production trust,
// browser receipt, an E1 generated Host, or Graph closure.
func TestHDGRealWorkspaceAuthorizedAdapterDispatch(t *testing.T) {
	resolver, authorizationAdapters := referenceAuthorizationDependencies(t)
	const actionID = "order_entry.submit_order"
	const capabilityID = "sales.order.submit"
	const bindingID = "order_entry/sales.order.submit"

	resolved, err := resolver.ResolveCommandBinding("order_entry", actionID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Pattern.Capability == nil || resolved.Pattern.Capability.ID != capabilityID ||
		resolved.Binding.Binding.Identity() != bindingID {
		t.Fatalf("unexpected actual Canonical Binding: %+v", resolved)
	}

	editor := Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1",
		Subject: "employee-oidc-sub", PrincipalID: "employee-1",
	}
	viewer := Principal{
		ProviderID: editor.ProviderID, Issuer: editor.Issuer,
		Subject: "viewer-oidc-sub", PrincipalID: "employee-viewer-1",
	}
	authorizer, err := NewCanonicalActionAuthorizer(
		resolver, "order_entry", actionID,
		func(_ context.Context, origin ExternalSubject) (SubjectLink, bool, error) {
			switch origin {
			case (ExternalSubject{ProviderID: editor.ProviderID, Issuer: editor.Issuer, Subject: editor.Subject}):
				return SubjectLink{PrincipalID: editor.PrincipalID, SubjectID: "subject-order-editor"}, true, nil
			case (ExternalSubject{ProviderID: viewer.ProviderID, Issuer: viewer.Issuer, Subject: viewer.Subject}):
				return SubjectLink{PrincipalID: viewer.PrincipalID, SubjectID: "subject-order-viewer"}, true, nil
			default:
				return SubjectLink{}, false, nil
			}
		},
		authorizationAdapters,
	)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name          string
		principal     Principal
		sessionFailed bool
		csrfPassed    bool
		resolveFailed bool
		invokeFailed  bool
		wrongResult   bool
		wantHTTP      int
		wantStage     string
		wantCalls     []string
	}{
		{"HDG-I01-authorized-real-Driver-synthetic-Adapter", editor, false, true, false, false, false, 202, "completed", []string{"session", "csrf", "authorize", "adapter-resolve", "adapter-invoke", "audit"}},
		{"HDG-I02-viewer-denied-no-command-Adapter", viewer, false, true, false, false, false, 403, "rejected", []string{"session", "csrf", "authorize", "audit"}},
		{"HDG-I03-csrf-denied-no-authorization-or-Adapter", editor, false, false, false, false, false, 403, "rejected", []string{"session", "csrf", "audit"}},
		{"HDG-I04-Adapter-resolution-failed", editor, false, true, true, false, false, 503, "error", []string{"session", "csrf", "authorize", "adapter-resolve", "audit"}},
		{"HDG-I05-Adapter-invocation-failed", editor, false, true, false, true, false, 503, "error", []string{"session", "csrf", "authorize", "adapter-resolve", "adapter-invoke", "audit"}},
		{"HDG-I06-session-denied-no-Adapter", editor, true, true, false, false, false, 401, "rejected", []string{"session", "audit"}},
		{"HDG-I07-Canonical-result-invalid-not-business-denial", editor, false, true, false, false, true, 503, "error", []string{"session", "csrf", "authorize", "adapter-resolve", "adapter-invoke", "audit"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := []string{}
			audits := []AuditRecord{}
			probe := referenceBindingResolver(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
				calls = append(calls, "adapter-resolve")
				if binding.Identity() != bindingID || binding.Capability != capabilityID {
					return nil, errors.New("wrong selected binding")
				}
				if tc.resolveFailed {
					return nil, errors.New("synthetic resolver unavailable")
				}
				return hdgSyntheticSubmitAdapter(func(_ context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
					calls = append(calls, "adapter-invoke")
					if capability != capabilityID {
						t.Errorf("different Capability dispatched: %s", capability)
					}
					if tc.invokeFailed {
						return capabilityinvoke.BusinessResult{}, errors.New("synthetic adapter unavailable")
					}
					if tc.wrongResult {
						return capabilityinvoke.BusinessResult{
							Capability: capability, Mode: capabilityinvoke.ResultOutcome,
							Outcome: "not-declared", Payload: "synthetic",
						}, nil
					}
					values, ok := input.(map[string]any)
					if !ok || values["orderId"] != "synthetic-1001" {
						t.Errorf("unexpected test-only Adapter input: %#v", input)
					}
					return capabilityinvoke.BusinessResult{
						Capability: capability, Mode: capabilityinvoke.ResultOutput,
						Payload: map[string]any{"accepted": true},
					}, nil
				}), nil
			})
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				verified, ok := PrincipalFromContext(r.Context())
				if !ok || verified != tc.principal {
					t.Errorf("no exact Host verified Principal at Action: %+v", verified)
				}
				result, err := resolver.InvokeCapability(r.Context(), "order_entry", capabilityID, probe,
					map[string]any{"orderId": "synthetic-1001"})
				if err != nil {
					// Execution Error is not a business rejection or an
					// observation of an absent/persisted effect.
					http.Error(w, "command unavailable", http.StatusServiceUnavailable)
					return
				}
				if result.Binding.Binding.Identity() != bindingID ||
					result.Result.Capability != capabilityID ||
					result.Result.Mode != capabilityinvoke.ResultOutput {
					t.Errorf("Driver invocation lost Binding/Result contract: %+v", result)
				}
				w.WriteHeader(http.StatusAccepted)
			})
			host, err := NewHost(Policy{
				Path: "/scenario/submit", Method: http.MethodPost, ActionID: actionID,
				Providers: map[string]string{editor.ProviderID: editor.Issuer},
			}, Dependencies{
				VerifySession: func(*http.Request) (Principal, error) {
					calls = append(calls, "session")
					if tc.sessionFailed {
						return Principal{}, errors.New("synthetic expired session")
					}
					return tc.principal, nil
				},
				VerifyCSRF: func(*http.Request, Principal) bool {
					calls = append(calls, "csrf")
					return tc.csrfPassed
				},
				Authorize: func(ctx context.Context, subject Principal, action string) (bool, error) {
					calls = append(calls, "authorize")
					return authorizer(ctx, subject, action)
				},
				Invoke: handler,
				Audit: func(record AuditRecord) {
					calls = append(calls, "audit")
					audits = append(audits, record)
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			host.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "http://example.test/scenario/submit", nil))
			if response.Code != tc.wantHTTP || !reflect.DeepEqual(calls, tc.wantCalls) {
				t.Fatalf("HTTP=%d want=%d callbacks=%v want=%v",
					response.Code, tc.wantHTTP, calls, tc.wantCalls)
			}
			if len(audits) != 1 || audits[0].Route != "POST /scenario/submit" ||
				audits[0].Action != actionID || audits[0].Status != tc.wantHTTP ||
				audits[0].Stage != tc.wantStage {
				t.Fatalf("wrong Host audit for bounded attempt: %+v", audits)
			}
		})
	}
}
