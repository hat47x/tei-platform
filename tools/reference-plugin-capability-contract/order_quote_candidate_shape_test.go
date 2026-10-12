package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

type quoteCandidateShapeMapping struct {
	Schema              string            `json:"schema"`
	Status              string            `json:"status"`
	Capability          string            `json:"capability"`
	RequestData         string            `json:"requestData"`
	LineData            string            `json:"lineData"`
	ResultData          string            `json:"resultData"`
	TransportContractID string            `json:"transportContractId"`
	RequestShape        string            `json:"requestShape"`
	ResultShape         string            `json:"resultShape"`
	FieldMappings       map[string]string `json:"fieldMappings"`
	Limits              []string          `json:"limits"`
}

func readQuoteCandidateShape(t *testing.T, relative string) InvocationShapeArtifact {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("../../reference/candidates/order-quote/shapes", relative))
	if err != nil {
		t.Fatal(err)
	}
	var artifact InvocationShapeArtifact
	if err := json.Unmarshal(bytes, &artifact); err != nil {
		t.Fatal(err)
	}
	if err := validateInvocationShapeArtifact(artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.Issuer.ID != "tei.reference.order-quote.candidate" ||
		artifact.PayloadCodec != "application/json" {
		t.Fatalf("candidate pretended to be a trusted publisher: %+v", artifact)
	}
	return artifact
}

