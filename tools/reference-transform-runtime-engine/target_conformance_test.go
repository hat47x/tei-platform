package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"tei/internal/plugin/transform"
)

// The same typed-wire transformation runs unchanged on native Go and
// js/wasm. The independent test driver compares the complete response digest
// across targets; passing only on each target is not sufficient.
func TestTargetConformanceWireDigest(t *testing.T) {
	source := referenceMetaBuilderSource(t)
	input, err := transform.SourceValue(source)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := encodeWireValue(input, source.Limits, 0)
	if err != nil {
		t.Fatal(err)
	}
	response, err := execute(source, requestBytes(t, engineRequest{
		Schema: engineRequestSchema,
		Inputs: []wireInput{{Name: "transformSource", Value: wire}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Schema != engineResponseSchema || response.Steps <= 0 ||
		response.Output.Kind != transform.ValueBytes || response.Output.BytesBase64 == nil {
		t.Fatalf("invalid typed response: %#v", response)
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(*response.Output.BytesBase64)
	if err != nil {
		t.Fatal(err)
	}
	expected, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(expected) {
		t.Fatal("target did not produce the canonical realization")
	}

	// Include envelope metadata and step accounting, not just the output.
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	t.Logf("TARGET_CONFORMANCE_SHA256=%x", sum)
	if len(fmt.Sprintf("%x", sum)) != 64 {
		t.Fatal("invalid conformance fingerprint")
	}
}
