package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const exampleWireRequest = `{"protocolVersion":1,"requestId":"request-one","plugin":{"id":"tei.process.reference","version":"1.0.0"},"capabilityId":"process.reference","descriptorFingerprint":"reference-process-descriptor","operation":"execute","payload":{"id":"command-one","actionId":"sales_expand/apply","idempotencyKey":"key-one"}}`

func TestReferenceProcessAcceptsCurrentWire(t *testing.T) {
	var output bytes.Buffer
	if err := serve(strings.NewReader(exampleWireRequest), &output, []string{referenceArgvMarker}); err != nil {
		t.Fatal(err)
	}
	var response processResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "ok" || response.RequestID != "request-one" ||
		response.CapabilityID != "process.reference" ||
		response.DescriptorFingerprint != "reference-process-descriptor" {
		t.Fatalf("invalid response identity: %+v", response)
	}
	var plugin struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Plugin, &plugin); err != nil || plugin.ID != "tei.process.reference" {
		t.Fatalf("plugin identity changed: %+v, err=%v", plugin, err)
	}
	var acknowledgment deploymentAcknowledgment
	if err := json.Unmarshal(response.Payload, &acknowledgment); err != nil {
		t.Fatal(err)
	}
	if acknowledgment.Outcome != "accepted" || acknowledgment.ActionID != "sales_expand/apply" ||
		acknowledgment.IdempotencyKey != "key-one" ||
		acknowledgment.ExternalRef != "reference-process://command-one" {
		t.Fatalf("invalid acknowledgment: %+v", acknowledgment)
	}
}

func TestReferenceProcessRejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		args         []string
		errorMessage string
	}{
		{"wrong-version", strings.Replace(exampleWireRequest, `"protocolVersion":1`, `"protocolVersion":2`, 1), []string{referenceArgvMarker}, "unsupported protocol version"},
		{"wrong-operation", strings.Replace(exampleWireRequest, `"operation":"execute"`, `"operation":"observe"`, 1), []string{referenceArgvMarker}, "unsupported operation"},
		{"no-argument", exampleWireRequest, nil, "reference argv marker missing"},
		{"invalid-payload", strings.Replace(exampleWireRequest, `"payload":{"id":"command-one","actionId":"sales_expand/apply","idempotencyKey":"key-one"}`, `"payload":"not an object"`, 1), []string{referenceArgvMarker}, "invalid deployment command payload"},
		{"empty-action", strings.Replace(exampleWireRequest, `"actionId":"sales_expand/apply"`, `"actionId":""`, 1), []string{referenceArgvMarker}, "incomplete deployment command"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := serve(strings.NewReader(tc.input), &output, tc.args); err != nil {
				t.Fatal(err)
			}
			var response processResponse
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Status != "error" || response.Error != tc.errorMessage || len(response.Payload) != 0 {
				t.Fatalf("expected error %q, got %+v", tc.errorMessage, response)
			}
		})
	}
}

func TestReferenceProcessRejectsAmbiguousWireWithoutEcho(t *testing.T) {
	for _, input := range []string{"{", exampleWireRequest + "\n" + exampleWireRequest} {
		var output bytes.Buffer
		if err := serve(strings.NewReader(input), &output, []string{referenceArgvMarker}); err == nil {
			t.Fatal("expected malformed or repeated request to be rejected")
		}
		if output.Len() != 0 {
			t.Fatalf("malformed request exposed output: %s", output.String())
		}
	}
}
