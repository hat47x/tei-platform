package webcomposition

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/servicehost"
)

type twoApplicationHostCall struct {
	Operation string
	Customer  int64
}

type twoApplicationHostProcess struct {
	calls []twoApplicationHostCall
}

func (p *twoApplicationHostProcess) Invoke(
	_ context.Context, _ pruntime.ProcessCommand, request pruntime.Request,
) (pruntime.Response, error) {
	var input struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(request.Payload, &input); err != nil {
		return pruntime.Response{}, err
	}
	p.calls = append(p.calls, twoApplicationHostCall{
		Operation: request.Operation, Customer: input.ID,
	})
	return pruntime.Response{
		Status: pruntime.ResponseOK, Payload: json.RawMessage(`{"ok":true}`),
	}, nil
}

// Both protected Host entry points intentionally share one physical Provider,
// Artifact digest and test ProcessLauncher. Their Application-scoped Composition
// selectors, Host guards, route sources and service bindings remain independent.
// Session, authorization, acceptance, verification and process are test doubles.
func TestTwoProtectedApplicationsDoNotBorrowHostServiceBindings(t *testing.T) {
	const contractID = "test-only.customer.find_by_id"
	contract := composition.Contract{
		ID: contractID, Version: "1.0.0", Shape: "test-only-customer-id",
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	physical := plugin.Identity{ID: "tei.test.shared-customer", Version: "1.0.0"}
	capabilityID := core.CanonicalID("reference.customer.query")
	selection := pruntime.ServiceProviderSelection{
		ContractID: contractID, ContractVersion: contract.Version,
		ContractFingerprint: "sha256:" + strings.Repeat("b", 64),
		Scope:               "application-runtime", Provider: physical,
		ImplementationID: "test-only-customer-adapter", ArtifactDigest: digest,
	}
	process := &twoApplicationHostProcess{}
	verifier := &serviceHandoffVerifier{}
	accepted := &serviceHandoffAcceptance{}
	runtimeHost := pruntime.Registry{
		Processes: process, Verifier: verifier, ServiceAcceptance: accepted,
	}

	guards := map[string]bool{"order_entry": true, "repair_intake": true}
	routePresent := map[string]bool{"order_entry": true, "repair_intake": true}
	routeCalls := map[string]int{}
	routes := map[string]*servicehost.SelectedServiceCarrierRoute{}
	editor := Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1",
		Subject: "order-operator", PrincipalID: "operator-order",
	}
	repairer := Principal{
		ProviderID: editor.ProviderID, Issuer: editor.Issuer,
		Subject: "repair-operator", PrincipalID: "operator-repair",
	}
	currentPrincipal := editor

	makeHost := func(app, action, path, subject string, customerID int64) http.Handler {
		t.Helper()
		scope := "reference/" + app
		logicalProvider := "logical-" + app
		registry := composition.NewRegistry()
		if _, err := registry.Apply(0, []composition.Provider{{
			ID: logicalProvider, Scope: scope, Offers: contract,
		}}, nil); err != nil {
			t.Fatal(err)
		}
		selected, err := registry.Select(logicalProvider, scope)
		if err != nil {
			t.Fatal(err)
		}
		binding := composition.Binding{
			ContractID: contractID, ContractVersion: contract.Version,
			ProviderID: selected.ProviderID, ArtifactDigest: digest, Generation: 1,
		}
		coordinator, err := composition.NewSelectedCoordinator(
			registry, selected, binding, composition.NewScope(),
			func(actual composition.Binding) error {
				if !guards[app] || actual != binding {
					return errors.New("test Application Host guard denied")
				}
				return nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		plan := pruntime.EnvironmentPlan{
			Environment: "development", PluginLockFingerprint: "test-only-lock",
			Plugins: []pruntime.PluginRuntime{{
				Plugin: physical, DescriptorFingerprint: "test-only-descriptor",
				Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
				Command: "/test-only/customer", SandboxPolicy: plugin.SandboxNone,
				Artifact: plugin.Artifact{Digest: digest},
			}},
			Capabilities: []pruntime.CapabilityRuntime{{
				Plugin: physical,
				Capability: plugin.Capability{
					Kind: plugin.CapabilityContractQueryExecutor, ID: capabilityID,
				},
				Boundary: pruntime.BoundaryProcess,
			}},
			ServiceBindings: []pruntime.ServiceCarrierBinding{{
				ServiceProviderSelection: selection,
				CarrierKind:              plugin.CapabilityContractQueryExecutor,
				CarrierID:                capabilityID, Operation: app + ".find_by_id",
			}},
		}
		plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		routes[app] = &servicehost.SelectedServiceCarrierRoute{
			Binding: binding, Selection: selection, Plan: plan,
		}
		invoker, err := servicehost.NewSelectedServiceCarrierInvoker(
			coordinator, runtimeHost,
			func(_ context.Context, pinned composition.Binding) (servicehost.SelectedServiceCarrierRoute, error) {
				routeCalls[app]++
				if !routePresent[app] {
					return servicehost.SelectedServiceCarrierRoute{}, errors.New("private Host route unavailable")
				}
				route := *routes[app]
				if route.Binding != pinned {
					return servicehost.SelectedServiceCarrierRoute{}, errors.New("unexpected pinned Application generation")
				}
				return route, nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(struct {
			ID int64 `json:"id"`
		}{ID: customerID})
		if err != nil {
			t.Fatal(err)
		}
		host, err := NewHost(Policy{
			Path: path, Method: http.MethodPost, ActionID: action,
			Providers: map[string]string{"corporate": editor.Issuer},
		}, Dependencies{
			VerifySession: func(*http.Request) (Principal, error) {
				return currentPrincipal, nil
			},
			VerifyCSRF: func(*http.Request, Principal) bool { return true },
			Authorize: func(_ context.Context, principal Principal, currentAction string) (bool, error) {
				return currentAction == action && principal.Subject == subject, nil
			},
			Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if principal, ok := PrincipalFromContext(r.Context()); !ok || principal != currentPrincipal {
					t.Error("protected Host Principal was not preserved")
				}
				result, err := invoker.Invoke(r.Context(), "request-"+app, payload)
				if err != nil {
					http.Error(w, "service unavailable", http.StatusServiceUnavailable)
					return
				}
				if result.Generation != binding ||
					result.Service.Selection != selection ||
					result.Service.Runtime.Provenance.Environment != "development" ||
					result.Service.Runtime.Provenance.Plugin.ID != physical.ID ||
					result.Service.Runtime.Provenance.Artifact.Digest != digest {
					t.Error("Host-selected Service generation/provenance differs")
				}
				w.WriteHeader(http.StatusAccepted)
			}),
			Audit: func(AuditRecord) {},
		})
		if err != nil {
			t.Fatal(err)
		}
		return host
	}

	orderHost := makeHost("order_entry", "order_entry.test_customer_lookup", "/order/customer", editor.Subject, 42)
	repairHost := makeHost("repair_intake", "repair_intake.test_customer_lookup", "/repair/customer", repairer.Subject, 84)
	call := func(host http.Handler, path string, principal Principal) int {
		t.Helper()
		currentPrincipal = principal
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(
			`{"applicationId":"other_application","environment":"production","provider":"forged","generation":999}`))
		req.Header.Set("X-Application-ID", "other_application")
		req.Header.Set("X-Environment", "production")
		req.Header.Set("X-Generation", "999")
		recorder := httptest.NewRecorder()
		host.ServeHTTP(recorder, req)
		return recorder.Code
	}
	check := func(expectedCalls []twoApplicationHostCall, acceptedCount, verifiedCount, orderRoutes, repairRoutes int) {
		t.Helper()
		if !slices.Equal(process.calls, expectedCalls) ||
			accepted.calls != acceptedCount || verifier.calls != verifiedCount ||
			routeCalls["order_entry"] != orderRoutes || routeCalls["repair_intake"] != repairRoutes {
			t.Fatalf("Host boundary mismatch: calls=%+v acceptance=%d verify=%d routes=%v",
				process.calls, accepted.calls, verifier.calls, routeCalls)
		}
	}
	expected := []twoApplicationHostCall{}
	if status := call(repairHost, "/repair/customer", editor); status != http.StatusForbidden {
		t.Fatalf("order Principal borrowed repair Host: %d", status)
	}
	if status := call(orderHost, "/order/customer", repairer); status != http.StatusForbidden {
		t.Fatalf("repair Principal borrowed order Host: %d", status)
	}
	check(expected, 0, 0, 0, 0)

	if status := call(orderHost, "/order/customer", editor); status != http.StatusAccepted {
		t.Fatalf("authorized order Host: %d", status)
	}
	if status := call(repairHost, "/repair/customer", repairer); status != http.StatusAccepted {
		t.Fatalf("authorized repair Host: %d", status)
	}
	expected = append(expected,
		twoApplicationHostCall{Operation: "order_entry.find_by_id", Customer: 42},
		twoApplicationHostCall{Operation: "repair_intake.find_by_id", Customer: 84},
	)
	check(expected, 2, 2, 1, 1)

	// Withdrawing one Host's Composition guard must not invalidate the other
	// Host's pinned generation even when both use the same physical Provider.
	guards["order_entry"] = false
	if status := call(orderHost, "/order/customer", editor); status != http.StatusServiceUnavailable {
		t.Fatalf("withdrawn order Host guard: %d", status)
	}
	if status := call(repairHost, "/repair/customer", repairer); status != http.StatusAccepted {
		t.Fatalf("repair Host lost a separately admitted generation: %d", status)
	}
	expected = append(expected, twoApplicationHostCall{Operation: "repair_intake.find_by_id", Customer: 84})
	check(expected, 3, 3, 1, 2)
	guards["order_entry"] = true

	// Remove ONLY the order Host's Service binding, keeping its Composition
	// generation installed and the shared physical Provider still accepted.
	order := routes["order_entry"]
	savedBindings := order.Plan.ServiceBindings
	order.Plan.ServiceBindings = nil
	order.Plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(order.Plan)
	if status := call(orderHost, "/order/customer", editor); status != http.StatusServiceUnavailable {
		t.Fatalf("unbound order Host was allowed: %d", status)
	}
	if status := call(repairHost, "/repair/customer", repairer); status != http.StatusAccepted {
		t.Fatalf("repair Host borrowed order's revoked binding: %d", status)
	}
	expected = append(expected, twoApplicationHostCall{Operation: "repair_intake.find_by_id", Customer: 84})
	check(expected, 4, 4, 2, 3)
	order.Plan.ServiceBindings = savedBindings
	order.Plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(order.Plan)

	// Shared Service acceptance is a DIFFERENT decision from either Host's
	// per-Application guard or binding. Revocation denies both next invocations.
	accepted.revoked = true
	for _, tc := range []struct {
		host http.Handler
		path string
		principal Principal
	}{
		{orderHost, "/order/customer", editor},
		{repairHost, "/repair/customer", repairer},
	} {
		if status := call(tc.host, tc.path, tc.principal); status != http.StatusServiceUnavailable {
			t.Fatalf("revoked shared Provider was allowed: %d", status)
		}
	}
	check(expected, 6, 4, 3, 4)
}
