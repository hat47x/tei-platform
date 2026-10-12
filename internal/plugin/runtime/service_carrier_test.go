package runtime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

type serviceCarrierVerifier struct {
	calls int
}

func (v *serviceCarrierVerifier) Verify(
	_ context.Context, _ plugin.Identity, _ plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	// This is not a production signature/provenance verifier. The test
	// establishes only that the existing Runtime verification gate is used.
	return pruntime.ArtifactVerification{
		VerifierID: "reference.test-only", DigestVerified: true,
	}, nil
}

// Test-only Host reader. Production must bind this port to the existing
// trusted Contract acceptance and Provider conformance records, including
// revocation; a plugin cannot install this reader.
type serviceCarrierAcceptedReader struct {
	record pruntime.ServiceAcceptanceRecord
	denied bool
	calls  int
}

func (r *serviceCarrierAcceptedReader) ReadCurrentAcceptedService(
	_ context.Context, _ pruntime.ServiceProviderSelection,
) (pruntime.ServiceAcceptanceRecord, error) {
	r.calls++
	if r.denied {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("service acceptance was revoked")
	}
	return r.record, nil
}

func serviceCarrierHost(
	selected pruntime.ServiceProviderSelection,
	processes *serviceCarrierProcess, verifier *serviceCarrierVerifier,
) (pruntime.Registry, *serviceCarrierAcceptedReader) {
	acceptance := &serviceCarrierAcceptedReader{
		record: pruntime.ServiceAcceptanceRecord{
			Selection:              selected,
			ContractAcceptanceRef:  "reference-contract-acceptance",
			ProviderConformanceRef: "reference-provider-conformance",
			AuthorityRevision:      "test-local-revision-1",
		},
	}
	return pruntime.Registry{
		Processes: processes, Verifier: verifier, ServiceAcceptance: acceptance,
	}, acceptance
}

type serviceCarrierProcess struct {
	calls int
	last  pruntime.Request
	base  pruntime.ProcessLauncher
}

func (p *serviceCarrierProcess) Invoke(
	ctx context.Context, command pruntime.ProcessCommand, request pruntime.Request,
) (pruntime.Response, error) {
	p.calls++
	p.last = request
	return p.base.Invoke(ctx, command, request)
}

