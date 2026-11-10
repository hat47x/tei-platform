package servicecarrierconnect

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/servicehost"
)

// recordingProcess answers every request with a fixed payload and records how
// many times it was asked. It proves the path ran, not that a Provider works.
type recordingProcess struct {
	calls   int
	payload string
}

func (p *recordingProcess) Invoke(
	_ context.Context, _ pruntime.ProcessCommand, _ pruntime.Request,
) (pruntime.Response, error) {
	p.calls++
	return pruntime.Response{
		Status: pruntime.ResponseOK, Payload: json.RawMessage(p.payload),
	}, nil
}

type acceptingVerifier struct{}

func (acceptingVerifier) Verify(context.Context, plugin.Identity, plugin.Artifact) (pruntime.ArtifactVerification, error) {
	return pruntime.ArtifactVerification{VerifierID: "reference-only", DigestVerified: true}, nil
}

type acceptingService struct{ revoked bool }

func (a acceptingService) ReadCurrentAcceptedService(
	_ context.Context, sel pruntime.ServiceProviderSelection,
) (pruntime.ServiceAcceptanceRecord, error) {
	if a.revoked {
		return pruntime.ServiceAcceptanceRecord{}, errors.New("revoked")
	}
	return pruntime.ServiceAcceptanceRecord{
		Selection: sel, ContractAcceptanceRef: "reference-contract",
		ProviderConformanceRef: "reference-conformance",
		AuthorityRevision:      "reference-revision",
	}, nil
}

func binding() composition.Binding {
	return composition.Binding{
		ContractID: "sales.quote-lines", ContractVersion: "1.0.0",
		ProviderID:     "logical-quote",
		ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
		Generation:     1,
	}
}

func route(b composition.Binding) servicehost.SelectedServiceCarrierRoute {
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.quote.v1"), Version: "1.0.0"}
	sel := pruntime.ServiceProviderSelection{
		ContractID: b.ContractID, ContractVersion: b.ContractVersion,
		ContractFingerprint: "sha256:" + strings.Repeat("b", 64),
		Scope:               "application-runtime",
		Provider:            identity,
		ImplementationID:    "quote-lines-v1",
		ArtifactDigest:      b.ArtifactDigest,
	}
	carrierID := core.MustCanonicalID("reference.quote.carrier")
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("development"),
		PluginLockFingerprint: "reference-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity, DescriptorFingerprint: "reference-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: "/reference-only/quote", SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: b.ArtifactDigest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity,
			Capability: plugin.Capability{
				Kind: plugin.CapabilityContractQueryExecutor, ID: carrierID,
			},
			Boundary: pruntime.BoundaryProcess,
		}},
		ServiceBindings: []pruntime.ServiceCarrierBinding{{
			ServiceProviderSelection: sel,
			CarrierKind:              plugin.CapabilityContractQueryExecutor,
			CarrierID:                carrierID,
			Operation:                "quote-v1",
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	return servicehost.SelectedServiceCarrierRoute{Binding: b, Selection: sel, Plan: plan}
}

func inputs(t *testing.T, process pruntime.ProcessLauncher, failure error) Inputs {
	t.Helper()
	b := binding()
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{
		ID: b.ProviderID, Scope: "reference/quote", Offers: composition.Contract{
			ID: b.ContractID, Version: b.ContractVersion, Shape: "quote-v0",
		},
	}}, nil); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select(b.ProviderID, "reference/quote")
	if err != nil {
		t.Fatal(err)
	}
	current := func(context.Context, core.CanonicalID, core.CanonicalID, composition.Binding) (servicehost.SelectedServiceCarrierRoute, error) {
		if failure != nil {
			return servicehost.SelectedServiceCarrierRoute{}, failure
		}
		return route(b), nil
	}
	return Inputs{
		Application: "order_entry", Environment: "development",
		Current: current, Binding: b, Selected: selected, Registry: registry,
		Guard: func(actual composition.Binding) error {
			if actual != b {
				return errors.New("the Host generation was withdrawn")
			}
			return nil
		},
		Runtime: pruntime.Registry{
			Processes: process, Verifier: acceptingVerifier{},
			ServiceAcceptance: acceptingService{},
		},
	}
}

func TestAPinnedGenerationCarriesTheRequest(t *testing.T) {
	process := &recordingProcess{payload: `{"subtotalMinor":1550}`}
	report := Run(inputs(t, process, nil), []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if report.Ran != 1 || report.Refused != 0 {
		t.Fatalf("report = %+v", report)
	}
	if process.calls != 1 {
		t.Fatalf("the process was called %d times", process.calls)
	}
	if got := report.Outcomes[0]; !got.Pinned || got.Generation != 1 || got.Payload == "" {
		t.Fatalf("outcome = %+v", got)
	}
}

func TestAnUnavailableRouteIsRefusedBeforeAnyInvocation(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	report := Run(inputs(t, process, errors.New("private route unavailable")),
		[]Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if report.Ran != 0 || report.Refused != 1 {
		t.Fatalf("report = %+v", report)
	}
	if process.calls != 0 {
		t.Fatal("the process ran even though the route was unavailable")
	}
	if !strings.Contains(report.Outcomes[0].Error, RefusedRouteUnavailable) {
		t.Fatalf("outcome = %+v", report.Outcomes[0])
	}
}

func TestARouteForAnotherBindingIsRefused(t *testing.T) {
	// The route returned for the pinned Binding is for a different one. The
	// reference must not treat "some route" as "this route".
	process := &recordingProcess{payload: `{}`}
	b := binding()
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{
		ID: b.ProviderID, Scope: "reference/quote", Offers: composition.Contract{
			ID: b.ContractID, Version: b.ContractVersion, Shape: "quote-v0",
		},
	}}, nil); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select(b.ProviderID, "reference/quote")
	if err != nil {
		t.Fatal(err)
	}
	other := b
	other.Generation = 2
	in := Inputs{
		Application: "order_entry", Environment: "development",
		Current: func(context.Context, core.CanonicalID, core.CanonicalID, composition.Binding) (servicehost.SelectedServiceCarrierRoute, error) {
			return route(other), nil
		},
		Binding: b, Selected: selected, Registry: registry,
		Guard: func(composition.Binding) error { return nil },
		Runtime: pruntime.Registry{
			Processes: process, Verifier: acceptingVerifier{},
			ServiceAcceptance: acceptingService{},
		},
	}
	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if report.Ran != 0 || report.Refused != 1 {
		t.Fatalf("a mismatched route was accepted: %+v", report)
	}
}

func TestMissingHostInputsAreRefusedNotInvented(t *testing.T) {
	report := Run(Inputs{}, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if report.Ran != 0 || report.Refused != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.Outcomes[0].Error == "" {
		t.Fatal("a refusal must explain itself")
	}
}

func TestReportStatesAuthorityAndSaysWhatItDoesNotClaim(t *testing.T) {
	report := Run(inputs(t, &recordingProcess{payload: `{}`}, nil), nil)
	if report.Authority != Authority || report.Schema != Schema || report.RuleFingerprint == "" {
		t.Fatalf("bad header: %+v", report)
	}
	if len(report.Notes) == 0 {
		t.Fatal("the report must state what it does not establish")
	}
	if report.Application != "order_entry" || report.Environment != "development" {
		t.Fatalf("identity = %q / %q", report.Application, report.Environment)
	}
}
