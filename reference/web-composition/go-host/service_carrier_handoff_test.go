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
)

type serviceHandoffProcess struct{ calls int }

func (p *serviceHandoffProcess) Invoke(_ context.Context, _ pruntime.ProcessCommand, req pruntime.Request) (pruntime.Response, error) {
	p.calls++
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: json.RawMessage(`{"status":"executed"}`)}, nil
}

type serviceHandoffVerifier struct{ calls int }

func (v *serviceHandoffVerifier) Verify(_ context.Context, _ plugin.Identity, _ plugin.Artifact) (pruntime.ArtifactVerification, error) {
	v.calls++
	return pruntime.ArtifactVerification{VerifierID: "test-only", DigestVerified: true}, nil
}

type serviceHandoffAcceptance struct {
	calls     int
	revoked   bool
	selection pruntime.ServiceProviderSelection
}

func (a *serviceHandoffAcceptance) ReadCurrentAcceptedService(_ context.Context, selected pruntime.ServiceProviderSelection) (pruntime.ServiceAcceptanceRecord, error) {
	a.calls++
	if a.revoked {
		return pruntime.ServiceAcceptanceRecord{}, errors.New("test-only revoked")
	}
	return pruntime.ServiceAcceptanceRecord{
		Selection: selected, ContractAcceptanceRef: "test-contract",
		ProviderConformanceRef: "test-conformance", AuthorityRevision: "test-rev-1",
	}, nil
}

