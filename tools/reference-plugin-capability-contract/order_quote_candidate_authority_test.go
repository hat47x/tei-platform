package main

import (
	"context"
	"strings"
	"testing"
)

// The candidate artifacts are unaccepted. All evidence and publisher grants
// below are created IN MEMORY by existing test helpers, not read from the
// candidate directory and not usable as production acceptance records.
func quoteCandidateRegistryFixture(t *testing.T, calls *int) (Contract, Provider, InvocationImplementation) {
	t.Helper()
	request := readQuoteCandidateShape(t, "request-v0.json")
	result := readQuoteCandidateShape(t, "result-v0.json")
	issuer := request.Issuer
	if issuer != result.Issuer {
		t.Fatal("candidate request and result claim different publishers")
	}
	contract := Contract{
		ID:                      "reference.order.quote-lines/v1",
		Version:                 "1.0.0",
		RequestSemantics:        "integer-JPY-lines-preview",
		ResultSemantics:         "integer-JPY-subtotal-preview",
		FailureSemantics:        "invalid-input-is-not-a-quote",
		RequestShape:            request.ID,
		RequestShapeFingerprint: invocationShapeFingerprint(request),
		ResultShape:             result.ID,
		ResultShapeFingerprint:  invocationShapeFingerprint(result),
		PayloadCodec:            "application/json",
		ShapeIssuer:             &issuer,
		InvocationEffect:        invocationEffectReadOnly,
		ConformanceProfile:      "reference.order-quote.candidate/v0",
	}
	provider := sourceAdapterProvider(
		contract,
		ScopeApplicationRuntime,
		"tei.reference.order_quote",
		"order-quote.v1",
	)
	implementation := InvocationImplementation{
		Provider:                provider,
		RequestShape:            request.ID,
		RequestShapeFingerprint: invocationShapeFingerprint(request),
		RequestShapeArtifact:    request,
		ResultShape:             result.ID,
		ResultShapeFingerprint:  invocationShapeFingerprint(result),
		ResultShapeArtifact:     result,
		PayloadCodec:            "application/json",
		AuthorityLimits: []string{
			"synthetic-test-grants-are-not-real-shape-publisher-trust",
			"provider-is-a-synthetic-in-process-stub-not-an-installed-binary",
			"no-runtime-permission-or-resource-lease-was-verified",
		},
		Invoke: func(_ context.Context, _ []byte) ([]byte, error) {
			*calls++
			return []byte(`{"currency":"JPY","lineCount":1,"subtotalMinor":750}`), nil
		},
	}
	return contract, provider, implementation
}

func TestQuoteCandidateCannotRegisterWithoutSeparateShapeEvidenceAndPublisherTrust(t *testing.T) {
	calls := 0
	_, _, implementation := quoteCandidateRegistryFixture(t, &calls)
	installed := []InvocationImplementation{implementation}

	if _, err := NewInvocationRegistry(installed, nil); err == nil ||
		!strings.Contains(err.Error(), "Shape Authority Evidence") {
		t.Fatalf("candidate publisher self-assertion registered without evidence: %v", err)
	}
	evidence := invocationShapeAuthoritiesForTest(installed)
	if len(evidence) != 2 {
		t.Fatalf("test fixture did not isolate request/result evidence: %+v", evidence)
	}
	if _, err := NewInvocationRegistry(installed, evidence); err == nil ||
		!strings.Contains(err.Error(), "publisher trust") {
		t.Fatalf("self-issued evidence bypassed publisher trust: %v", err)
	}

	testGrants := invocationShapePublisherTrustForTest(t, evidence)
	if len(testGrants) != 2 {
		t.Fatalf("expected two synthetic publisher grants, got %d", len(testGrants))
	}
	rejected := append([]InvocationShapeAuthorityEvidence(nil), evidence...)
	rejected[0].ShapeFingerprint = "sha256:" + strings.Repeat("b", 64)
	rejected[0].EvidenceFingerprint = invocationShapeAuthorityEvidenceFingerprint(rejected[0])
	if _, err := NewInvocationRegistry(installed, rejected,
		invocationShapePublisherTrustForTest(t, rejected)); err == nil ||
		!strings.Contains(err.Error(), "no accepted Shape Authority Evidence") {
		t.Fatalf("shape artifact drift did not fail closed: %v", err)
	}
	if calls != 0 {
		t.Fatalf("untrusted candidate invoked implementation %d times", calls)
	}
}

func TestQuoteCandidateRequiresDistinctProviderConformanceAfterSyntheticPublisherTrust(t *testing.T) {
	calls := 0
	contract, provider, implementation := quoteCandidateRegistryFixture(t, &calls)
	installed := []InvocationImplementation{implementation}
	shapeEvidence := invocationShapeAuthoritiesForTest(installed)

	// Explicitly synthetic grants let us exercise the NEXT independent gate.
	// They do not turn reference/candidates into accepted product authority.
	registry, err := NewInvocationRegistry(installed, shapeEvidence,
		invocationShapePublisherTrustForTest(t, shapeEvidence))
	if err != nil {
		t.Fatal(err)
	}
	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
	}
	quote := []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250}]}`)

	if _, err := registry.ResolveAndInvoke(context.Background(), input, quote); err == nil ||
		!strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("synthetic publisher trust alone permitted provider invocation: %v", err)
	}
	stale := evidenceFor(provider, contract, "synthetic-provider-evidence")
	stale.ArtifactDigest = "sha256:" + strings.Repeat("c", 64)
	input.Evidence = []ConformanceEvidence{stale}
	if _, err := registry.ResolveAndInvoke(context.Background(), input, quote); err == nil ||
		!strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("wrong artifact conformance evidence permitted invocation: %v", err)
	}
	if calls != 0 {
		t.Fatalf("provider with no matching conformance was invoked %d times", calls)
	}

	input.Evidence = []ConformanceEvidence{
		evidenceFor(provider, contract, "synthetic-test-only-provider-conformance"),
	}
	invalid := []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":"three","unitPriceMinor":250}]}`)
	if _, err := registry.ResolveAndInvoke(context.Background(), input, invalid); err == nil ||
		!strings.Contains(err.Error(), "request does not conform") {
		t.Fatalf("invalid request reached candidate invoker: %v", err)
	}
	if calls != 0 {
		t.Fatal("invalid request reached synthetic implementation")
	}
	execution, err := registry.ResolveAndInvoke(context.Background(), input, quote)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 ||
		execution.Resolution.Provider != provider ||
		execution.RequestShapeAuthorityEvidenceFingerprint == "" ||
		execution.RequestShapeTrustQualificationFingerprint == "" ||
		execution.ResultShapeAuthorityEvidenceFingerprint == "" ||
		execution.ResultShapeTrustQualificationFingerprint == "" {
		t.Fatalf("test-only evidence gates or synthetic call not observed: %+v, calls=%d", execution, calls)
	}
	if !containsString(execution.AuthorityLimits,
		"no-runtime-permission-or-resource-lease-was-verified") {
		t.Fatalf("synthetic invocation concealed runtime authority limitation: %+v", execution.AuthorityLimits)
	}
}

func containsString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
