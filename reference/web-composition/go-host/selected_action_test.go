package webcomposition

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tei/internal/composition"
)

func TestSelectedActionCannotBypassOrBorrowHostAuthorization(t *testing.T) {
	const actionID = "order_entry.submit_order"
	const version = "1.0.0"
	const providerID = "action-provider"
	digest := "sha256:" + strings.Repeat("c", 64)
	contract := composition.Contract{ID: actionID, Version: version, Shape: "reference-order-command"}
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: providerID, Scope: "reference/order_entry", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select(providerID, "reference/order_entry")
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{
		ContractID: actionID, ContractVersion: version,
		ProviderID: providerID, ArtifactDigest: digest, Generation: 1,
	}
	permission := true
	guardCalls, invokes := 0, 0
	coordinator, err := composition.NewSelectedCoordinator(
		registry, selection, binding, composition.NewScope(), func(b composition.Binding) error {
			guardCalls++
			if !permission || b != binding {
				return errors.New("private host permission denied")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	// The downstream could be a real external process. No untrusted request
	// may reach this callback without a matching Host approval.
	selected, err := NewSelectedActionInvoker(coordinator, actionID, version, []SelectedActionProvider{{
		ProviderID: providerID, ArtifactDigest: digest,
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			invokes++
			if _, ok := PrincipalFromContext(r.Context()); !ok {
				t.Error("selected callback lost Host principal")
			}
			w.WriteHeader(http.StatusAccepted)
		}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	principal := Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1",
		Subject: "external-1", PrincipalID: "employee-1",
	}
	sessionValid, csrfValid, actionAllowed := true, true, true
	var audit []AuditRecord
	authorize := func(_ context.Context, verified Principal, action string) (bool, error) {
		if verified != principal {
			return false, errors.New("wrong verified principal")
		}
		return actionAllowed, nil
	}
	newProtected := func(action string) http.Handler {
		t.Helper()
		h, err := NewHost(Policy{
			Path: "/reference/submit", Method: http.MethodPost, ActionID: action,
			Providers: map[string]string{"corporate": principal.Issuer},
		}, Dependencies{
			VerifySession: func(*http.Request) (Principal, error) {
				if !sessionValid {
					return Principal{}, errors.New("no session")
				}
				return principal, nil
			},
			VerifyCSRF: func(*http.Request, Principal) bool { return csrfValid },
			Authorize:  authorize,
			Invoke:     selected,
			Audit:      func(a AuditRecord) { audit = append(audit, a) },
		})
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	host := newProtected(actionID)
	run := func(h http.Handler) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/reference/submit",
			strings.NewReader(`{"principalId":"attacker","actionId":"order_entry.submit_order","providerId":"attacker"}`))
		req.Header.Set("X-Principal-ID", "attacker")
		req.Header.Set("X-Action-ID", actionID)
		req.Header.Set("X-CSRF-Checked", "true")
		response := httptest.NewRecorder()
		h.ServeHTTP(response, req)
		return response.Code
	}
	assertNoSelection := func(name string, h http.Handler, want int) {
		t.Helper()
		beforeGuard, beforeInvokes := guardCalls, invokes
		if status := run(h); status != want || guardCalls != beforeGuard || invokes != beforeInvokes {
			t.Fatalf("%s: status=%d expected=%d guards=%d->%d invokes=%d->%d",
				name, status, want, beforeGuard, guardCalls, beforeInvokes, invokes)
		}
	}
	assertNoSelection("bare handler", selected, http.StatusForbidden)
	assertNoSelection("different protected Action", newProtected("order_entry.cancel_order"), http.StatusForbidden)
	sessionValid = false
	assertNoSelection("session rejected", host, http.StatusUnauthorized)
	sessionValid, csrfValid = true, false
	assertNoSelection("CSRF rejected", host, http.StatusForbidden)
	csrfValid, actionAllowed = true, false
	assertNoSelection("business authorization rejected", host, http.StatusForbidden)
	actionAllowed = true
	before := guardCalls
	if status := run(host); status != http.StatusAccepted || guardCalls != before+1 || invokes != 1 {
		t.Fatalf("authorized Action did not select one provider: status=%d guards=%d invokes=%d",
			status, guardCalls-before, invokes)
	}
	permission = false
	beforeGuard, beforeInvokes := guardCalls, invokes
	if status := run(host); status != http.StatusServiceUnavailable ||
		guardCalls != beforeGuard+1 || invokes != beforeInvokes {
		t.Fatalf("revoked host authority bypass: status=%d guards=%d invokes=%d",
			status, guardCalls-beforeGuard, invokes-beforeInvokes)
	}
	permission = true
	if _, err := registry.Apply(snapshot.Revision, nil, []string{providerID}); err != nil {
		t.Fatal(err)
	}
	beforeInvokes := invokes
	if status := run(host); status != http.StatusServiceUnavailable || invokes != beforeInvokes {
		t.Fatalf("withdrawn provider dispatched: status=%d invokes=%d", status, invokes)
	}
	if len(audit) == 0 || audit[len(audit)-1].Stage != "error" {
		t.Fatalf("selected dispatch denial is not audited: %+v", audit)
	}
}

func TestSelectedActionRequiresBothPrivateHostProofs(t *testing.T) {
	contract := composition.Contract{ID: "order_entry.submit_order", Version: "1.0.0", Shape: "order-v1"}
	registry := composition.NewRegistry()
	_, err := registry.Apply(0, []composition.Provider{{ID: "p", Scope: "test", Offers: contract}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("p", "test")
	if err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("1", 64)
	binding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: "p", ArtifactDigest: digest, Generation: 1,
	}
	acquires, invoked := 0, 0
	coordinator, err := composition.NewSelectedCoordinator(registry, selection, binding, composition.NewScope(), func(composition.Binding) error {
		acquires++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := NewSelectedActionInvoker(coordinator, contract.ID, contract.Version, []SelectedActionProvider{{
		ProviderID: "p", ArtifactDigest: digest,
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			invoked++
			w.WriteHeader(http.StatusAccepted)
		}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	principal := Principal{ProviderID: "idp", Issuer: "issuer", Subject: "subject", PrincipalID: "user"}
	for _, scenario := range []struct {
		name         string
		addPrincipal bool
		addApproval  bool
	}{
		{"no private proof", false, false},
		{"principal without Action approval", true, false},
		{"Action approval without principal", false, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/reference/submit", nil)
			ctx := req.Context()
			if scenario.addPrincipal {
				ctx = context.WithValue(ctx, principalContextKey{}, principal)
			}
			if scenario.addApproval {
				ctx = context.WithValue(ctx, authorizedActionContextKey{}, contract.ID)
			}
			w := httptest.NewRecorder()
			selected.ServeHTTP(w, req.WithContext(ctx))
			if w.Code != http.StatusForbidden || acquires != 1 || invoked != 0 {
				t.Fatalf("partial Host proof allowed selection: status=%d acquired=%d invoked=%d", w.Code, acquires, invoked)
			}
		})
	}
}
