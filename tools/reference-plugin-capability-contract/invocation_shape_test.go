package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestReferenceInvocationShapesShareCanonicalIssuer(t *testing.T) {
	shapes := []InvocationShapeArtifact{
		sourceAdapterRequestShapeArtifact(),
		sourceAdapterResultShapeArtifact(),
		contractQueryRequestShapeArtifact(),
		contractQueryResultShapeArtifact(),
	}
	want := shapes[0].Issuer
	if want.ID == "" || want.Version == "" {
		t.Fatalf("canonical issuer is incomplete: %+v", want)
	}
	for _, shape := range shapes {
		if shape.Issuer != want {
			t.Fatalf("shape %s issuer=%+v want=%+v", shape.ID, shape.Issuer, want)
		}
		if invocationShapeFingerprint(shape) == "" {
			t.Fatalf("shape %s fingerprint is empty", shape.ID)
		}
	}
}

func TestInvocationShapeCanonicalizationIsVersioned(t *testing.T) {
	artifact := sourceAdapterRequestShapeArtifact()
	changed := artifact
	changed.Canonicalization = "tei.reference.invocation-shape-canonicalization/v1"
	if invocationShapeFingerprint(artifact) == invocationShapeFingerprint(changed) {
		t.Fatal("canonicalization revision change did not change shape fingerprint")
	}
	if err := validateInvocationShapeArtifact(changed); err == nil ||
		!strings.Contains(err.Error(), "canonicalization") {
		t.Fatalf("unsupported canonicalization was accepted: %v", err)
	}
}

func TestInvocationShapeFingerprintNormalizesRequiredOrdering(t *testing.T) {
	left := sourceAdapterRequestShapeArtifact()
	right := sourceAdapterRequestShapeArtifact()
	right.Root.Required = []string{
		"TranscriptionContract",
		"Format",
		"SourceBytes",
	}
	if got, want := invocationShapeFingerprint(left), invocationShapeFingerprint(right); got != want {
		t.Fatalf("required ordering changed shape fingerprint: left=%s right=%s", got, want)
	}
}

func TestInvocationShapeFingerprintBindsIssuer(t *testing.T) {
	left := sourceAdapterRequestShapeArtifact()
	right := sourceAdapterRequestShapeArtifact()
	right.Issuer = InvocationShapeIssuer{
		ID:      left.Issuer.ID,
		Version: "0.2.0",
	}
	if invocationShapeFingerprint(left) == invocationShapeFingerprint(right) {
		t.Fatal("shape issuer change did not change fingerprint")
	}
}

func TestInvocationShapeRejectsMissingAndUnknownRequestFields(t *testing.T) {
	artifact := sourceAdapterRequestShapeArtifact()
	for name, payload := range map[string][]byte{
		"missing": []byte("{\"Format\":\"xead\",\"SourceBytes\":\"PHN5c3RlbS8+\"}"),
		"unknown": []byte("{\"Format\":\"xead\",\"SourceBytes\":\"PHN5c3RlbS8+\",\"TranscriptionContract\":\"e30=\",\"Unexpected\":true}"),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateInvocationPayload(artifact, payload); err == nil {
				t.Fatalf("invalid request shape was accepted: %s", payload)
			}
		})
	}
}

func TestInvocationShapeRejectsUnexpectedResultField(t *testing.T) {
	artifact := sourceAdapterResultShapeArtifact()
	payload := []byte("{\"document\":{},\"candidates\":[],\"relationalFacts\":{},\"relationalResolutionStatus\":\"resolved\",\"relationalResolutionPolicyRevision\":\"v1\",\"authorityLimits\":[],\"unexpected\":true}")
	if err := validateInvocationPayload(artifact, payload); err == nil ||
		!strings.Contains(err.Error(), "undeclared property") {
		t.Fatalf("unexpected result field was accepted: %v", err)
	}
}

func TestInvocationRegistryRejectsShapeArtifactDriftAtRegistration(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract, ScopeAuthoringTool, "reference.compat.xead.source", referenceXeadReaderImplementationID,
	)
	typed, err := ReferenceXeadSourceAdapterImplementation(provider)
	if err != nil { t.Fatal(err) }
	implementation, err := sourceAdapterInvocationImplementation(typed)
	if err != nil { t.Fatal(err) }
	implementation.RequestShapeArtifact.Root.Required = append(
		implementation.RequestShapeArtifact.Root.Required, "NewRequiredField",
	)
	_, err = NewInvocationRegistry([]InvocationImplementation{implementation})
	if err == nil || !strings.Contains(err.Error(), "artifact does not match declaration") {
		t.Fatalf("mutated shape artifact was registered: %v", err)
	}
}

