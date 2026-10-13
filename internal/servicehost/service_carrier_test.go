package servicehost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

type generationCarrierProcess struct {
	mu         sync.Mutex
	requests   []pruntime.Request
	oldStarted chan struct{}
	oldRelease chan struct{}
}

func (p *generationCarrierProcess) Invoke(
	ctx context.Context, _ pruntime.ProcessCommand, req pruntime.Request,
) (pruntime.Response, error) {
	p.mu.Lock()
	p.requests = append(p.requests, req)
	p.mu.Unlock()
	if req.Operation == "quote-v1" && p.oldStarted != nil {
		close(p.oldStarted)
		select {
		case <-p.oldRelease:
		case <-ctx.Done():
			return pruntime.Response{}, ctx.Err()
		}
	}
	return pruntime.Response{
		Status: pruntime.ResponseOK, Payload: json.RawMessage(`{"subtotalMinor":1550}`),
	}, nil
}

func (p *generationCarrierProcess) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.requests)
}

type generationCarrierVerifier struct{ calls int }

func (v *generationCarrierVerifier) Verify(
	_ context.Context, _ plugin.Identity, _ plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	// Test only: this does not check a binary signature or provenance.
	return pruntime.ArtifactVerification{
		VerifierID: "test-generation-digest", DigestVerified: true,
	}, nil
}

type generationCarrierAcceptance struct {
	reads   int
	revoked bool
}

func (a *generationCarrierAcceptance) ReadCurrentAcceptedService(
	_ context.Context, selected pruntime.ServiceProviderSelection,
) (pruntime.ServiceAcceptanceRecord, error) {
	a.reads++
	if a.revoked {
		return pruntime.ServiceAcceptanceRecord{}, errors.New("test-only revoked")
	}
	return pruntime.ServiceAcceptanceRecord{
		Selection: selected, ContractAcceptanceRef: "test-contract-publisher",
		ProviderConformanceRef: "test-provider-conformance",
		AuthorityRevision: "test-current-revision",
	}, nil
}

