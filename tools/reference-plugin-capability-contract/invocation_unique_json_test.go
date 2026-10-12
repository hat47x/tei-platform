package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// These are reference-shape checks, not acceptance of the candidate quote
// contract. The existing quote implementation rejects duplicate JSON keys;
// a common Service Contract boundary must not interpret them differently.
func TestCandidateQuoteShapeRejectsDuplicateJSONMembers(t *testing.T) {
	request := readQuoteCandidateShape(t, "request-v0.json")
	result := readQuoteCandidateShape(t, "result-v0.json")
	cases := []struct {
		name    string
		shape   InvocationShapeArtifact
		payload string
		valid   bool
	}{
		{
			name: "distinct lines can reuse the same member names",
			shape: request,
			payload: `{"currency":"JPY","lines":[{"sku":"PEN","quantity":1,"unitPriceMinor":250},{"sku":"BOOK","quantity":2,"unitPriceMinor":400}]}`,
			valid: true,
		},
		{
			name: "duplicate top-level member",
			shape: request,
			payload: `{"currency":"JPY","currency":"USD","lines":[{"sku":"PEN","quantity":1,"unitPriceMinor":250}]}`,
		},
		{
			name: "escaped duplicate top-level member",
			shape: request,
			payload: `{"currency":"JPY","\u0063urrency":"USD","lines":[{"sku":"PEN","quantity":1,"unitPriceMinor":250}]}`,
		},
		{
			name: "duplicate nested member in array item",
			shape: request,
			payload: `{"currency":"JPY","lines":[{"sku":"PEN","quantity":1,"quantity":2,"unitPriceMinor":250}]}`,
		},
		{
			name: "duplicate result member",
			shape: result,
			payload: `{"currency":"JPY","lineCount":1,"subtotalMinor":250,"subtotalMinor":500}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInvocationPayload(tc.shape, []byte(tc.payload))
			if tc.valid {
				if err != nil {
					t.Fatalf("valid JSON shape rejected: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "duplicate JSON property") {
				t.Fatalf("ambiguous JSON shape accepted: %v", err)
			}
		})
	}
}

func TestGenericInvocationRejectsDuplicateJSONBeforeAndAfterDomainCall(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract, ScopeAuthoringTool, "reference.unique-json", "unique.reader.v1",
	)
	input := sourceAdapterResolveInput(contract, provider)
	calls := 0
	result := []byte(`{"document":{},"document":{}}`)
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
			return result, nil
		},
	}
	registry := newInvocationRegistryForTest(t, []InvocationImplementation{implementation})
	ambiguousRequest := []byte(`{"Format":"xead","Format":"xeaf","SourceBytes":"","TranscriptionContract":""}`)
	if _, err := registry.ResolveAndInvoke(context.Background(), input, ambiguousRequest); err == nil ||
		!strings.Contains(err.Error(), "duplicate JSON property") || calls != 0 {
		t.Fatalf("ambiguous request reached domain code: calls=%d err=%v", calls, err)
	}
	validRequest, err := json.Marshal(SourceAdapterRequest{
		Format: "xead", SourceBytes: []byte("<System FormatVersion=\"1.2\"/>"),
		TranscriptionContract: []byte("{}"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ResolveAndInvoke(context.Background(), input, validRequest); err == nil ||
		!strings.Contains(err.Error(), "duplicate JSON property") || calls != 1 {
		t.Fatalf("ambiguous result passed shared boundary: calls=%d err=%v", calls, err)
	}
}