// This reference traverses the EXISTING protected Web Host, Composition
// selection and Product Runtime Service Carrier without inventing a resolver.
// Host session/authorization, Service acceptance and artifact trust are test
// doubles. In particular, this is NOT an issued business/Provider permission.
func TestProtectedActionSelectsProductServiceCarrierWithoutBorrowingApproval(t *testing.T) {
	const actionID = "order_entry.preview_order_quote"
	const serviceID = "sales.order.quote_lines"
	const scopeID = "reference/order_entry"
	digest := "sha256:" + strings.Repeat("a", 64)
	service := composition.Contract{ID: serviceID, Version: "1.0.0", Shape: "test-only-quote-shape"}
	registry := composition.NewRegistry()
	_, err := registry.Apply(0, []composition.Provider{{ID: "quote-provider", Scope: scopeID, Offers: service}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("quote-provider", scopeID)
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{ContractID: serviceID, ContractVersion: service.Version, ProviderID: selected.ProviderID, ArtifactDigest: digest, Generation: 1}
	hostAllowed := true
	coordinator, err := composition.NewSelectedCoordinator(registry, selected, binding, composition.NewScope(), func(actual composition.Binding) error {
		if !hostAllowed || actual != binding {
			return errors.New("test Host guard denied")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	provider := plugin.Identity{ID: "tei.reference.quote", Version: "1.0.0"}
	carrierID := core.CanonicalID("reference.quote.executor")
	serviceSelection := pruntime.ServiceProviderSelection{
		ContractID: serviceID, ContractVersion: service.Version,
		ContractFingerprint: "sha256:" + strings.Repeat("b", 64),
		Scope:               "application-runtime", Provider: provider,
		ImplementationID: "quote-bridge-v1", ArtifactDigest: digest,
	}
	plan := pruntime.EnvironmentPlan{
		Environment: "development", PluginLockFingerprint: "test-locked",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: provider, DescriptorFingerprint: "test-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: "/test-only/quote", SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: provider, Capability: plugin.Capability{Kind: plugin.CapabilityContractQueryExecutor, ID: carrierID},
			Boundary: pruntime.BoundaryProcess,
		}},
		ServiceBindings: []pruntime.ServiceCarrierBinding{{
			ServiceProviderSelection: serviceSelection,
			CarrierKind:              plugin.CapabilityContractQueryExecutor, CarrierID: carrierID, Operation: "quote",
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	process := &serviceHandoffProcess{}
	verifier := &serviceHandoffVerifier{}
	accepted := &serviceHandoffAcceptance{selection: serviceSelection}
	runtimeHost := pruntime.Registry{Processes: process, Verifier: verifier, ServiceAcceptance: accepted}

	var captured *http.Request
	serviceHandler, err := NewHostAuthorizedSelectedCapabilityInvoker(
		coordinator, actionID, serviceID, service.Version, []SelectedActionProvider{{
			ProviderID: binding.ProviderID, ArtifactDigest: digest,
			Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = r
				if got, ok := PrincipalFromContext(r.Context()); !ok || got != currentPrincipal {
					t.Error("Host principal was changed before Service invocation")
				}
				// The trusted Host supplies this exact Plan and Service selection,
				// NOT the browser's claimed application, provider or operation.
				result, callErr := runtimeHost.InvokeSelectedServiceCarrier(r.Context(), plan, serviceSelection,
					"host-action-quote", json.RawMessage(`{"quantity":2,"unitPrice":100}`))
				if callErr != nil {
					http.Error(w, "service unavailable", http.StatusServiceUnavailable)
					return
				}
				if result.Selection != serviceSelection || result.Runtime.Provenance.Artifact.Digest != digest {
					t.Error("Service / Runtime provenance lost")
				}
				w.WriteHeader(http.StatusAccepted)
			}),
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	// Use the actual candidate business Pattern and existing security.authorize
	// adapter, not a boolean substitute for business authorization. These
	// fixture policies and subject links remain synthetic, not production trust.
	resolver, adapters := quoteCandidateAuthorizationDependencies(t)
	resolved, err := resolver.ResolveCommandBinding("order_entry", actionID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Pattern.Capability == nil ||
		string(resolved.Pattern.Capability.ID) != serviceID ||
		resolved.Pattern.Capability.Effect != nil {
		t.Fatalf("quote semantic contract does not match selected service identity: %+v", resolved.Pattern.Capability)
	}
	principal := Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1",
		Subject: "corporate-editor", PrincipalID: "employee-1",
	}
	viewer := Principal{
		ProviderID: "corporate", Issuer: principal.Issuer,
		Subject: "corporate-viewer", PrincipalID: "employee-2",
	}
	currentPrincipal := principal
	linkRevoked, mappingUnavailable := false, false
	authorizer, err := NewCanonicalActionAuthorizer(
		resolver, "order_entry", actionID,
		func(_ context.Context, identity ExternalSubject) (SubjectLink, bool, error) {
			if mappingUnavailable {
				return SubjectLink{}, false, errors.New("synthetic subject mapping unavailable")
			}
			if linkRevoked || identity != (ExternalSubject{
				ProviderID: principal.ProviderID, Issuer: principal.Issuer, Subject: principal.Subject,
			}) {
				return SubjectLink{}, false, nil
			}
			return SubjectLink{PrincipalID: principal.PrincipalID, SubjectID: "subject-order-editor"}, true, nil
		}, adapters,
	)
	if err != nil {
		t.Fatal(err)
	}
	sessionOK, csrfOK := true, true
	host, err := NewHost(Policy{Path: "/protected/quote", Method: http.MethodPost, ActionID: actionID,
		Providers: map[string]string{"corporate": principal.Issuer}}, Dependencies{
		VerifySession: func(*http.Request) (Principal, error) {
			if !sessionOK {
				return Principal{}, errors.New("session revoked")
			}
			return currentPrincipal, nil
		},
		VerifyCSRF: func(*http.Request, Principal) bool { return csrfOK },
		Authorize:  authorizer,
		Invoke:     serviceHandler,
		Audit:  func(AuditRecord) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	invokeHTTP := func(h http.Handler) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/protected/quote",
			strings.NewReader(`{"applicationId":"repair_intake","provider":"forged","operation":"other"}`))
		req.Header.Set("X-Principal-ID", "forged-user")
		req.Header.Set("X-Action-ID", "another.action")
		response := httptest.NewRecorder()
		h.ServeHTTP(response, req)
		return response.Code
	}
	assertCalls := func(expected int) {
		t.Helper()
		if process.calls != expected || verifier.calls != expected || accepted.calls != expected {
			t.Fatalf("unexpected execution: process=%d verifier=%d accepted=%d; want=%d",
				process.calls, verifier.calls, accepted.calls, expected)
		}
	}
	if status := invokeHTTP(serviceHandler); status != http.StatusForbidden {
		t.Fatalf("bare route: %d", status)
	}
	sessionOK = false
	if status := invokeHTTP(host); status != http.StatusUnauthorized {
		t.Fatalf("session denied: %d", status)
	}
	sessionOK, csrfOK = true, false
	if status := invokeHTTP(host); status != http.StatusForbidden {
		t.Fatalf("CSRF denied: %d", status)
	}
	csrfOK = true
	currentPrincipal = viewer
	if status := invokeHTTP(host); status != http.StatusForbidden {
		t.Fatalf("viewer without canonical quote grant was allowed: %d", status)
	}
	currentPrincipal = principal
	linkRevoked = true
	if status := invokeHTTP(host); status != http.StatusForbidden {
		t.Fatalf("revoked subject mapping was allowed: %d", status)
	}
	linkRevoked, mappingUnavailable = false, true
	if status := invokeHTTP(host); status != http.StatusServiceUnavailable {
		t.Fatalf("unavailable subject mapping was silently authorized: %d", status)
	}
	mappingUnavailable = false
	assertCalls(0)
	if status := invokeHTTP(host); status != http.StatusAccepted {
		t.Fatalf("authorized invocation: %d", status)
	}
	assertCalls(1)
	if captured == nil {
		t.Fatal("no protected request reached selected Action")
	}
	replay := httptest.NewRecorder()
	serviceHandler.ServeHTTP(replay, captured)
	if replay.Code != http.StatusForbidden {
		t.Fatalf("expired Host approval replay: %d", replay.Code)
	}
	assertCalls(1)
	hostAllowed = false
	if status := invokeHTTP(host); status != http.StatusServiceUnavailable {
		t.Fatalf("Composition guard denied: %d", status)
	}
	assertCalls(1)
	hostAllowed = true
	accepted.revoked = true
	if status := invokeHTTP(host); status != http.StatusServiceUnavailable {
		t.Fatalf("Provider revoked: %d", status)
	}
	if process.calls != 1 || verifier.calls != 1 || accepted.calls != 2 {
		t.Fatalf("revoked Service reached process/verifier or was not rechecked: process=%d verifier=%d accepted=%d",
			process.calls, verifier.calls, accepted.calls)
	}
}
