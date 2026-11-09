// tei-service-carrier-connect is a read-only reference CLI for the C-2 connect
// point. It builds the invoker and carries one request through it.
//
// Everything that would be Host-owned in production is supplied here by
// reference-only stand-ins: a recording process, a verifier that only reports
// a digest match, and an acceptance reader that returns fixed references. None
// of them proves that a Provider works or that external state changed.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/servicehost"
	servicecarrierconnect "tei/tools/reference-service-carrier-connect"
)

type recordingProcess struct{ payload string }

func (p recordingProcess) Invoke(
	_ context.Context, _ pruntime.ProcessCommand, _ pruntime.Request,
) (pruntime.Response, error) {
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: json.RawMessage(p.payload)}, nil
}

type digestOnlyVerifier struct{}

func (digestOnlyVerifier) Verify(context.Context, plugin.Identity, plugin.Artifact) (pruntime.ArtifactVerification, error) {
	return pruntime.ArtifactVerification{VerifierID: "reference-digest-only", DigestVerified: true}, nil
}

type fixedAcceptance struct{}

func (fixedAcceptance) ReadCurrentAcceptedService(
	_ context.Context, sel pruntime.ServiceProviderSelection,
) (pruntime.ServiceAcceptanceRecord, error) {
	return pruntime.ServiceAcceptanceRecord{
		Selection: sel, ContractAcceptanceRef: "reference-contract-acceptance",
		ProviderConformanceRef: "reference-provider-conformance",
		AuthorityRevision:      "reference-authority-revision",
	}, nil
}

func main() {
	application := flag.String("application", "", "Application whose route the Host owns")
	environmentID := flag.String("environment", "", "Environment whose route the Host owns")
	routeUnavailable := flag.Bool("route-unavailable", false, "simulate a Host with no current route")
	unconfigured := flag.Bool("unconfigured", false, "simulate a Host that supplied no pinned Binding")
	flag.Parse()

	if *unconfigured {
		emit(servicecarrierconnect.Run(servicecarrierconnect.Inputs{}, referenceRequests()))
		return
	}
	if *application == "" || *environmentID == "" {
		fail(fmt.Errorf("-application and -environment are required (or use -unconfigured)"))
	}

	binding := composition.Binding{
		ContractID: "sales.quote-lines", ContractVersion: "1.0.0",
		ProviderID: "logical-quote", Generation: 1,
		ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
	}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{
		ID: binding.ProviderID, Scope: "reference/quote",
		Offers: composition.Contract{
			ID: binding.ContractID, Version: binding.ContractVersion, Shape: "quote-v0",
		},
	}}, nil); err != nil {
		fail(err)
	}
	selected, err := registry.Select(binding.ProviderID, "reference/quote")
	if err != nil {
		fail(err)
	}

	current := func(context.Context, core.CanonicalID, core.CanonicalID, composition.Binding) (servicehost.SelectedServiceCarrierRoute, error) {
		if *routeUnavailable {
			return servicehost.SelectedServiceCarrierRoute{}, fmt.Errorf("the Host has no current route for this generation")
		}
		return referenceRoute(binding), nil
	}

	emit(servicecarrierconnect.Run(servicecarrierconnect.Inputs{
		Application: core.CanonicalID(*application),
		Environment: core.CanonicalID(*environmentID),
		Current:     current,
		Binding:     binding,
		Selected:    selected,
		Registry:    registry,
		Guard: func(actual composition.Binding) error {
			if actual != binding {
				return fmt.Errorf("the Host generation was withdrawn")
			}
			return nil
		},
		Runtime: pruntime.Registry{
			Processes:         recordingProcess{payload: `{"subtotalMinor":1550}`},
			Verifier:          digestOnlyVerifier{},
			ServiceAcceptance: fixedAcceptance{},
		},
	}, referenceRequests()))
}

func referenceRequests() []servicecarrierconnect.Request {
	return []servicecarrierconnect.Request{
		{RequestID: "reference-request", Payload: json.RawMessage(`{"customerCode":"C-1"}`)},
	}
}

func referenceRoute(b composition.Binding) servicehost.SelectedServiceCarrierRoute {
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.quote.v1"), Version: "1.0.0"}
	selection := pruntime.ServiceProviderSelection{
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
			ServiceProviderSelection: selection,
			CarrierKind:              plugin.CapabilityContractQueryExecutor,
			CarrierID:                carrierID,
			Operation:                "quote-v1",
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	return servicehost.SelectedServiceCarrierRoute{Binding: b, Selection: selection, Plan: plan}
}

func emit(report servicecarrierconnect.Report) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
