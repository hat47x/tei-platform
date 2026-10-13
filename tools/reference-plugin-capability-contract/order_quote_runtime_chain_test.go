package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// This wire DTO is confined to a test of the existing legacy stdio carrier.
// It is not a new TEI Capability ABI or a mapping from Quote to Contract Query.
type quoteChainWire struct {
	Schema                  string            `json:"schema"`
	ContractID              string            `json:"contractId"`
	ContractFingerprint     string            `json:"contractFingerprint"`
	Target                  string            `json:"target"`
	Provider                map[string]string `json:"provider"`
	RequestShapeFingerprint string            `json:"requestShapeFingerprint"`
	ResultShapeFingerprint  string            `json:"resultShapeFingerprint"`
	PayloadCodec            string            `json:"payloadCodec"`
	Payload                 []byte            `json:"payload"`
}

type quoteChainHashVerifier struct {
	path  string
	calls int
}

func (v *quoteChainHashVerifier) Verify(
	_ context.Context, _ plugin.Identity, artifact plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	bytes, err := os.ReadFile(v.path)
	if err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	sum := sha256.Sum256(bytes)
	if artifact.Digest != "sha256:"+hex.EncodeToString(sum[:]) {
		return pruntime.ArtifactVerification{}, errors.New("test-only executable digest mismatch")
	}
	return pruntime.ArtifactVerification{
		VerifierID: "reference.quote-chain.test-hash-only", DigestVerified: true,
	}, nil
}

type quoteChainLauncher struct {
	calls int
	last  pruntime.Request
}

func (p *quoteChainLauncher) Invoke(ctx context.Context, cmd pruntime.ProcessCommand, req pruntime.Request) (pruntime.Response, error) {
	p.calls++
	p.last = req
	return (pruntime.StdioProcessLauncher{}).Invoke(ctx, cmd, req)
}

func quoteChainDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// These stages are exercised in sequence, on the SAME request:
// candidate shape authority + publisher trust (synthetic test fixtures only);
// provider conformance (synthetic evidence, exact real-binary SHA-256);
// product Runtime Registry (test EnvironmentPlan, digest/permission/sandbox
// gates); actual independent child process; shape-checked business result.
// This deliberately proves no production trust grant, pinned executable
// image, signing, OS sandbox, durable effect or canonical model acceptance.
func TestCandidateQuoteShapeConformanceAndRealRuntimeProcessOneInvocation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference process requires a platform-specific executable policy")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	source, err := os.ReadFile("../../plugins/reference-order-quote/main.go")
	if err != nil {
		t.Fatal(err)
	}
	isolated := t.TempDir()
	if err := os.WriteFile(filepath.Join(isolated, "main.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(isolated, "go.mod"),
		[]byte("module example.invalid/quote-chain\n\ngo 1.23\n"), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "quote-chain")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = isolated
	build.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile independent quote provider: %v\n%s", err, output)
	}
	binaryBytes, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	digest := quoteChainDigest(binaryBytes)

	unusedCalls := 0
	contract, provider, impl := quoteCandidateRegistryFixture(t, &unusedCalls)
	provider.ArtifactDigest = digest
	impl.Provider = provider
	impl.AuthorityLimits = []string{
		"candidate-shape-and-publisher-grants-are-only-test-fixtures",
		"provider-conformance-evidence-is-only-test-fixture",
		"runtime-plan-and-image-verifier-are-test-only",
		"legacy-contract-query-kind-is-stdio-carrier-not-domain-query",
		"no-signed-publisher-provider-or-artifact-acceptance",
		"no-pinned-process-image-sandbox-or-resource-lease",
	}

	identity := plugin.Identity{
		ID: core.MustCanonicalID(provider.PluginID), Version: provider.PluginVersion,
	}
	capabilityID := core.MustCanonicalID("transport.order_quote_carrier")
	descriptor := quoteChainDigest([]byte("reference.order-quote.transport-carrier/v1"))
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "test-only-quote-chain-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity, DescriptorFingerprint: descriptor,
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: binary, SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity,
			Capability: plugin.Capability{
				Kind: plugin.CapabilityContractQueryExecutor, ID: capabilityID,
			},
			Boundary: pruntime.BoundaryProcess,
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	verifier := &quoteChainHashVerifier{path: binary}
	launcher := &quoteChainLauncher{}
	host := pruntime.Registry{Processes: launcher, Verifier: verifier}
	var provenance pruntime.InvocationProvenance
	// Faults are injected AFTER the real child has returned. They exercise
	// consumer-side proof boundaries, not the child process implementation.
	var corruptTransportIdentity bool
	var alterBusinessResult func([]byte) []byte
	sequence := 0
	impl.Invoke = func(ctx context.Context, quote []byte) ([]byte, error) {
		sequence++
		wire := quoteChainWire{
			Schema:              "tei.reference.capability-transport-dispatch/v0",
			ContractID:          contract.ID,
			ContractFingerprint: quoteChainDigest([]byte("reference.order.quote-lines.contract/v1")),
			Target:              "reference/order_entry",
			Provider: map[string]string{
				"pluginId": provider.PluginID, "pluginVersion": provider.PluginVersion,
				"implementationId": provider.ImplementationID,
				"artifactDigest":   provider.ArtifactDigest,
			},
			RequestShapeFingerprint: quoteChainDigest([]byte("reference.order.quote-lines.request/v1")),
			ResultShapeFingerprint:  quoteChainDigest([]byte("reference.order.quote-lines.result/v1")),
			PayloadCodec:            "application/json",
			Payload:                 quote,
		}
		// The inner transport retains the pre-existing legacy reference
		// fingerprints. The OUTER candidate shapes are validated by the
		// InvocationRegistry, not asserted as accepted by the child binary.
		inv, err := host.InvokeWithProvenance(
			ctx, plan, plugin.CapabilityContractQueryExecutor,
			capabilityID, "generic-dispatch-v0",
			fmt.Sprintf("quote-chain-%d", sequence), wire,
		)
		if err != nil {
			return nil, err
		}
		provenance = inv.Provenance
		var response quoteChainWire
		if err := decodeReferenceJSONExact(inv.Response.Payload, &response); err != nil {
			return nil, err
		}
		expected := wire
		expected.Schema = "tei.reference.capability-transport-result/v0"
		expected.Payload = nil
		out := response.Payload
		response.Payload = nil
		if corruptTransportIdentity {
			response.ContractFingerprint = quoteChainDigest([]byte("forged-transport-contract"))
		}
		if !reflect.DeepEqual(expected, response) {
			return nil, errors.New("real process transport identity drifted")
		}
		if alterBusinessResult != nil {
			return alterBusinessResult(out), nil
		}
		return out, nil
	}
	shapeEvidence := invocationShapeAuthoritiesForTest([]InvocationImplementation{impl})
	generic, err := NewInvocationRegistry(
		[]InvocationImplementation{impl},
		shapeEvidence,
		invocationShapePublisherTrustForTest(t, shapeEvidence),
	)
	if err != nil {
		t.Fatal(err)
	}
	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
	}
	payload := []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250},{"sku":"BOOK","quantity":2,"unitPriceMinor":400}]}`)
	attempt := func(t *testing.T, name string, want string, mutate func()) {
		t.Helper()
		before := launcher.calls
		mutate()
		_, err := generic.ResolveAndInvoke(ctx, input, payload)
		if err == nil || !strings.Contains(err.Error(), want) || launcher.calls != before {
			t.Fatalf("%s: want pre-launch rejection %q; err=%v launches=%d",
				name, want, err, launcher.calls-before)
		}
	}

	t.Run("candidate shape grants do not imply provider conformance", func(t *testing.T) {
		attempt(t, "no evidence", "no conformant provider", func() { input.Evidence = nil })
	})
	t.Run("incorrect provider artifact evidence stops before Runtime", func(t *testing.T) {
		e := evidenceFor(provider, contract, "synthetic-test-only")
		e.ArtifactDigest = quoteChainDigest([]byte("other artifact"))
		attempt(t, "wrong evidence", "no conformant provider", func() { input.Evidence = []ConformanceEvidence{e} })
	})
	input.Evidence = []ConformanceEvidence{evidenceFor(provider, contract, "synthetic-test-only")}
	t.Run("outer request shape violation prevents Runtime", func(t *testing.T) {
		bad := payload
		payload = []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":"3","unitPriceMinor":250}]}`)
		defer func() { payload = bad }()
		attempt(t, "bad input", "request does not conform", func() {})
	})
	t.Run("missing Runtime verifier prevents execution", func(t *testing.T) {
		before := host
		defer func() { host = before }()
		attempt(t, "no verifier", "no artifact verifier", func() { host.Verifier = nil })
	})
	t.Run("Runtime digest mismatch prevents process", func(t *testing.T) {
		before := plan
		defer func() { plan = before }()
		attempt(t, "wrong digest", "digest mismatch", func() {
			plan.Plugins = append([]pruntime.PluginRuntime(nil), plan.Plugins...)
			plan.Plugins[0].Artifact.Digest = quoteChainDigest([]byte("incorrect bytes"))
			plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		})
	})
	t.Run("missing Runtime capability prevents process", func(t *testing.T) {
		before := plan
		defer func() { plan = before }()
		attempt(t, "no capability", "not uniquely resolved", func() {
			plan.Capabilities = nil
			plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		})
	})
	t.Run("Runtime permission not granted prevents process", func(t *testing.T) {
		before := plan
		defer func() { plan = before }()
		attempt(t, "no network grant", "requires permission", func() {
			plan.Capabilities = append([]pruntime.CapabilityRuntime(nil), plan.Capabilities...)
			plan.Capabilities[0].Capability.Permissions = []permission.Spec{{
				Kind: permission.KindNetwork, Access: permission.AccessConnect,
				Scope: "database.primary",
			}}
			plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		})
	})
	t.Run("missing required sandbox prevents process", func(t *testing.T) {
		before := plan
		defer func() { plan = before }()
		attempt(t, "required sandbox", "requires sandbox launcher", func() {
			plan.Plugins = append([]pruntime.PluginRuntime(nil), plan.Plugins...)
			plan.Plugins[0].SandboxPolicy = plugin.SandboxRequired
			plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		})
	})

	// A process can execute successfully while returning a response that is
	// unacceptable to the selected contract. The outer InvocationRegistry
	// must reject the result instead of treating Runtime launch success as
	// proof of business-conformant output. No completed InvocationExecution
	// (and therefore no success provenance) may escape.
	t.Run("tampered transport identity rejected after real child execution", func(t *testing.T) {
		before := launcher.calls
		corruptTransportIdentity = true
		defer func() { corruptTransportIdentity = false }()
		got, err := generic.ResolveAndInvoke(ctx, input, payload)
		if err == nil || !strings.Contains(err.Error(), "real process transport identity drifted") ||
			launcher.calls != before+1 || !reflect.DeepEqual(got, InvocationExecution{}) {
			t.Fatalf("invalid transport identity was accepted: err=%v launches=%d result=%+v",
				err, launcher.calls-before, got)
		}
	})
	for _, tc := range []struct {
		name   string
		result string
	}{
		{"wrong subtotal type", `{"currency":"JPY","lineCount":2,"subtotalMinor":"1550"}`},
		{"missing currency", `{"lineCount":2,"subtotalMinor":1550}`},
		{"unknown submitted effect", `{"currency":"JPY","lineCount":2,"subtotalMinor":1550,"submitted":true}`},
	} {
		t.Run("real process response rejected: "+tc.name, func(t *testing.T) {
			before := launcher.calls
			alterBusinessResult = func([]byte) []byte { return []byte(tc.result) }
			defer func() { alterBusinessResult = nil }()
			got, err := generic.ResolveAndInvoke(ctx, input, payload)
			if err == nil || !strings.Contains(err.Error(), "result does not conform") ||
				launcher.calls != before+1 || !reflect.DeepEqual(got, InvocationExecution{}) {
				t.Fatalf("invalid business response was accepted: err=%v launches=%d result=%+v",
					err, launcher.calls-before, got)
			}
		})
	}
	// A structurally valid response may still contain a wrong subtotal.
	// The current shape is only structural; this test MUST NOT claim that
	// provider conformance verifies arithmetic or result business semantics.
	t.Run("structural shape alone cannot prove correct subtotal", func(t *testing.T) {
		before := launcher.calls
		alterBusinessResult = func([]byte) []byte {
			return []byte(`{"currency":"JPY","lineCount":2,"subtotalMinor":1}`)
		}
		defer func() { alterBusinessResult = nil }()
		got, err := generic.ResolveAndInvoke(ctx, input, payload)
		if err != nil || launcher.calls != before+1 ||
			!strings.Contains(string(got.ResultPayload), `"subtotalMinor":1`) {
			t.Fatalf("shape unexpectedly proved business arithmetic: err=%v launches=%d result=%+v",
				err, launcher.calls-before, got)
		}
	})

	t.Run("synthetic outer evidence plus real Runtime yields actual quote", func(t *testing.T) {
		before := launcher.calls
		got, err := generic.ResolveAndInvoke(ctx, input, payload)
		if err != nil {
			t.Fatal(err)
		}
		var quote struct {
			Currency      string `json:"currency"`
			LineCount     int    `json:"lineCount"`
			SubtotalMinor int64  `json:"subtotalMinor"`
		}
		if err := decodeReferenceJSONExact(got.ResultPayload, &quote); err != nil {
			t.Fatal(err)
		}
		if quote.Currency != "JPY" || quote.LineCount != 2 || quote.SubtotalMinor != 1550 ||
			launcher.calls != before+1 || unusedCalls != 0 {
			t.Fatalf("expected real child calculation, not stub: %+v launcher=%d stub=%d",
				quote, launcher.calls-before, unusedCalls)
		}
		if got.Resolution.Provider.ArtifactDigest != digest ||
			got.RequestShapeAuthorityEvidenceFingerprint == "" ||
			got.ResultShapeAuthorityEvidenceFingerprint == "" ||
			got.RequestShapeTrustQualificationFingerprint == "" ||
			got.ResultShapeTrustQualificationFingerprint == "" ||
			got.Resolution.EvidenceFingerprint == "" ||
			!containsString(got.AuthorityLimits, "candidate-shape-and-publisher-grants-are-only-test-fixtures") {
			t.Fatalf("outer synthetic provenance/limits dropped: %+v", got)
		}
		if provenance.Plugin != identity ||
			provenance.Artifact.Digest != digest ||
			provenance.PluginLockFingerprint != plan.PluginLockFingerprint ||
			provenance.CapabilityID != capabilityID ||
			provenance.Boundary != pruntime.BoundaryProcess ||
			provenance.Verification == nil ||
			!provenance.Verification.DigestVerified ||
			provenance.Verification.SignatureVerified ||
			provenance.Verification.ProvenanceChecked ||
			provenance.Sandboxed ||
			provenance.ResourceLeaseID != "" {
			t.Fatalf("Runtime provenance falsely implies production authority: %+v", provenance)
		}
		if launcher.last.PluginLockFingerprint != plan.PluginLockFingerprint ||
			launcher.last.CapabilityID != capabilityID {
			t.Fatalf("child did not receive correct Runtime carrier identity: %+v", launcher.last)
		}
		for _, forbidden := range []string{"Bearer ", "principalId", "sessionId", "hostApproval"} {
			if strings.Contains(string(launcher.last.Payload), forbidden) {
				t.Fatalf("Host credential/approval leaked to child: %q", forbidden)
			}
		}
	})
}
