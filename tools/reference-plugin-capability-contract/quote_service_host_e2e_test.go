package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// This verifier only checks the process bytes in this test. It neither
// authenticates a publisher nor establishes artifact origin or signature.
type quoteServiceHostHashVerifier struct {
	path  string
	calls int
}

func (v *quoteServiceHostHashVerifier) Verify(
	_ context.Context, _ plugin.Identity, artifact plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	raw, err := os.ReadFile(v.path)
	if err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	sum := sha256.Sum256(raw)
	if "sha256:"+hex.EncodeToString(sum[:]) != artifact.Digest {
		return pruntime.ArtifactVerification{}, fmt.Errorf("quote service Host rejected artifact digest")
	}
	return pruntime.ArtifactVerification{
		VerifierID: "reference.quote.e2e.hash-only", DigestVerified: true,
	}, nil
}

type quoteServiceHostProcess struct {
	calls int
	last  pruntime.Request
}

func (p *quoteServiceHostProcess) Invoke(
	ctx context.Context, command pruntime.ProcessCommand, req pruntime.Request,
) (pruntime.Response, error) {
	p.calls++
	p.last = req
	return (pruntime.StdioProcessLauncher{}).Invoke(ctx, command, req)
}

// This is a single E2E path: candidate Contract selection and distinct Shape
// trust/conformance evidence -> request Shape -> Host ServiceCarrier binding ->
// existing product Runtime verification / real external process -> result
// Shape. The candidate's trust evidence is SYNTHETIC and test-local.
func TestQuoteCandidateContractRunsThroughProductServiceCarrierAndRealHost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference StdioProcessLauncher restricts whitespace in executable paths")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	binary := filepath.Join(t.TempDir(), "quote-service-host")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./plugins/reference-order-quote")
	build.Dir = filepath.Join("..", "..")
	build.Env = append(os.Environ(),
		"GOENV=off", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local",
	)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build independent quote plugin: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(hash[:])

	ignoredStubCalls := 0
	contract, provider, impl := quoteCandidateRegistryFixture(t, &ignoredStubCalls)
	provider.ArtifactDigest = digest
	impl.Provider = provider
	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
	}
	// Publisher and conformance are independent gates: before adding the
	// synthetic conformance evidence, the resolver must deny the provider.
	if _, err := resolveCapability(input); err == nil ||
		!strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("candidate selected without separate conformance evidence: %v", err)
	}
	input.Evidence = []ConformanceEvidence{
		evidenceFor(provider, contract, "synthetic-reference-only-conformance"),
	}
	resolved, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Provider != provider ||
		resolved.ContractFingerprint != fingerprint(contract) ||
		resolved.EvidenceFingerprint == "" {
		t.Fatalf("candidate resolver did not preserve Contract/Provider evidence: %+v", resolved)
	}

	identity := plugin.Identity{
		ID: core.MustCanonicalID(resolved.Provider.PluginID),
		Version: resolved.Provider.PluginVersion,
	}
	selection := pruntime.ServiceProviderSelection{
		ContractID:          resolved.Contract.ID,
		ContractVersion:     resolved.Contract.Version,
		ContractFingerprint: resolved.ContractFingerprint,
		Scope:               string(resolved.Scope),
		Provider:            identity,
		ImplementationID:    resolved.Provider.ImplementationID,
		ArtifactDigest:      resolved.Provider.ArtifactDigest,
	}
	carrierID := core.MustCanonicalID("transport.order_quote_carrier")
	descriptor := quoteLegacyFingerprint("reference-order-quote.transport-carrier/v1")
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "reference.test-only-plugin-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin:                identity,
			DescriptorFingerprint: descriptor,
			Boundary:              pruntime.BoundaryProcess,
			Trust:                 pruntime.TrustExternalProcess,
			Command:               binary,
			SandboxPolicy:         plugin.SandboxNone,
			Artifact:              plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity,
			Capability: plugin.Capability{
				Kind: plugin.CapabilityContractQueryExecutor,
				ID:   carrierID,
			},
			Boundary: pruntime.BoundaryProcess,
		}},
		ServiceBindings: []pruntime.ServiceCarrierBinding{{
			ServiceProviderSelection: selection,
			CarrierKind:              plugin.CapabilityContractQueryExecutor,
			CarrierID:                carrierID,
			Operation:                "generic-dispatch-v0",
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	verifier := &quoteServiceHostHashVerifier{path: binary}
	processes := &quoteServiceHostProcess{}
	host := pruntime.Registry{Processes: processes, Verifier: verifier}
	var hostInvocation pruntime.ServiceCarrierInvocation
	impl.AuthorityLimits = []string{
		"candidate-publisher-trust-is-test-only",
		"candidate-provider-conformance-is-test-only",
		"product-host-binding-does-not-attest-Contract-publication",
		"hash-verifier-does-not-attest-publisher-or-signature",
		"legacy-carrier-is-not-semantic-contract-ABI",
	}
	impl.Invoke = func(ctx context.Context, request []byte) ([]byte, error) {
		return quoteLegacyProcessDispatchUsing(
			ctx, provider, request,
			func(ctx context.Context, carrierReq pruntime.Request) (pruntime.Response, error) {
				r, err := host.InvokeSelectedServiceCarrier(
					ctx, plan, selection, carrierReq.RequestID,
					json.RawMessage(carrierReq.Payload),
				)
				if err != nil {
					return pruntime.Response{}, err
				}
				hostInvocation = r
				return r.Runtime.Response, nil
			},
		)
	}
	// The existing reference Shape Registry uses deliberately synthetic
	// publisher grants; no production grant is created or stored.
	registry := newInvocationRegistryForTest(t, []InvocationImplementation{impl})
	request := []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250},{"sku":"BOOK","quantity":2,"unitPriceMinor":400}]}`)

	t.Run("request shape is rejected before Host and executable", func(t *testing.T) {
		malformed := []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":"three","unitPriceMinor":250}]}`)
		if _, err := registry.ResolveAndInvoke(ctx, input, malformed); err == nil ||
			!strings.Contains(err.Error(), "request does not conform") ||
			verifier.calls != 0 || processes.calls != 0 {
			t.Fatalf("invalid Contract request reached Host: err=%v verifier=%d process=%d",
				err, verifier.calls, processes.calls)
		}
	})
	t.Run("unbound service cannot execute despite conformant provider", func(t *testing.T) {
		previous := plan
		plan.ServiceBindings = nil
		plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		defer func() { plan = previous }()
		if _, err := registry.ResolveAndInvoke(ctx, input, request); err == nil ||
			!strings.Contains(err.Error(), "no Host binding") ||
			verifier.calls != 0 || processes.calls != 0 {
			t.Fatalf("unbound selected Contract reached Host process: %v", err)
		}
	})
	t.Run("Host verifier withdrawal fails closed", func(t *testing.T) {
		previous := host
		host.Verifier = nil
		defer func() { host = previous }()
		if _, err := registry.ResolveAndInvoke(ctx, input, request); err == nil ||
			!strings.Contains(err.Error(), "no artifact verifier") ||
			verifier.calls != 0 || processes.calls != 0 {
			t.Fatalf("selected Contract bypassed missing verifier: %v", err)
		}
	})
	t.Run("selected physical artifact differs from installed binary", func(t *testing.T) {
		previous := plan
		plan.Plugins = append([]pruntime.PluginRuntime(nil), previous.Plugins...)
		plan.Plugins[0].Artifact.Digest = quoteLegacyFingerprint("other-binary")
		plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		defer func() { plan = previous }()
		if _, err := registry.ResolveAndInvoke(ctx, input, request); err == nil ||
			!strings.Contains(err.Error(), "artifact differs from Environment Plan") ||
			verifier.calls != 0 || processes.calls != 0 {
			t.Fatalf("unselected artifact reached Host process: %v", err)
		}
	})
	execution, err := registry.ResolveAndInvoke(ctx, input, request)
	if err != nil {
		t.Fatal(err)
	}
	var quote struct {
		Currency      string `json:"currency"`
		LineCount     int    `json:"lineCount"`
		SubtotalMinor int64  `json:"subtotalMinor"`
	}
	if err := json.Unmarshal(execution.ResultPayload, &quote); err != nil {
		t.Fatal(err)
	}
	if quote.Currency != "JPY" || quote.LineCount != 2 || quote.SubtotalMinor != 1550 ||
		ignoredStubCalls != 0 || processes.calls != 1 || verifier.calls != 1 {
		t.Fatalf("business computation did not pass both Contract and Host: quote=%+v stub=%d child=%d verifier=%d",
			quote, ignoredStubCalls, processes.calls, verifier.calls)
	}
	if execution.Resolution.Provider != provider ||
		execution.RequestShapeTrustQualificationFingerprint == "" ||
		execution.ResultShapeTrustQualificationFingerprint == "" ||
		hostInvocation.Selection != selection ||
		hostInvocation.Runtime.Provenance.Plugin != identity ||
		hostInvocation.Runtime.Provenance.Artifact.Digest != digest ||
		hostInvocation.Runtime.Provenance.Verification == nil ||
		!hostInvocation.Runtime.Provenance.Verification.DigestVerified ||
		hostInvocation.Runtime.Provenance.Verification.SignatureVerified ||
		hostInvocation.Runtime.Provenance.Verification.ProvenanceChecked ||
		len(hostInvocation.AuthorityLimits) == 0 ||
		processes.last.Operation != "generic-dispatch-v0" {
		t.Fatalf("the two separate evidence records were conflated: contract=%+v host=%+v",
			execution, hostInvocation)
	}
	for _, limit := range impl.AuthorityLimits {
		if !containsString(execution.AuthorityLimits, limit) {
			t.Fatalf("synthetic trust limitation %s was concealed", limit)
		}
	}
}
