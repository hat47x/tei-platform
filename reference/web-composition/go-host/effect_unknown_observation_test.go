package webcomposition

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
)

// A deliberately isolated, synthetic store: an Adapter can change its value
// before returning an error, and an HTTP acknowledgement need not mutate it.
// This is not a durable database, receipt, retry strategy or production grant.
type hdgEffectFixture struct {
	status string
}

// HDG-E01..E05: the SAME HTTP 503 may follow an applied or an unapplied
// synthetic effect. Conversely HTTP 202 need not mean the effect was applied.
// Only a separate, explicitly invoked read determines this fixture's state.
func TestHDGHttpResultDoesNotSubstituteForIndependentEffectObservation(t *testing.T) {
	resolver, authorizationAdapters := referenceAuthorizationDependencies(t)
	const (
		actionID     = "order_entry.submit_order"
		capabilityID = "sales.order.submit"
		bindingID    = "order_entry/sales.order.submit"
	)
	canonical, err := resolver.ResolveCommandBinding("order_entry", actionID)
	if err != nil || canonical.Binding.Binding.Identity() != bindingID ||
		canonical.Pattern.Capability == nil || canonical.Pattern.Capability.ID != capabilityID {
		t.Fatalf("actual Canonical command Binding unavailable: %+v / %v", canonical, err)
	}
	editor := Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1",
		Subject: "employee-oidc-sub", PrincipalID: "employee-1",
	}
	authorizer, err := NewCanonicalActionAuthorizer(resolver, "order_entry", actionID,
		func(_ context.Context, origin ExternalSubject) (SubjectLink, bool, error) {
			if origin != (ExternalSubject{ProviderID: editor.ProviderID, Issuer: editor.Issuer, Subject: editor.Subject}) {
				return SubjectLink{}, false, nil
			}
			return SubjectLink{PrincipalID: editor.PrincipalID, SubjectID: "subject-order-editor"}, true, nil
		}, authorizationAdapters)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name             string
		mutateBeforeExit bool
		invokeError      bool
		observationError bool
		wantHTTP         int
		wantStage        string
		wantObserved     string
	}{
		{"HDG-E01-503-after-effect-then-explicit-read-shows-applied", true, true, false, 503, "error", "submitted"},
		{"HDG-E02-202-ack-before-effect-explicit-read-shows-pending", false, false, false, 202, "completed", "pending"},
		{"HDG-E03-202-ack-after-effect-explicit-read-shows-applied", true, false, false, 202, "completed", "submitted"},
		{"HDG-E04-503-before-effect-explicit-read-shows-pending", false, true, false, 503, "error", "pending"},
		{"HDG-E05-503-after-effect-and-read-failure-remains-unknown", true, true, true, 503, "error", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// This exact test-only operation reference is passed through both
			// boundaries; its string does not establish a product receipt.
			operationRef := "hdg-reference-order-1001"
			store := &hdgEffectFixture{status: "pending"}
			adapterResolveCalls, adapterInvokeCalls := 0, 0
			readCalls, actionCalls := 0, 0
			var audit []AuditRecord
			adapters := referenceBindingResolver(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
				adapterResolveCalls++
				if binding.Identity() != bindingID {
					return nil, fmt.Errorf("unexpected Binding %s", binding.Identity())
				}
				return hdgSyntheticSubmitAdapter(func(_ context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
					adapterInvokeCalls++
					values, ok := input.(map[string]string)
					if capability != capabilityID || !ok || values["operationRef"] != operationRef {
						t.Errorf("Adapter received wrong operation/capability: %s %#v", capability, input)
					}
					if tc.mutateBeforeExit {
						store.status = "submitted"
					}
					if tc.invokeError {
						return capabilityinvoke.BusinessResult{}, errors.New("synthetic response lost after possible state change")
					}
					return capabilityinvoke.BusinessResult{
						Capability: capability, Mode: capabilityinvoke.ResultOutput,
						Payload: map[string]any{"accepted": true},
					}, nil
				}), nil
			})
			action := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				actionCalls++
				if principal, ok := PrincipalFromContext(r.Context()); !ok || principal != editor {
					t.Error("protected Action lost the Host's verified principal")
				}
				result, err := resolver.InvokeCapability(r.Context(), "order_entry", capabilityID, adapters,
					map[string]string{"operationRef": operationRef})
				if err != nil {
					http.Error(w, "command unavailable", http.StatusServiceUnavailable)
					return
				}
				if result.Binding.Binding.Identity() != bindingID ||
					result.Result.Capability != capabilityID ||
					result.Result.Mode != capabilityinvoke.ResultOutput {
					t.Errorf("result contract differs from selected Canonical command: %+v", result)
				}
				w.WriteHeader(http.StatusAccepted)
			})
			host, err := NewHost(Policy{
				Path: "/scenario/submit", Method: http.MethodPost, ActionID: actionID,
				Providers: map[string]string{editor.ProviderID: editor.Issuer},
			}, Dependencies{
				VerifySession: func(*http.Request) (Principal, error) { return editor, nil }, // synthetic verifier
				VerifyCSRF:    func(*http.Request, Principal) bool { return true },           // synthetic verifier
				Authorize:     authorizer, Invoke: action,
				Audit: func(record AuditRecord) { audit = append(audit, record) },
			})
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			host.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "http://example.test/scenario/submit", nil))
			if response.Code != tc.wantHTTP || len(audit) != 1 ||
				audit[0].Status != tc.wantHTTP || audit[0].Stage != tc.wantStage ||
				audit[0].Action != actionID || audit[0].Route != "POST /scenario/submit" {
				t.Fatalf("HTTP/audit differed: HTTP=%d audit=%+v expected=%d/%s", response.Code, audit, tc.wantHTTP, tc.wantStage)
			}
			if actionCalls != 1 || adapterResolveCalls != 1 || adapterInvokeCalls != 1 || readCalls != 0 {
				t.Fatalf("HTTP command must not implicitly read or retry: action=%d resolve=%d invoke=%d read=%d",
					actionCalls, adapterResolveCalls, adapterInvokeCalls, readCalls)
			}
			// No API response or Host Audit value provides an ObservationRead
			// here. In particular, 202 cannot assert "submitted" and 503
			// cannot assert "pending", despite the test owning the store.
			observation, readErr := driver.ExecuteObservationRead(context.Background(),
				"synthetic.order_state.read/"+operationRef, driver.ObservationEvidenceSynthetic,
				func(context.Context) (any, error) {
					readCalls++
					if tc.observationError {
						return nil, errors.New("synthetic state reader unavailable")
					}
					return store.status, nil
				})
			if readCalls != 1 || adapterInvokeCalls != 1 {
				t.Fatalf("explicit read performed hidden retry/read: read=%d invoke=%d", readCalls, adapterInvokeCalls)
			}
			if observation.ReadIdentity != "synthetic.order_state.read/"+operationRef ||
				observation.Nature != driver.ObservationEvidenceSynthetic {
				t.Fatalf("read evidence lost the exact synthetic identity and nature: %+v", observation)
			}
			if tc.observationError {
				if readErr == nil || observation.Result != nil {
					t.Fatalf("unavailable read fabricated effect disposition: %+v / %v", observation, readErr)
				}
				return
			}
			if readErr != nil || observation.Result != tc.wantObserved {
				t.Fatalf("independent fixture read=%+v / %v; want=%s", observation, readErr, tc.wantObserved)
			}
		})
	}
}
