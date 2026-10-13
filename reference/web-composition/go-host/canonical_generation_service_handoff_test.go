package webcomposition

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/servicehost"
)

// This is a synthetic single-Host wiring test, NOT a production Plan selector.
// No request field or header is permitted to select the Host's Application,
// Environment, Generation, Provider, or artifact route.
func TestCanonicalQuoteActionUsesHostOwnedGenerationServiceRoute(t *testing.T) {
	const actionID = "order_entry.preview_order_quote"
	const serviceID = "sales.order.quote_lines"
	const appScope = "reference/order_entry"
	digest := "sha256:" + strings.Repeat("a", 64)
	contract := composition.Contract{ID: serviceID, Version: "1.0.0", Shape: "test-only-quote-shape"}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{ID: "order-quote-logical", Scope: appScope, Offers: contract}}, nil); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("order-quote-logical", appScope)
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{
		ContractID: serviceID, ContractVersion: contract.Version,
		ProviderID: selected.ProviderID, ArtifactDigest: digest, Generation: 1,
	}
	hostGuardAllowed := true
	coordinator, err := composition.NewSelectedCoordinator(registry, selected, binding, composition.NewScope(), func(current composition.Binding) error {
		if !hostGuardAllowed || current != binding {
			return errors.New("test Host application route withdrawn")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	physical := plugin.Identity{ID: "tei.reference.quote", Version: "1.0.0"}
	carrierID := core.CanonicalID("reference.quote.executor")
	selection := pruntime.ServiceProviderSelection{
		ContractID: serviceID, ContractVersion: contract.Version,
		ContractFingerprint: "sha256:" + strings.Repeat("b", 64),
		Scope:               "application-runtime", Provider: physical,
		ImplementationID: "test-quote-bridge", ArtifactDigest: digest,
	}
	plan := pruntime.EnvironmentPlan{
		Environment: "development", PluginLockFingerprint: "test-only-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: physical, DescriptorFingerprint: "test-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: "/test-only/quote", SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin:     physical,
			Capability: plugin.Capability{Kind: plugin.CapabilityContractQueryExecutor, ID: carrierID},
			Boundary:   pruntime.BoundaryProcess,
		}},
		ServiceBindings: []pruntime.ServiceCarrierBinding{{
			ServiceProviderSelection: selection,
			CarrierKind:              plugin.CapabilityContractQueryExecutor, CarrierID: carrierID, Operation: "quote",
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	process := &serviceHandoffProcess{}
	verifier := &serviceHandoffVerifier{}
	accepted := &serviceHandoffAcceptance{}
	runtimeHost := pruntime.Registry{Processes: process, Verifier: verifier, ServiceAcceptance: accepted}

	routeCalls := 0
	routeUnavailable, wrongGeneration := false, false
	invoker, err := servicehost.NewSelectedServiceCarrierInvoker(
		coordinator, runtimeHost,
		func(_ context.Context, pinned composition.Binding) (servicehost.SelectedServiceCarrierRoute, error) {
			routeCalls++
			if routeUnavailable {
				return servicehost.SelectedServiceCarrierRoute{}, errors.New("private Host route unavailable")
			}
			route := servicehost.SelectedServiceCarrierRoute{Binding: binding, Selection: selection, Plan: plan}
			if wrongGeneration {
				route.Binding.Generation++
			}
			// Route is Host-owned; pinned is provided only by the Coordinator,
			// not by the HTTP request (which advertises a forged application).
			if pinned != binding {
				return servicehost.SelectedServiceCarrierRoute{}, errors.New("unrecognized Host binding")
			}
			return route, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	resolver, adapters := quoteCandidateAuthorizationDependencies(t)
	resolved, err := resolver.ResolveCommandBinding("order_entry", actionID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Pattern.Capability == nil || string(resolved.Pattern.Capability.ID) != serviceID || resolved.Pattern.Capability.Effect != nil {
		t.Fatal("Web Host Action does not resolve to the expected read-only quote Capability")
	}
	editor := Principal{ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1", Subject: "corporate-editor", PrincipalID: "employee-1"}
	viewer := Principal{ProviderID: "corporate", Issuer: editor.Issuer, Subject: "corporate-viewer", PrincipalID: "employee-2"}
	currentPrincipal := editor
	authorizer, err := NewCanonicalActionAuthorizer(resolver, "order_entry", actionID,
		func(_ context.Context, identity ExternalSubject) (SubjectLink, bool, error) {
			if identity != (ExternalSubject{ProviderID: editor.ProviderID, Issuer: editor.Issuer, Subject: editor.Subject}) {
				return SubjectLink{}, false, nil
			}
			return SubjectLink{PrincipalID: editor.PrincipalID, SubjectID: "subject-order-editor"}, true, nil
		}, adapters)
	if err != nil {
		t.Fatal(err)
	}

	// The callback is registered ONLY as NewHost.Invoke. It is not published as
	// a standalone route; the Host authorizes the exact Action before calling it.
	host, err := NewHost(Policy{
		Path: "/host/quote", Method: http.MethodPost, ActionID: actionID,
		Providers: map[string]string{"corporate": editor.Issuer},
	}, Dependencies{
		VerifySession: func(*http.Request) (Principal, error) { return currentPrincipal, nil },
		VerifyCSRF:    func(*http.Request, Principal) bool { return true },
		Authorize:     authorizer,
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if principal, ok := PrincipalFromContext(r.Context()); !ok || principal != currentPrincipal {
				t.Error("untrusted Principal reached Service invocation")
			}
			response, invokeErr := invoker.Invoke(r.Context(), "host-quote-request", json.RawMessage(`{"quantity":2}`))
			if invokeErr != nil {
				http.Error(w, "service unavailable", http.StatusServiceUnavailable)
				return
			}
			if response.Generation != binding || response.Service.Selection != selection ||
				response.Service.Runtime.Provenance.Artifact.Digest != digest ||
				response.Service.Runtime.Provenance.Plugin.ID != physical.ID {
				t.Error("Host generation or physical Service provenance changed")
			}
			w.WriteHeader(http.StatusAccepted)
		}),
		Audit: func(AuditRecord) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	call := func() int {
		t.Helper()
		// Every message deliberately lies about Application, Environment,
		// Provider, and generation; none is an input to Host route selection.
		req := httptest.NewRequest(http.MethodPost, "/host/quote", strings.NewReader(`{"applicationId":"repair_intake","environment":"production","provider":"other","generation":999}`))
		req.Header.Set("X-Application-ID", "repair_intake")
		req.Header.Set("X-Environment", "production")
		req.Header.Set("X-Generation", "999")
		result := httptest.NewRecorder()
		host.ServeHTTP(result, req)
		return result.Code
	}
	assertCounts := func(routes, acceptance, verify, child int) {
		t.Helper()
		if routeCalls != routes || accepted.calls != acceptance || verifier.calls != verify || process.calls != child {
			t.Fatalf("unexpected Host execution: routes=%d accepted=%d verified=%d processes=%d; want %d/%d/%d/%d",
				routeCalls, accepted.calls, verifier.calls, process.calls, routes, acceptance, verify, child)
		}
	}
	currentPrincipal = viewer
	if status := call(); status != http.StatusForbidden {
		t.Fatalf("unauthorized viewer status %d", status)
	}
	assertCounts(0, 0, 0, 0)
	currentPrincipal = editor
	if status := call(); status != http.StatusAccepted {
		t.Fatalf("authorized Action status %d", status)
	}
	assertCounts(1, 1, 1, 1)

	wrongGeneration = true
	if status := call(); status != http.StatusServiceUnavailable {
		t.Fatalf("mismatched Host route status %d", status)
	}
	assertCounts(2, 1, 1, 1)
	wrongGeneration = false
	routeUnavailable = true
	if status := call(); status != http.StatusServiceUnavailable {
		t.Fatalf("absent Host route status %d", status)
	}
	assertCounts(3, 1, 1, 1)
	routeUnavailable = false
	hostGuardAllowed = false
	if status := call(); status != http.StatusServiceUnavailable {
		t.Fatalf("withdrawn Application guard status %d", status)
	}
	assertCounts(3, 1, 1, 1)
	hostGuardAllowed = true
	accepted.revoked = true
	if status := call(); status != http.StatusServiceUnavailable {
		t.Fatalf("revoked Provider status %d", status)
	}
	assertCounts(4, 2, 1, 1)
}