func loadQuoteOverlaySemantic(t *testing.T) semantic.Namespace {
	t.Helper()
	source := "../../reference/workspace"
	scratch := filepath.Join(t.TempDir(), "workspace")
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(scratch, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{
		"semantic/sales/quote.tei",
		"applications/order-entry/patterns/preview-order-quote.tei",
		"applications/order-entry/realizations/quote-lines.tei",
	} {
		data, err := os.ReadFile(filepath.Join("../../reference/candidates/order-quote", relative))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(scratch, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := workspace.Load(scratch)
	if err != nil {
		t.Fatal(err)
	}
	for _, ns := range loaded.World.Semantics {
		if ns.ID == core.MustCanonicalID("sales") {
			return ns
		}
	}
	t.Fatal("candidate sales semantic namespace unavailable")
	return semantic.Namespace{}
}

func TestCandidateQuoteShapesMapToOverlaySemanticWithoutClaimingAuthority(t *testing.T) {
	root := "../../reference/candidates/order-quote"
	raw, err := os.ReadFile(filepath.Join(root, "shape-mapping.json"))
	if err != nil {
		t.Fatal(err)
	}
	var mapping quoteCandidateShapeMapping
	if err := json.Unmarshal(raw, &mapping); err != nil {
		t.Fatal(err)
	}
	if mapping.Schema != "tei.reference.quote-shape-mapping/candidate-v0" ||
		mapping.Status != "unaccepted" ||
		mapping.Capability != "sales.order.quote_lines" ||
		mapping.RequestData != "sales.quote_request" ||
		mapping.LineData != "sales.quote_line" ||
		mapping.ResultData != "sales.quote_result" ||
		mapping.TransportContractID != "reference.order.quote-lines/v1" ||
		len(mapping.Limits) == 0 {
		t.Fatalf("unaccepted candidate identity drift: %+v", mapping)
	}
	req := readQuoteCandidateShape(t, "request-v0.json")
	res := readQuoteCandidateShape(t, "result-v0.json")
	if mapping.RequestShape != req.ID || mapping.ResultShape != res.ID ||
		req.Issuer != res.Issuer || invocationShapeFingerprint(req) == invocationShapeFingerprint(res) {
		t.Fatal("candidate shape identity or publisher differs from declared mapping")
	}
	expected := map[string]string{
		"sales.quote_request.currency": "currency",
		"sales.quote_request.lines": "lines",
		"sales.quote_line.sku": "lines[].sku",
		"sales.quote_line.quantity": "lines[].quantity",
		"sales.quote_line.unit_price_minor": "lines[].unitPriceMinor",
		"sales.quote_result.currency": "currency",
		"sales.quote_result.line_count": "lineCount",
		"sales.quote_result.subtotal_minor": "subtotalMinor",
	}
	if !reflect.DeepEqual(mapping.FieldMappings, expected) {
		t.Fatalf("explicit field mapping changed: %+v", mapping.FieldMappings)
	}
	if !reflect.DeepEqual(req.Root.Required, []string{"currency", "lines"}) ||
		!reflect.DeepEqual(res.Root.Required, []string{"currency", "lineCount", "subtotalMinor"}) ||
		req.Root.Properties["lines"].Items == nil ||
		!reflect.DeepEqual(req.Root.Properties["lines"].Items.Required, []string{"sku", "quantity", "unitPriceMinor"}) {
		t.Fatal("candidate required input or output field shape drifted")
	}
	semanticNamespace := loadQuoteOverlaySemantic(t)
	data := map[string]semantic.Data{}
	for _, d := range semanticNamespace.Data {
		data[string(d.ID)] = d
	}
	types := map[string]string{
		"sales.quote_request.currency": "std.code",
		"sales.quote_line.sku": "std.code",
		"sales.quote_line.quantity": "std.integer",
		"sales.quote_line.unit_price_minor": "std.integer",
		"sales.quote_result.currency": "std.code",
		"sales.quote_result.line_count": "std.integer",
		"sales.quote_result.subtotal_minor": "std.integer",
	}
	for id, typ := range types {
		parts := strings.Split(id, ".")
		if len(parts) != 3 {
			t.Fatal("invalid candidate attribute identity")
		}
		dataID := parts[0] + "." + parts[1]
		model, ok := data[dataID]
		if !ok {
			t.Fatalf("candidate Data %s absent", dataID)
		}
		found := false
		for _, attr := range model.Attributes {
			if string(attr.ID) == id && string(attr.TypeRef) == typ {
				found = true
			}
		}
		if !found {
			t.Fatalf("candidate attribute %s must have type %s", id, typ)
		}
		path := mapping.FieldMappings[id]
		node := res.Root
		if parts[1] == "quote_line" || parts[1] == "quote_request" {
			node = req.Root
		}
		if strings.HasPrefix(path, "lines[].") {
			node = *req.Root.Properties["lines"].Items
			path = strings.TrimPrefix(path, "lines[].")
		}
		field, ok := node.Properties[path]
		wantKind := "number"
		if typ == "std.code" {
			wantKind = "string"
		}
		if !ok || field.Kind != wantKind {
			t.Fatalf("%s maps to wrong JSON kind: path=%q got=%+v want=%s", id, path, field, wantKind)
		}
	}
	relation := data["sales.quote_request"].Relations
	if len(relation) != 1 || relation[0].ID != "sales.quote_request.lines" ||
		relation[0].Target != "sales.quote_line" || relation[0].Cardinality != semantic.CardinalityMany {
		t.Fatalf("JSON lines array is not grounded in candidate semantic relation: %+v", relation)
	}
	matches := 0
	for _, cap := range semanticNamespace.Capabilities {
		if string(cap.ID) != mapping.Capability {
			continue
		}
		matches++
		if len(cap.Inputs) != 1 || cap.Inputs[0].Name != "request" ||
			string(cap.Inputs[0].Type) != mapping.RequestData ||
			string(cap.Output) != mapping.ResultData || cap.Effect != nil {
			t.Fatalf("candidate quote is not read-only or input/result drifted: %+v", cap)
		}
	}
	if matches != 1 {
		t.Fatalf("candidate has %d quote capabilities, want 1", matches)
	}
}

func TestCandidateQuoteShapeChecksStructureNotBusinessConstraints(t *testing.T) {
	request := readQuoteCandidateShape(t, "request-v0.json")
	result := readQuoteCandidateShape(t, "result-v0.json")
	for _, tc := range []struct {
		name string
		artifact InvocationShapeArtifact
		payload string
		allowed bool
	}{
		{"valid request", request, `{"currency":"JPY","lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250}]}`, true},
		{"valid result", result, `{"currency":"JPY","lineCount":1,"subtotalMinor":750}`, true},
		{"missing request member", request, `{"currency":"JPY"}`, false},
		{"wrong nested type", request, `{"currency":"JPY","lines":[{"sku":"PEN","quantity":"3","unitPriceMinor":250}]}`, false},
		{"unexpected security field", request, `{"currency":"JPY","lines":[],"principalId":"admin"}`, false},
		{"unexpected line field", request, `{"currency":"JPY","lines":[{"sku":"PEN","quantity":1,"unitPriceMinor":250,"grant":"admin"}]}`, false},
		{"missing result field", result, `{"currency":"JPY","subtotalMinor":750}`, false},
		{"wrong result type", result, `{"currency":"JPY","lineCount":1,"subtotalMinor":"750"}`, false},
		{"unexpected result field", result, `{"currency":"JPY","lineCount":1,"subtotalMinor":750,"submitted":true}`, false},
		{"fraction passes structural number shape only", request, `{"currency":"JPY","lines":[{"sku":"PEN","quantity":1.5,"unitPriceMinor":250}]}`, true},
		{"negative passes structural number shape only", request, `{"currency":"JPY","lines":[{"sku":"PEN","quantity":-1,"unitPriceMinor":250}]}`, true},
		{"unknown currency passes structural string shape only", request, `{"currency":"USD","lines":[{"sku":"PEN","quantity":1,"unitPriceMinor":250}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInvocationPayload(tc.artifact, []byte(tc.payload))
			if (err == nil) != tc.allowed {
				t.Fatalf("%s: accepted=%t want=%t err=%v", tc.name, err == nil, tc.allowed, err)
			}
		})
	}
	// A shape fingerprint is only a derived identifier: without trusted
	// publisher evidence it is not Shape Authority or provider conformance.
	if got := invocationShapeFingerprint(request); !strings.HasPrefix(got, "sha256:") {
		t.Fatal(fmt.Sprintf("candidate shape has no derived fingerprint: %q", got))
	}
}