func generationCarrierRoute(binding composition.Binding, pluginID string) SelectedServiceCarrierRoute {
	identity := plugin.Identity{
		ID: core.MustCanonicalID(pluginID), Version: "1.0.0",
	}
	selection := pruntime.ServiceProviderSelection{
		ContractID: binding.ContractID, ContractVersion: binding.ContractVersion,
		ContractFingerprint: "sha256:" + strings.Repeat("b", 64),
		Scope:               "application-runtime",
		Provider:            identity,
		ImplementationID:    "quote-lines-v1",
		ArtifactDigest:      binding.ArtifactDigest,
	}
	carrierID := core.MustCanonicalID("reference.quote.carrier")
	plan := pruntime.EnvironmentPlan{
		Environment: core.MustCanonicalID("development"),
		PluginLockFingerprint: "test-host-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity, DescriptorFingerprint: "test-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: "/test-only/quote", SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: binding.ArtifactDigest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity,
			Capability: plugin.Capability{
				Kind: plugin.CapabilityContractQueryExecutor, ID: carrierID,
			},
			Boundary: pruntime.BoundaryProcess,
		}},
		ServiceBindings: []pruntime.ServiceCarrierBinding{{
			ServiceProviderSelection: selection,
			CarrierKind:              plugin.CapabilityContractQueryExecutor,
			CarrierID:                carrierID,
			Operation:                fmt.Sprintf("quote-v%d", binding.Generation),
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	return SelectedServiceCarrierRoute{
		Binding: binding, Selection: selection, Plan: plan,
	}
}

func generationCarrierFixture(t *testing.T) (
	*composition.Coordinator, *SelectedServiceCarrierInvoker,
	*generationCarrierProcess, *generationCarrierVerifier,
	*generationCarrierAcceptance, composition.Binding, composition.Binding,
) {
	t.Helper()
	first := composition.Binding{
		ContractID: "sales.quote-lines", ContractVersion: "1.0.0",
		ProviderID: "logical-quote-A",
		ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
		Generation: 1,
	}
	second := composition.Binding{
		ContractID: first.ContractID, ContractVersion: first.ContractVersion,
		ProviderID: "logical-quote-B",
		ArtifactDigest: "sha256:" + strings.Repeat("c", 64),
		Generation: 2,
	}
	router, err := composition.NewRouter(first, composition.NewScope())
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := composition.NewCoordinator(router, func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	process := &generationCarrierProcess{}
	verifier := &generationCarrierVerifier{}
	accepted := &generationCarrierAcceptance{}
	runtimeHost := pruntime.Registry{
		Processes: process, Verifier: verifier, ServiceAcceptance: accepted,
	}
	routes := map[uint64]SelectedServiceCarrierRoute{
		1: generationCarrierRoute(first, "tei.quote.v1"),
		2: generationCarrierRoute(second, "tei.quote.v2"),
	}
	invoker, err := NewSelectedServiceCarrierInvoker(
		coordinator, runtimeHost,
		func(_ context.Context, b composition.Binding) (SelectedServiceCarrierRoute, error) {
			r, exists := routes[b.Generation]
			if !exists {
				return SelectedServiceCarrierRoute{}, errors.New("private unavailable route")
			}
			return r, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return coordinator, invoker, process, verifier, accepted, first, second
}

func TestSelectedServiceCarrierPinsGenerationAcrossCutover(t *testing.T) {
	coordinator, invoker, process, verifier, accepted, first, second := generationCarrierFixture(t)
	process.oldStarted = make(chan struct{})
	process.oldRelease = make(chan struct{})
	oldResult := make(chan SelectedServiceCarrierInvocation, 1)
	oldErr := make(chan error, 1)
	go func() {
		result, err := invoker.Invoke(
			context.Background(), "old-generation",
			json.RawMessage(`{"lines":[{"quantity":3,"unitPrice":250}]}`),
		)
		oldResult <- result
		oldErr <- err
	}()
	<-process.oldStarted

	if err := coordinator.Replace(
		context.Background(), first.Generation, second,
		composition.Lifecycle{
			Prepare: func(context.Context, *composition.Scope) error { return nil },
			Ready: func(context.Context, composition.Binding) error { return nil },
		},
	); err != nil {
		t.Fatal(err)
	}
	current, err := invoker.Invoke(
		context.Background(), "new-generation", json.RawMessage(`{"lines":[]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if current.Generation != second ||
		current.Service.Selection.ArtifactDigest != second.ArtifactDigest ||
		current.Service.Runtime.Provenance.Plugin.ID != core.MustCanonicalID("tei.quote.v2") {
		t.Fatalf("new invocation was not bound to new generation: %+v", current)
	}
	close(process.oldRelease)
	old := <-oldResult
	if err := <-oldErr; err != nil {
		t.Fatal(err)
	}
	if old.Generation != first ||
		old.Service.Selection.ArtifactDigest != first.ArtifactDigest ||
		old.Service.Runtime.Provenance.Plugin.ID != core.MustCanonicalID("tei.quote.v1") {
		t.Fatalf("in-flight invocation switched Provider across cutover: %+v", old)
	}
	if process.count() != 2 || verifier.calls != 2 || accepted.reads != 2 {
		t.Fatalf("generation invocations bypassed existing Runtime: child=%d verify=%d accepted=%d",
			process.count(), verifier.calls, accepted.reads)
	}
	if err := coordinator.Reap(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSelectedServiceCarrierRejectsWrongGenerationRouteBeforeHostExecution(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*SelectedServiceCarrierRoute)
	}{
		{
			name: "different generation",
			mutate: func(r *SelectedServiceCarrierRoute) { r.Binding.Generation++ },
		},
		{
			name: "different logical provider",
			mutate: func(r *SelectedServiceCarrierRoute) { r.Binding.ProviderID = "other-logical-provider" },
		},
		{
			name: "different Contract",
			mutate: func(r *SelectedServiceCarrierRoute) { r.Selection.ContractID = "sales.other-contract" },
		},
		{
			name: "different Contract version",
			mutate: func(r *SelectedServiceCarrierRoute) { r.Selection.ContractVersion = "2.0.0" },
		},
		{
			name: "different physical artifact",
			mutate: func(r *SelectedServiceCarrierRoute) { r.Selection.ArtifactDigest = "sha256:" + strings.Repeat("e", 64) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			coordinator, _, process, verifier, accepted, first, _ := generationCarrierFixture(t)
			route := generationCarrierRoute(first, "tei.quote.v1")
			tc.mutate(&route)
			host := pruntime.Registry{
				Processes: process, Verifier: verifier, ServiceAcceptance: accepted,
			}
			invoker, err := NewSelectedServiceCarrierInvoker(
				coordinator, host,
				func(context.Context, composition.Binding) (SelectedServiceCarrierRoute, error) {
					return route, nil
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = invoker.Invoke(context.Background(), "wrong-route", json.RawMessage(`{"ok":true}`))
			if !errors.Is(err, ErrServiceCarrierGenerationMismatch) {
				t.Fatalf("mismatched pinned generation was accepted: %v", err)
			}
			if accepted.reads != 0 || verifier.calls != 0 || process.count() != 0 {
				t.Fatal("wrong generation reached Host current acceptance or process")
			}
		})
	}
}

func TestSelectedServiceCarrierRouteUnavailableAndCancellationFailClosed(t *testing.T) {
	coordinator, _, process, verifier, accepted, _, _ := generationCarrierFixture(t)
	host := pruntime.Registry{
		Processes: process, Verifier: verifier, ServiceAcceptance: accepted,
	}
	if _, err := NewSelectedServiceCarrierInvoker(nil, host, nil); !errors.Is(err, ErrServiceCarrierGenerationInvalid) {
		t.Fatalf("missing Host coordinator/source accepted: %v", err)
	}
	lookupCalls := 0
	secret := errors.New("private/host/routes/issuer-token")
	invoker, err := NewSelectedServiceCarrierInvoker(
		coordinator, host,
		func(context.Context, composition.Binding) (SelectedServiceCarrierRoute, error) {
			lookupCalls++
			return SelectedServiceCarrierRoute{}, secret
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = invoker.Invoke(context.Background(), "unavailable", json.RawMessage(`{"ok":true}`))
	if !errors.Is(err, ErrServiceCarrierGenerationUnavailable) ||
		errors.Is(err, secret) || strings.Contains(err.Error(), "issuer-token") {
		t.Fatalf("Host private route error leaked: %v", err)
	}
	before, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = invoker.Invoke(before, "canceled", json.RawMessage(`{"ok":true}`))
	if !errors.Is(err, context.Canceled) || lookupCalls != 1 {
		t.Fatalf("canceled request reached Host route: %v lookups=%d", err, lookupCalls)
	}
	_, err = invoker.Invoke(nil, "nil-context", json.RawMessage(`{"ok":true}`))
	if !errors.Is(err, ErrServiceCarrierGenerationInvalid) || lookupCalls != 1 {
		t.Fatalf("nil context reached Host route: %v", err)
	}
	if accepted.reads != 0 || verifier.calls != 0 || process.count() != 0 {
		t.Fatal("unavailable route reached accepted Runtime execution")
	}
}

func TestSelectedServiceCarrierCurrentAcceptanceIsNotGenerationAuthorization(t *testing.T) {
	_, invoker, process, verifier, accepted, _, _ := generationCarrierFixture(t)
	accepted.revoked = true
	_, err := invoker.Invoke(
		context.Background(), "revoked-provider", json.RawMessage(`{"ok":true}`),
	)
	if !errors.Is(err, pruntime.ErrServiceAcceptanceDenied) {
		t.Fatalf("generation lease bypassed current Provider acceptance: %v", err)
	}
	if accepted.reads != 1 || verifier.calls != 0 || process.count() != 0 {
		t.Fatal("revoked Provider reached verifier or external process")
	}
}
