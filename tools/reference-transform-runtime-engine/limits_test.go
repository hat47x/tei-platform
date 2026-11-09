package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"tei/internal/plugin/transform"
)

// testLimits returns small, explicit limits so the boundary tests do not depend
// on the limits of the reference meta-builder Source.
func testLimits() transform.Limits {
	return transform.Limits{
		MaxSteps:        32,
		MaxCollection:   8,
		MaxInputNodes:   64,
		MaxInputBytes:   1024,
		MaxOutputBytes:  1024,
		MaxNestingDepth: 8,
	}
}

// nestedArrayRequestJSON builds a request whose input value is an array nested
// `levels` times around a null. Each semantic level adds two JSON levels
// (the wire-value object and its array), so a modest semantic depth is already
// a deeper JSON structure.
func nestedArrayRequestJSON(levels int) []byte {
	var b strings.Builder
	b.WriteString(`{"schema":"` + engineRequestSchema + `","inputs":[{"name":"x","value":`)
	for i := 0; i < levels; i++ {
		b.WriteString(`{"kind":"array","array":[`)
	}
	b.WriteString(`{"kind":"null"}`)
	for i := 0; i < levels; i++ {
		b.WriteString(`]}`)
	}
	b.WriteString(`}]}`)
	return []byte(b.String())
}

// nestedArrayWire builds the same shape as a decoded wire value.
func nestedArrayWire(levels int) wireValue {
	w := wireValue{Kind: transform.ValueNull}
	for i := 0; i < levels; i++ {
		items := []wireValue{w}
		w = wireValue{Kind: transform.ValueArray, Array: &items}
	}
	return w
}

// TestDecodeRequestRejectsDeepJSONBeforeTypedDecode pins the streaming
// structure walk that runs before json.Decoder allocates the request (#1079).
//
// The structure limit is MaxNestingDepth+8. A wire value at semantic depth 11
// is still within the semantic limit (16), but its JSON object is already at
// depth 25 and exceeds the structure limit. Only the pre-decode walk rejects
// this request. If the walk is removed, decodeRequest succeeds.
func TestDecodeRequestRejectsDeepJSONBeforeTypedDecode(t *testing.T) {
	limits := testLimits()
	raw := nestedArrayRequestJSON(8)
	if _, err := decodeRequest(raw, limits); err == nil {
		t.Fatal("expected the pre-decode structure walk to reject deep JSON")
	} else if !strings.Contains(err.Error(), "JSON nesting exceeds limit") {
		t.Fatalf("error = %v, want the streaming structure error; a different error means the typed decode ran first", err)
	}
}

// TestDecodeRequestAcceptsShallowJSON is the positive control for the previous
// test: a shallow request must still decode.
func TestDecodeRequestAcceptsShallowJSON(t *testing.T) {
	limits := testLimits()
	request, err := decodeRequest(nestedArrayRequestJSON(3), limits)
	if err != nil {
		t.Fatalf("shallow request was rejected: %v", err)
	}
	if len(request.Inputs) != 1 {
		t.Fatalf("decoded inputs: %#v", request.Inputs)
	}
}

// TestDecodeWireRejectsDepthAboveMaxNestingDepth pins the wire-decode boundary.
// The structure walk allows MaxNestingDepth+8 JSON levels for the envelope, so
// the wire decoder must enforce the semantic limit itself.
func TestDecodeWireRejectsDepthAboveMaxNestingDepth(t *testing.T) {
	limits := testLimits()
	if _, err := decodeWireValue(nestedArrayWire(limits.MaxNestingDepth), limits, 0, &wireBudget{}); err != nil {
		t.Fatalf("depth at the limit was rejected: %v", err)
	}
	_, err := decodeWireValue(nestedArrayWire(limits.MaxNestingDepth+1), limits, 0, &wireBudget{})
	if err == nil {
		t.Fatal("expected depth above MaxNestingDepth to fail")
	}
	if !strings.Contains(err.Error(), "maxNestingDepth") {
		t.Fatalf("error = %v, want maxNestingDepth", err)
	}
}

// TestDecodeWireRejectsCollectionAboveMaxCollection pins the array and object
// collection limits at wire decode.
func TestDecodeWireRejectsCollectionAboveMaxCollection(t *testing.T) {
	limits := testLimits()
	null := wireValue{Kind: transform.ValueNull}

	items := make([]wireValue, limits.MaxCollection+1)
	for i := range items {
		items[i] = null
	}
	if _, err := decodeWireValue(wireValue{Kind: transform.ValueArray, Array: &items}, limits, 0, &wireBudget{}); err == nil {
		t.Fatal("expected an array above maxCollection to fail")
	} else if !strings.Contains(err.Error(), "maxCollection") {
		t.Fatalf("array error = %v, want maxCollection", err)
	}

	fields := make([]wireField, limits.MaxCollection+1)
	for i := range fields {
		fields[i] = wireField{Name: fmt.Sprintf("f%d", i), Value: null}
	}
	if _, err := decodeWireValue(wireValue{Kind: transform.ValueObject, Object: &fields}, limits, 0, &wireBudget{}); err == nil {
		t.Fatal("expected an object above maxCollection to fail")
	} else if !strings.Contains(err.Error(), "maxCollection") {
		t.Fatalf("object error = %v, want maxCollection", err)
	}
}

// TestDecodeWireRejectsNodesAboveMaxInputNodes pins the node budget at wire
// decode. An array at the collection limit plus its items already exceeds a
// node budget of the same number.
func TestDecodeWireRejectsNodesAboveMaxInputNodes(t *testing.T) {
	limits := testLimits()
	limits.MaxInputNodes = 8
	null := wireValue{Kind: transform.ValueNull}
	items := make([]wireValue, limits.MaxInputNodes)
	for i := range items {
		items[i] = null
	}
	_, err := decodeWireValue(wireValue{Kind: transform.ValueArray, Array: &items}, limits, 0, &wireBudget{})
	if err == nil {
		t.Fatal("expected an array plus items above maxInputNodes to fail")
	}
	if !strings.Contains(err.Error(), "maxInputNodes") {
		t.Fatalf("error = %v, want maxInputNodes", err)
	}

	// One node fewer succeeds, so the boundary is not over-rejecting.
	smaller := make([]wireValue, limits.MaxInputNodes-1)
	for i := range smaller {
		smaller[i] = null
	}
	if _, err := decodeWireValue(wireValue{Kind: transform.ValueArray, Array: &smaller}, limits, 0, &wireBudget{}); err != nil {
		t.Fatalf("array at the node budget was rejected: %v", err)
	}
}

// TestDecodeRequestRejectsInputsAboveMaxCollection pins the request-level
// collection limit that runs after the typed decode.
func TestDecodeRequestRejectsInputsAboveMaxCollection(t *testing.T) {
	limits := testLimits()
	null := wireValue{Kind: transform.ValueNull}
	request := engineRequest{Schema: engineRequestSchema}
	for i := 0; i < limits.MaxCollection+1; i++ {
		request.Inputs = append(request.Inputs, wireInput{Name: fmt.Sprintf("i%d", i), Value: null})
	}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	_, err = decodeRequest(raw, limits)
	if err == nil {
		t.Fatal("expected request inputs above maxCollection to fail")
	}
	if !strings.Contains(err.Error(), "maxCollection") {
		t.Fatalf("error = %v, want maxCollection", err)
	}
}
