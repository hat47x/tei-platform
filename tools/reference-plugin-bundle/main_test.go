package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestBundleDispatchesOnlyKnownPluginIdentities(t *testing.T) {
	cases := []struct {
		name, pluginID, capabilityID, wantMessage, wantStatus string
	}{
		{"alpha", "tei.reference.bundle.alpha", "process.bundle.alpha", "alpha accepted intent", "ok"},
		{"beta", "tei.reference.bundle.beta", "process.bundle.beta", "beta accepted intent", "ok"},
		{"unknown-identity", "tei.reference.bundle.unknown", "process.bundle.alpha", "", "error"},
		{"cross-capability", "tei.reference.bundle.alpha", "process.bundle.beta", "", "error"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			wire := request{
				ProtocolVersion: 1, RequestID: "request",
				Plugin:       identity{ID: tt.pluginID, Version: "1.0.0"},
				CapabilityID: tt.capabilityID, DescriptorFingerprint: "descriptor",
				Operation: "execute",
				Payload:   json.RawMessage(`{"id":"intent","actionId":"sales_expand/apply","idempotencyKey":"key"}`),
			}
			input, err := json.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := serve(bytes.NewReader(input), &output, []string{referenceMarker}); err != nil {
				t.Fatal(err)
			}
			var got response
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Status != tt.wantStatus || got.Plugin != wire.Plugin ||
				got.CapabilityID != wire.CapabilityID || got.RequestID != wire.RequestID {
				t.Fatalf("response lost request identity: %+v", got)
			}
			if got.Status == "error" {
				if got.Error != "unknown plugin identity or capability" || len(got.Payload) != 0 {
					t.Fatalf("unexpected rejection: %+v", got)
				}
				return
			}
			var ack acknowledgment
			if err := json.Unmarshal(got.Payload, &ack); err != nil {
				t.Fatal(err)
			}
			if ack.Outcome != "accepted" || ack.Message != tt.wantMessage ||
				ack.ActionID != "sales_expand/apply" || ack.IdempotencyKey != "key" {
				t.Fatalf("unexpected bundle acknowledgment: %+v", ack)
			}
		})
	}
}

func TestBundleDoesNotAcceptAmbiguousOrIncompleteInput(t *testing.T) {
	base := request{
		ProtocolVersion: 1, RequestID: "request",
		Plugin:       identity{ID: "tei.reference.bundle.alpha", Version: "1.0.0"},
		CapabilityID: "process.bundle.alpha", DescriptorFingerprint: "descriptor",
		Operation: "execute", Payload: json.RawMessage(`{"id":"intent","actionId":"sales_expand/apply","idempotencyKey":"key"}`),
	}
	wire, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"{", string(wire) + "\n" + string(wire)} {
		var output bytes.Buffer
		if err := serve(strings.NewReader(input), &output, []string{referenceMarker}); err == nil {
			t.Fatal("expected malformed/multiple JSON inputs to be rejected")
		}
		if output.Len() != 0 {
			t.Fatal("malformed input must not produce an acknowledgment")
		}
	}
	for _, args := range [][]string{nil, {"--untrusted-argument"}} {
		var output bytes.Buffer
		if err := serve(bytes.NewReader(wire), &output, args); err != nil {
			t.Fatal(err)
		}
		var got response
		if err := json.Unmarshal(output.Bytes(), &got); err != nil || got.Status != "error" {
			t.Fatalf("unverified argv was accepted: %+v, %v", got, err)
		}
	}
}