func serviceCarrierFixture(t *testing.T) (
	pruntime.EnvironmentPlan, pruntime.ServiceProviderSelection,
	*serviceCarrierProcess, *serviceCarrierVerifier,
) {
	t.Helper()
	request := processRequest()
	base, command := helperLauncher(t, "echo")
	digest := "sha256:" + strings.Repeat("a", 64)
	selected := pruntime.ServiceProviderSelection{
		ContractID:          "reference.order.quote-lines/v1",
		ContractVersion:     "1.0.0",
		ContractFingerprint: "sha256:" + strings.Repeat("b", 64),
		Scope:               "application-runtime",
		Provider:            request.Plugin,
		ImplementationID:    "quote.v1",
		ArtifactDigest:      digest,
	}
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("development"),
		PluginLockFingerprint: "reference-locked-plugins",
		Plugins: []pruntime.PluginRuntime{{
			Plugin:                request.Plugin,
			DescriptorFingerprint: request.DescriptorFingerprint,
			Boundary:              pruntime.BoundaryProcess,
			Trust:                 pruntime.TrustExternalProcess,
			Command:               command.Executable,
			SandboxPolicy:         plugin.SandboxNone,
			Artifact:              plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: request.Plugin,
			Capability: plugin.Capability{
				Kind: request.CapabilityKind,
				ID:   request.CapabilityID,
			},
			Boundary: pruntime.BoundaryProcess,
		}},
		ServiceBindings: []pruntime.ServiceCarrierBinding{{
			ServiceProviderSelection: selected,
			CarrierKind:              request.CapabilityKind,
			CarrierID:                request.CapabilityID,
			Operation:                request.Operation,
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	return plan, selected, &serviceCarrierProcess{base: base}, &serviceCarrierVerifier{}
}

func TestSelectedServiceUsesExistingRuntimeHostAuthorityAndProcess(t *testing.T) {
	plan, selected, child, verifier := serviceCarrierFixture(t)
	host, acceptance := serviceCarrierHost(selected, child, verifier)
	request := json.RawMessage(`{"value":"ok","operation":"not-a-host-operation"}`)
	result, err := host.InvokeSelectedServiceCarrier(
		context.Background(), plan, selected, "service-execution-1", request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if child.calls != 1 || verifier.calls != 1 || acceptance.calls != 1 ||
		child.last.Operation != plan.ServiceBindings[0].Operation ||
		child.last.Plugin != selected.Provider ||
		child.last.CapabilityID != plan.ServiceBindings[0].CarrierID ||
		string(child.last.Payload) != string(request) {
		t.Fatalf("selected service bypassed fixed Runtime dispatch: calls=%d verified=%d request=%+v", child.calls, verifier.calls, child.last)
	}
	if string(result.Runtime.Response.Payload) != string(request) ||
		result.Runtime.Provenance.Artifact.Digest != selected.ArtifactDigest ||
		result.Runtime.Provenance.Plugin != selected.Provider ||
		result.Selection != selected || result.Acceptance != acceptance.record {
		t.Fatalf("missing exact selected service/Runtime provenance or current Host acceptance: %+v", result)
	}
	if len(result.AuthorityLimits) != 4 ||
		result.Runtime.Provenance.Verification == nil ||
		!result.Runtime.Provenance.Verification.DigestVerified ||
		result.Runtime.Provenance.Verification.SignatureVerified ||
		result.Runtime.Provenance.Verification.ProvenanceChecked {
		t.Fatalf("Host evidence or explicit non-authority limit was lost: %+v", result)
	}
}

func TestSelectedServiceRejectsDriftBeforeVerifierOrProcess(t *testing.T) {
	base, selected, child, verifier := serviceCarrierFixture(t)
	host, _ := serviceCarrierHost(selected, child, verifier)
	payload := json.RawMessage(`{"value":"ok"}`)
	for _, tc := range []struct {
		name   string
		mutate func(*pruntime.EnvironmentPlan, *pruntime.ServiceProviderSelection, *pruntime.Registry)
		want   string
	}{
		{
			name: "contract fingerprint drift",
			mutate: func(_ *pruntime.EnvironmentPlan, s *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				s.ContractFingerprint = "sha256:" + strings.Repeat("c", 64)
			},
			want: "differs from Host binding",
		},
		{
			name: "implementation drift",
			mutate: func(_ *pruntime.EnvironmentPlan, s *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				s.ImplementationID = "different-provider"
			},
			want: "differs from Host binding",
		},
		{
			name: "contract version drift",
			mutate: func(_ *pruntime.EnvironmentPlan, s *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				s.ContractVersion = "1.1.0"
			},
			want: "no Host binding",
		},
		{
			name: "wrong authority scope",
			mutate: func(_ *pruntime.EnvironmentPlan, s *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				s.Scope = "authoring-tool"
			},
			want: "only supports application-runtime",
		},
		{
			name: "selected physical artifact drift",
			mutate: func(_ *pruntime.EnvironmentPlan, s *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				s.ArtifactDigest = "sha256:" + strings.Repeat("d", 64)
			},
			want: "differs from Host binding",
		},
		{
			name: "Host binding omitted",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				p.ServiceBindings = nil
			},
			want: "no Host binding",
		},
		{
			name: "two Host candidates are ambiguous",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				p.ServiceBindings = append(p.ServiceBindings, p.ServiceBindings[0])
			},
			want: "ambiguous Host bindings",
		},
		{
			name: "Host binding tampered without policy update",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				p.ServiceBindings[0].ImplementationID = "tampered"
				p.RuntimePolicyFingerprint = base.RuntimePolicyFingerprint
			},
			want: "Environment Policy fingerprint",
		},
		{
			name: "installed artifact drift",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				p.Plugins[0].Artifact.Digest = "sha256:" + strings.Repeat("e", 64)
			},
			want: "artifact differs from Environment Plan",
		},
		{
			name: "existing Runtime capability absent",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				p.Capabilities = nil
			},
			want: "does not match unique Runtime capability",
		},
		{
			name: "required Runtime permission not granted",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				p.Capabilities[0].Capability.Permissions = []permission.Spec{{
					Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database.primary",
				}}
			},
			want: "requires permission",
		},
		{
			name: "missing Runtime artifact verifier",
			mutate: func(_ *pruntime.EnvironmentPlan, _ *pruntime.ServiceProviderSelection, r *pruntime.Registry) {
				r.Verifier = nil
			},
			want: "no artifact verifier",
		},
		{
			name: "caller cannot invent new Host legacy carrier",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.ServiceProviderSelection, _ *pruntime.Registry) {
				p.ServiceBindings[0].CarrierID = core.MustCanonicalID("different.legacy.carrier")
			},
			want: "does not match unique Runtime capability",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, s, runtimeHost := base, selected, host
			plan.Plugins = append([]pruntime.PluginRuntime(nil), base.Plugins...)
			plan.Capabilities = append([]pruntime.CapabilityRuntime(nil), base.Capabilities...)
			plan.ServiceBindings = append([]pruntime.ServiceCarrierBinding(nil), base.ServiceBindings...)
			tc.mutate(&plan, &s, &runtimeHost)
			if tc.name != "Host binding tampered without policy update" {
				plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
			}
			beforeProcess, beforeVerifier := child.calls, verifier.calls
			_, err := runtimeHost.InvokeSelectedServiceCarrier(
				context.Background(), plan, s, "service-deny-1", payload,
			)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
			// Permission rejection may happen after the pure binding gate but
			// before artifact verification and external process launch.
			if child.calls != beforeProcess || verifier.calls != beforeVerifier {
				t.Fatalf("invalid service reached Host verifier/process: %s", tc.name)
			}
		})
	}
}