func TestGenericInvocationRejectsShapeIssuerDriftBeforeDomainCall(t *testing.T) {
	contract := sourceAdapterContract()
	contract.ShapeIssuer = &InvocationShapeIssuer{ID: contract.ShapeIssuer.ID, Version: "0.2.0"}
	provider := sourceAdapterProvider(
		contract, ScopeAuthoringTool, "reference.compat.xead.source", referenceXeadReaderImplementationID,
	)
	typed, err := ReferenceXeadSourceAdapterImplementation(provider)
	if err != nil { t.Fatal(err) }
	implementation, err := sourceAdapterInvocationImplementation(typed)
	if err != nil { t.Fatal(err) }
	registry, err := NewInvocationRegistry([]InvocationImplementation{implementation})
	if err != nil { t.Fatal(err) }
	payload, err := json.Marshal(SourceAdapterRequest{
		Format: "xead",
		SourceBytes: []byte("<System FormatVersion=\"1.2\"/>"),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err != nil { t.Fatal(err) }
	_, err = registry.ResolveAndInvoke(context.Background(), sourceAdapterResolveInput(contract, provider), payload)
	if err == nil || !strings.Contains(err.Error(), "shape issuer") {
		t.Fatalf("shape issuer drift reached domain implementation: %v", err)
	}
}

func TestGenericInvocationValidatesRequestBeforeInvoker(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.request-validation", "recording.reader.v1")
	calls := 0
	implementation := InvocationImplementation{
		Provider: provider,
		RequestShape: sourceAdapterRequestShape,
		RequestShapeFingerprint: sourceAdapterRequestShapeFingerprint,
		RequestShapeArtifact: sourceAdapterRequestShapeArtifact(),
		ResultShape: sourceAdapterResultShape,
		ResultShapeFingerprint: sourceAdapterResultShapeFingerprint,
		ResultShapeArtifact: sourceAdapterResultShapeArtifact(),
		PayloadCodec: referenceJSONPayloadCodec,
		Invoke: func(context.Context, []byte) ([]byte, error) { calls++; return nil, nil },
	}
	registry, err := NewInvocationRegistry([]InvocationImplementation{implementation})
	if err != nil { t.Fatal(err) }
	_, err = registry.ResolveAndInvoke(context.Background(), ResolveInput{
		Scope: ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts: []Contract{contract},
		Providers: []Provider{provider},
		Evidence: []ConformanceEvidence{evidenceFor(provider, contract, "request-shape-conformance")},
	}, []byte("{\"Format\":\"xead\"}"))
	if err == nil || !strings.Contains(err.Error(), "capability request does not conform") {
		t.Fatalf("invalid request reached invoker: err=%v calls=%d", err, calls)
	}
	if calls != 0 { t.Fatalf("invalid request invoked domain implementation %d times", calls) }
}

func TestGenericInvocationValidatesResultAfterInvoker(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.result-validation", "recording.reader.v1")
	calls := 0
	implementation := InvocationImplementation{
		Provider: provider,
		RequestShape: sourceAdapterRequestShape,
		RequestShapeFingerprint: sourceAdapterRequestShapeFingerprint,
		RequestShapeArtifact: sourceAdapterRequestShapeArtifact(),
		ResultShape: sourceAdapterResultShape,
		ResultShapeFingerprint: sourceAdapterResultShapeFingerprint,
		ResultShapeArtifact: sourceAdapterResultShapeArtifact(),
		PayloadCodec: referenceJSONPayloadCodec,
		Invoke: func(context.Context, []byte) ([]byte, error) { calls++; return []byte("{\"unexpected\":true}"), nil },
	}
	registry, err := NewInvocationRegistry([]InvocationImplementation{implementation})
	if err != nil { t.Fatal(err) }
	requestPayload, err := json.Marshal(SourceAdapterRequest{
		Format: "xead",
		SourceBytes: []byte("<System FormatVersion=\"1.2\"/>"),
		TranscriptionContract: []byte("{}"),
	})
	if err != nil { t.Fatal(err) }
	_, err = registry.ResolveAndInvoke(context.Background(), ResolveInput{
		Scope: ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts: []Contract{contract},
		Providers: []Provider{provider},
		Evidence: []ConformanceEvidence{evidenceFor(provider, contract, "result-shape-conformance")},
	}, requestPayload)
	if err == nil || !strings.Contains(err.Error(), "capability result does not conform") {
		t.Fatalf("invalid result escaped generic boundary: err=%v calls=%d", err, calls)
	}
	if calls != 1 { t.Fatalf("valid request did not invoke domain implementation exactly once: %d", calls) }
}


func TestGenericInvocationRejectsMutatingEffectBeforeInvoker(t *testing.T) {
	contract := sourceAdapterContract()
	contract.InvocationEffect = "mutating"
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.mutating-pressure",
		"mutating.reader.v1",
	)
	calls := 0
	implementation := InvocationImplementation{
		Provider:                provider,
		RequestShape:            sourceAdapterRequestShape,
		RequestShapeFingerprint: sourceAdapterRequestShapeFingerprint,
		RequestShapeArtifact:    sourceAdapterRequestShapeArtifact(),
		ResultShape:             sourceAdapterResultShape,
		ResultShapeFingerprint:  sourceAdapterResultShapeFingerprint,
		ResultShapeArtifact:     sourceAdapterResultShapeArtifact(),
		PayloadCodec:            referenceJSONPayloadCodec,
		Invoke: func(context.Context, []byte) ([]byte, error) {
			calls++
			return []byte("{}"), nil
		},
	}
	registry, err := NewInvocationRegistry([]InvocationImplementation{implementation})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(SourceAdapterRequest{
		Format:                "xead",
		SourceBytes:           []byte("<System FormatVersion=\"1.2\"/>"),
		TranscriptionContract: []byte("{}"),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.ResolveAndInvoke(
		context.Background(),
		ResolveInput{
			Scope:       ScopeAuthoringTool,
			Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
			Contracts:   []Contract{contract},
			Providers:   []Provider{provider},
			Evidence: []ConformanceEvidence{
				evidenceFor(provider, contract, "mutating-pressure-conformance"),
			},
		},
		payload,
	)
	if err == nil || !strings.Contains(err.Error(), "invocation effect") {
		t.Fatalf("mutating invocation was not rejected: %v", err)
	}
	if calls != 0 {
		t.Fatalf("mutating implementation was invoked %d times", calls)
	}
}
