package capabilitytransport

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// The verifier checks executable bytes, not publisher signatures or artifact
// provenance. It is a TEST-ONLY host verifier, not a trusted product signer.
type quoteRuntimeHashVerifier struct {
	path  string
	calls int
}

func (v *quoteRuntimeHashVerifier) Verify(
	_ context.Context, _ plugin.Identity, artifact plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	data, err := os.ReadFile(v.path)
	if err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	if digestBytes(data) != artifact.Digest {
		return pruntime.ArtifactVerification{}, errors.New("executable digest mismatch")
	}
	return pruntime.ArtifactVerification{
		VerifierID:     "reference.quote.test-hash-only",
		DigestVerified: true,
	}, nil
}

type quoteRuntimeProcessLauncher struct {
	calls int
	last  pruntime.Request
}

func (p *quoteRuntimeProcessLauncher) Invoke(ctx context.Context, cmd pruntime.ProcessCommand, req pruntime.Request) (pruntime.Response, error) {
	p.calls++
	p.last = req
	return (pruntime.StdioProcessLauncher{}).Invoke(ctx, cmd, req)
}

// This test deliberately enters the EXISTING product Runtime Registry,
// rather than directly launching the process after a Composition guard.
// The generic carrier remains a legacy transport for this REFERENCE only;
// the read-only quote is not being registered as a ContractQuery business
// capability. Neither the test hash verifier nor SandboxNone proves full
// production artifact authority, operating-system isolation or a resource
// lease. Formal candidate Shape/Provider trust is tested separately.
func TestIndependentQuoteRunsThroughExistingRuntimeAuthorityBoundary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference process command paths need platform-specific path policy")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Compile the actual independent provider, with no import of TEI internals.
	work := t.TempDir()
	source, err := os.ReadFile("../../plugins/reference-order-quote/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "main.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "go.mod"),
		[]byte("module example.invalid/quote-runtime-boundary\n\ngo 1.23\n"), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "order-quote")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = work
	build.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile reference quote executable: %v\n%s", err, output)
	}
	image, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	digest := digestBytes(image)
	identity := plugin.Identity{
		ID: core.MustCanonicalID("tei.reference.order_quote"), Version: "1.0.0",
	}
	legacyID := core.MustCanonicalID("transport.order_quote_carrier")
	descriptor := digestBytes([]byte("reference.order-quote.transport-carrier/v1"))
	const lockFingerprint = "test-only-quote-process-lock"
	provider := GenericProviderIdentity{
		PluginID: string(identity.ID), PluginVersion: identity.Version,
		ImplementationID: "order-quote.v1", ArtifactDigest: digest,
	}
	dispatch := GenericDispatch{
		Schema: dispatchSchema,
		ContractID: "reference.order.quote-lines/v1",
		ContractFingerprint: digestBytes([]byte("reference.order.quote-lines.contract/v1")),
		Target: "reference/order_entry",
		Provider: provider,
		RequestShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.request/v1")),
		ResultShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.result/v1")),
		PayloadCodec: "application/json",
		Payload: []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250},{"sku":"BOOK","quantity":2,"unitPriceMinor":400}]}`),
	}
	transportPlan, err := buildTransportDispatchPlan(genericDispatchIdentity(dispatch))
	if err != nil {
		t.Fatal(err)
	}
	// The legacy kind is the EXISTING stdio carrier. It does not turn this
	// price calculation into a domain Contract Query.
	capability := plugin.Capability{
		Kind: plugin.CapabilityContractQueryExecutor, ID: legacyID,
	}
	plan := pruntime.EnvironmentPlan{
		Environment: core.MustCanonicalID("reference"),
		PluginLockFingerprint: lockFingerprint,
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity,
			DescriptorFingerprint: descriptor,
			Boundary: pruntime.BoundaryProcess,
			Trust: pruntime.TrustExternalProcess,
			Command: binary,
			SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity, Capability: capability,
			Boundary: pruntime.BoundaryProcess,
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	verifier := &quoteRuntimeHashVerifier{path: binary}
	processes := &quoteRuntimeProcessLauncher{}
	registry := pruntime.Registry{Processes: processes, Verifier: verifier}
	sequence := 0
	call := func(candidate pruntime.EnvironmentPlan, host pruntime.Registry, request GenericDispatch) (pruntime.Invocation, error) {
		if err := validateDispatchAgainstPlan(transportPlan, request); err != nil {
			return pruntime.Invocation{}, err
		}
		sequence++
		return host.InvokeWithProvenance(
			ctx, candidate, plugin.CapabilityContractQueryExecutor, legacyID,
			"generic-dispatch-v0", "quote-runtime-"+strconv.Itoa(sequence), request,
		)
	}

	invocation, err := call(plan, registry, dispatch)
	if err != nil {
		t.Fatal(err)
	}
	var result GenericTransportResult
	if err := decodeJSONExact(invocation.Response.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if err := validateGenericTransportResult(dispatch, result); err != nil {
		t.Fatal(err)
	}
	var quote struct {
		Currency      string `json:"currency"`
		LineCount     int    `json:"lineCount"`
		SubtotalMinor int64  `json:"subtotalMinor"`
	}
	if err := decodeJSONExact(result.Payload, &quote); err != nil {
		t.Fatal(err)
	}
	if quote.Currency != "JPY" || quote.LineCount != 2 || quote.SubtotalMinor != 1550 {
		t.Fatalf("real independent business result incorrect: %+v", quote)
	}
	if processes.calls != 1 || verifier.calls != 1 {
		t.Fatalf("host failed to verify before child launch: verifier=%d launch=%d",
			verifier.calls, processes.calls)
	}
	provenance := invocation.Provenance
	if provenance.Plugin != identity ||
		provenance.CapabilityID != legacyID ||
		provenance.CapabilityKind != plugin.CapabilityContractQueryExecutor ||
		provenance.Boundary != pruntime.BoundaryProcess ||
		provenance.PluginLockFingerprint != lockFingerprint ||
		provenance.Artifact.Digest != digest ||
		provenance.Verification == nil ||
		!provenance.Verification.DigestVerified ||
		provenance.Verification.SignatureVerified ||
		provenance.Verification.ProvenanceChecked ||
		provenance.Sandboxed || provenance.ResourceLeaseID != "" {
		t.Fatalf("reference Runtime evidence must not imply absent authority: %+v", provenance)
	}
	if processes.last.PluginLockFingerprint != lockFingerprint ||
		processes.last.Plugin != identity ||
		processes.last.CapabilityID != legacyID ||
		processes.last.DescriptorFingerprint != descriptor ||
		processes.last.Operation != "generic-dispatch-v0" {
		t.Fatalf("real process lost locked Runtime routing identity: %+v", processes.last)
	}
	wire := string(processes.last.Payload)
	for _, secret := range []string{"Bearer ", "principalId", "sessionId", "csrf", "hostActionApproval"} {
		if strings.Contains(wire, secret) {
			t.Fatalf("unexpected Host credential/approval on process wire: %q", secret)
		}
	}

	// Every rejection must happen BEFORE the operating system starts a new
	// process. Failure to launch is not proof that a sandbox is installed.
	tests := []struct {
		name       string
		mutate     func(*pruntime.EnvironmentPlan, *pruntime.Registry)
		wantError  string
	}{
		{
			name: "missing host verifier",
			mutate: func(_ *pruntime.EnvironmentPlan, r *pruntime.Registry) { r.Verifier = nil },
			wantError: "no artifact verifier",
		},
		{
			name: "artifact bytes do not match selected digest",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.Registry) {
				p.Plugins[0].Artifact.Digest = digestBytes([]byte("different image"))
			},
			wantError: "digest mismatch",
		},
		{
			name: "signature claimed but reference verifier cannot attest",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.Registry) {
				p.Plugins[0].Artifact.SignatureRef = "unverified-signer"
			},
			wantError: "signature was not verified",
		},
		{
			name: "provenance claimed but reference verifier cannot attest",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.Registry) {
				p.Plugins[0].Artifact.Provenance = "unverified-origin"
			},
			wantError: "provenance was not checked",
		},
		{
			name: "sandbox required but no sandbox launcher",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.Registry) {
				p.Plugins[0].SandboxPolicy = plugin.SandboxRequired
			},
			wantError: "requires sandbox launcher",
		},
		{
			name: "environment capability omitted",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.Registry) {
				p.Capabilities = nil
			},
			wantError: "not uniquely resolved",
		},
		{
			name: "capability permission not granted",
			mutate: func(p *pruntime.EnvironmentPlan, _ *pruntime.Registry) {
				p.Capabilities[0].Capability.Permissions = []permission.Spec{{
					Kind: permission.KindNetwork, Access: permission.AccessConnect,
					Scope: "database.primary",
				}}
			},
			wantError: "requires permission",
		},
		{
			name: "host process launcher unavailable",
			mutate: func(_ *pruntime.EnvironmentPlan, r *pruntime.Registry) {
				r.Processes = nil
			},
			wantError: "no host process or sandbox launcher",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := plan
			next.Plugins = append([]pruntime.PluginRuntime(nil), plan.Plugins...)
			next.Capabilities = append([]pruntime.CapabilityRuntime(nil), plan.Capabilities...)
			next.Capabilities[0].Capability.Permissions = append([]permission.Spec(nil), plan.Capabilities[0].Capability.Permissions...)
			host := registry
			tt.mutate(&next, &host)
			next.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(next)
			before := processes.calls
			_, err := call(next, host, dispatch)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected fail-closed %q, got %v", tt.wantError, err)
			}
			if processes.calls != before {
				t.Fatalf("untrusted Runtime launched child: before=%d after=%d", before, processes.calls)
			}
		})
	}
	t.Run("transport identity drift rejected before Runtime", func(t *testing.T) {
		modified := dispatch
		modified.ContractFingerprint = digestBytes([]byte("unaccepted contract"))
		before, verified := processes.calls, verifier.calls
		if _, err := call(plan, registry, modified); err == nil ||
			processes.calls != before || verifier.calls != verified {
			t.Fatalf("transport identity drift reached Runtime: err=%v", err)
		}
	})
}