func TestSelectedServiceAcceptanceIsHostOwnedAndRecheckedForRevocation(t *testing.T) {
	plan, selected, child, verifier := serviceCarrierFixture(t)
	host, acceptance := serviceCarrierHost(selected, child, verifier)
	payload := json.RawMessage(`{"quantity":1}`)

	// The Environment Plan is insufficient without an external current
	// acceptance source, even though its fingerprint matches.
	noReader := host
	noReader.ServiceAcceptance = nil
	if _, err := noReader.InvokeSelectedServiceCarrier(
		context.Background(), plan, selected, "no-reader", payload,
	); err == nil || !strings.Contains(err.Error(), "no current Service acceptance source") {
		t.Fatalf("Service executed using Plan alone: %v", err)
	}
	if verifier.calls != 0 || child.calls != 0 {
		t.Fatal("service request reached verifier or process without accepted source")
	}

	for _, tc := range []struct {
		name   string
		mutate func()
		want   string
	}{
		{
			name: "contract acceptance missing",
			mutate: func() { acceptance.record.ContractAcceptanceRef = "" },
			want: "acceptance record is incomplete",
		},
		{
			name: "provider conformance missing",
			mutate: func() { acceptance.record.ProviderConformanceRef = "" },
			want: "acceptance record is incomplete",
		},
		{
			name: "authority current revision missing",
			mutate: func() { acceptance.record.AuthorityRevision = "" },
			want: "acceptance record is incomplete",
		},
		{
			name: "Host record for different implementation",
			mutate: func() { acceptance.record.Selection.ImplementationID = "unrelated" },
			want: "selection differs from selected Provider",
		},
		{
			name: "Host acceptance was revoked",
			mutate: func() { acceptance.denied = true },
			want: "current Service acceptance denied",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := acceptance.record
			tc.mutate()
			_, err := host.InvokeSelectedServiceCarrier(
				context.Background(), plan, selected, "unaccepted", payload,
			)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unaccepted Service should fail closed: %v", err)
			}
			if verifier.calls != 0 || child.calls != 0 {
				t.Fatal("unaccepted Service reached runtime verifier or external process")
			}
			acceptance.record = original
			acceptance.denied = false
		})
	}

	// Current Host acceptance authorizes this ROUTE only. Existing Runtime
	// still independently validates artifact/permissions and executes.
	if _, err := host.InvokeSelectedServiceCarrier(
		context.Background(), plan, selected, "accepted", payload,
	); err != nil {
		t.Fatal(err)
	}
	if child.calls != 1 || verifier.calls != 1 {
		t.Fatal("current Host acceptance did not reach existing Runtime")
	}
	// Source changes while the Environment Plan and Service selection do not.
	acceptance.denied = true
	before := acceptance.calls
	if _, err := host.InvokeSelectedServiceCarrier(
		context.Background(), plan, selected, "revoked-after-success", payload,
	); err == nil || !strings.Contains(err.Error(), "current Service acceptance denied") {
		t.Fatalf("revocation with unchanged Plan/selection was ignored: %v", err)
	}
	if acceptance.calls != before+1 || child.calls != 1 || verifier.calls != 1 {
		t.Fatal("stale accepted decision reused after revocation")
	}
}

func TestSelectedServiceRequiresValidWireAndDeclaredHostPolicy(t *testing.T) {
	plan, selection, child, verifier := serviceCarrierFixture(t)
	host, acceptance := serviceCarrierHost(selection, child, verifier)
	for _, payload := range []json.RawMessage{
		nil, json.RawMessage(`{"not":"complete"`),
	} {
		if _, err := host.InvokeSelectedServiceCarrier(context.Background(), plan, selection, "bad-wire", payload); err == nil {
			t.Fatal("invalid service JSON reached Host Runtime")
		}
	}
	if child.calls != 0 || verifier.calls != 0 || acceptance.calls != 0 {
		t.Fatal("invalid JSON reached Host acceptance, verifier or process")
	}
}
