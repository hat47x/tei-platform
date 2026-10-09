package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin/transform"
)

func requestBytes(t *testing.T, request engineRequest) []byte {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestExecuteMetaBuilderThroughTypedWire(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	value, err := transform.SourceValue(source)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := encodeWireValue(value, source.Limits, 0)
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
	if response.Schema != engineResponseSchema || response.Steps <= 0 {
		t.Fatalf("response metadata: %#v", response)
	}
	if response.Output.Kind != transform.ValueBytes || response.Output.BytesBase64 == nil {
		t.Fatalf("output wire value: %#v", response.Output)
	}
	got, err := base64.StdEncoding.Strict().DecodeString(*response.Output.BytesBase64)
	if err != nil {
		t.Fatal(err)
	}
	want, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("engine output differs from canonical Source artifact")
	}
}

func TestExecutePreservesBytesValue(t *testing.T) {
	input := transform.Expr{Op: transform.OpInput, Name: "payload"}
	source := transform.Source{
		FormatVersion: transform.SourceFormatVersion,
		Identity: transform.SourceIdentity{
			ID:              core.MustCanonicalID("tei.runtime.engine.bytes_test"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-transform/v1",
		},
		Profile: transform.ProfileV1,
		Limits: transform.Limits{
			MaxSteps:        32,
			MaxCollection:   16,
			MaxInputNodes:   16,
			MaxInputBytes:   1024,
			MaxOutputBytes:  1024,
			MaxNestingDepth: 16,
		},
		Inputs:  []transform.InputSpec{{Name: "payload", Kind: transform.InputBytes}},
		Program: transform.Program{Output: input},
	}
	payload := []byte{0, 1, 2, 3, 0xff}
	encoded := base64.StdEncoding.EncodeToString(payload)
	response, err := execute(source, requestBytes(t, engineRequest{
		Schema: engineRequestSchema,
		Inputs: []wireInput{{
			Name: "payload",
			Value: wireValue{
				Kind:        transform.ValueBytes,
				BytesBase64: &encoded,
			},
		}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Output.BytesBase64 == nil {
		t.Fatal("bytes output is missing")
	}
	got, err := base64.StdEncoding.Strict().DecodeString(*response.Output.BytesBase64)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("bytes changed: %v != %v", got, payload)
	}
}

func TestDecodeRequestRejectsAmbiguousOrUnknownJSON(t *testing.T) {
	limits := transform.TransformSourceRealizerV1().Limits
	tests := []string{
		`{"schema":"tei.reference.transform-engine-request/v1","schema":"tei.reference.transform-engine-request/v1","inputs":[]}`,
		`{"schema":"tei.reference.transform-engine-request/v1","inputs":[],"unknown":true}`,
		`{"schema":"wrong","inputs":[]}`,
		`{"schema":"tei.reference.transform-engine-request/v1","inputs":[{"name":"x","value":{"kind":"null","unknown":true}}]}`,
	}
	for _, raw := range tests {
		if _, err := decodeRequest([]byte(raw), limits); err == nil {
			t.Fatalf("expected request to fail: %s", raw)
		}
	}
	if _, err := decodeRequest([]byte{0xff}, limits); err == nil {
		t.Fatal("expected invalid UTF-8 request to fail")
	}
}

func TestDecodeRequestRejectsExplicitNullForWrongOrRequiredWireField(t *testing.T) {
	limits := transform.TransformSourceRealizerV1().Limits
	tests := []string{
		`{"schema":"tei.reference.transform-engine-request/v1","inputs":[{"name":"x","value":{"kind":"null","bool":null}}]}`,
		`{"schema":"tei.reference.transform-engine-request/v1","inputs":[{"name":"x","value":{"kind":"bool","bool":null}}]}`,
		`{"schema":"tei.reference.transform-engine-request/v1","inputs":[{"name":"x","value":{"kind":"string","string":null}}]}`,
		`{"schema":"tei.reference.transform-engine-request/v1","inputs":[{"name":"x","value":{"kind":"bytes","bytesBase64":null}}]}`,
	}
	for _, raw := range tests {
		request, err := decodeRequest([]byte(raw), limits)
		if err != nil {
			t.Fatalf("request envelope unexpectedly failed before wire validation: %v", err)
		}
		if len(request.Inputs) != 1 {
			t.Fatalf("decoded inputs: %#v", request.Inputs)
		}
		if _, err := decodeWireValue(request.Inputs[0].Value, limits, 0, &wireBudget{}); err == nil {
			t.Fatalf("expected explicit null wire field to fail: %s", raw)
		}
	}
}

func TestExecuteRejectsDuplicateInputNames(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	null := wireValue{Kind: transform.ValueNull}
	_, err := execute(source, requestBytes(t, engineRequest{
		Schema: engineRequestSchema,
		Inputs: []wireInput{
			{Name: "transformSource", Value: null},
			{Name: "transformSource", Value: null},
		},
	}))
	if err == nil {
		t.Fatal("expected duplicate input name to fail")
	}
}

func TestDecodeWireRejectsHiddenFieldsAndInvalidBase64(t *testing.T) {
	limits := transform.TransformSourceRealizerV1().Limits
	flag := true
	text := "not-base64!"
	tests := []wireValue{
		{Kind: transform.ValueString, String: &text, Bool: &flag},
		{Kind: transform.ValueBytes, BytesBase64: &text},
	}
	for _, wire := range tests {
		if _, err := decodeWireValue(wire, limits, 0, &wireBudget{}); err == nil {
			t.Fatalf("expected wire value to fail: %#v", wire)
		}
	}
}

func TestWireBytesBudgetUsesExactDecodedLength(t *testing.T) {
	limits := transform.TransformSourceRealizerV1().Limits
	limits.MaxInputBytes = 1
	encoded := base64.StdEncoding.EncodeToString([]byte{0xff})
	value, err := decodeWireValue(
		wireValue{Kind: transform.ValueBytes, BytesBase64: &encoded},
		limits,
		0,
		&wireBudget{},
	)
	if err != nil {
		t.Fatalf("one decoded byte at exact budget was rejected: %v", err)
	}
	if value.Kind != transform.ValueBytes || len(value.Bytes) != 1 || value.Bytes[0] != 0xff {
		t.Fatalf("decoded bytes: %#v", value)
	}

	over := base64.StdEncoding.EncodeToString([]byte{0xff, 0x00})
	if _, err := decodeWireValue(
		wireValue{Kind: transform.ValueBytes, BytesBase64: &over},
		limits,
		0,
		&wireBudget{},
	); err == nil {
		t.Fatal("expected two decoded bytes to exceed one-byte budget")
	}
}

func TestWireDecodeStopsAtSemanticInputBudget(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	source.Limits.MaxInputBytes = 8
	long := strings.Repeat("x", 32)
	_, err := execute(source, requestBytes(t, engineRequest{
		Schema: engineRequestSchema,
		Inputs: []wireInput{{
			Name: "x",
			Value: wireValue{
				Kind:   transform.ValueString,
				String: &long,
			},
		}},
	}))
	if err == nil {
		t.Fatal("expected semantic input byte budget to fail")
	}
}

func TestRequestLimitIsHostBoundedAboveSemanticInput(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	got := requestLimit(source)
	if got <= int64(source.Limits.MaxInputBytes) {
		t.Fatalf("wire envelope limit must account for typed/base64 overhead: %d", got)
	}
	if got > int64(source.Limits.MaxInputBytes)*4+requestEnvelopeSlack {
		t.Fatalf("wire envelope limit escaped Host bound: %d", got)
	}
}


func TestWireObjectPreservesFieldOrder(t *testing.T) {
	limits := transform.TransformSourceRealizerV1().Limits
	a := "first"
	b := "second"
	fields := []wireField{
		{Name: "b", Value: wireValue{Kind: transform.ValueString, String: &b}},
		{Name: "a", Value: wireValue{Kind: transform.ValueString, String: &a}},
	}
	wire := wireValue{Kind: transform.ValueObject, Object: &fields}
	value, err := decodeWireValue(wire, limits, 0, &wireBudget{})
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Object) != 2 || value.Object[0].Name != "b" || value.Object[1].Name != "a" {
		t.Fatalf("object field order changed: %#v", value.Object)
	}
	encoded, err := encodeWireValue(value, limits, 0)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.Object == nil || len(*encoded.Object) != 2 ||
		(*encoded.Object)[0].Name != "b" || (*encoded.Object)[1].Name != "a" {
		t.Fatalf("wire field order changed: %#v", encoded.Object)
	}
}
