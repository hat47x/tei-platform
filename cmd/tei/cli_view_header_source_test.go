package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/realization"
	"tei/internal/workspace"
)

func TestRuntimeHeaderSourceUsesDeclaredInputAndSemanticAttribute(t *testing.T) {
	for _, tc := range []struct {
		name      string
		inputName string
		attribute core.CanonicalID
		value     string
		number    int64
	}{
		{name: "reference order", inputName: "order_id", attribute: "sales.order.order_id", value: "1001", number: 1001},
		{name: "different declaration", inputName: "parent_key", attribute: "other.header.identity", value: "42", number: 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bindings := []realization.HeaderDetailContractQueryInputBinding{
				{Input: core.LocalName(tc.inputName), SourceAttribute: tc.attribute},
			}
			got, err := runtimeHeaderSourceFromBinding(bindings, map[string]string{tc.inputName: tc.value})
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Columns) != 1 || got.Columns[0].Raw != tc.inputName ||
				got.Columns[0].Attribute != tc.attribute {
				t.Fatalf("source mapping did not follow the binding: %+v", got.Columns)
			}
			if !reflect.DeepEqual(got.Rows, [][]any{{tc.number}}) {
				t.Fatalf("mapped source rows = %#v, want int64(%d)", got.Rows, tc.number)
			}
		})
	}
}

func TestRuntimeHeaderSourceRejectsUnboundOrMalformedInputs(t *testing.T) {
	good := []realization.HeaderDetailContractQueryInputBinding{
		{Input: "order_id", SourceAttribute: "sales.order.order_id"},
	}
	for _, tc := range []struct {
		name     string
		bindings []realization.HeaderDetailContractQueryInputBinding
		inputs   map[string]string
		errorHas string
	}{
		{name: "no input", bindings: good, inputs: map[string]string{}, errorHas: "order_id"},
		{name: "other input", bindings: good, inputs: map[string]string{"parent_key": "1001"}, errorHas: "order_id"},
		{name: "extra input", bindings: good, inputs: map[string]string{"order_id": "1001", "parent_key": "1001"}, errorHas: "order_id"},
		{name: "empty", bindings: good, inputs: map[string]string{"order_id": ""}, errorHas: "order_id"},
		{name: "zero", bindings: good, inputs: map[string]string{"order_id": "0"}, errorHas: "positive 64-bit integer"},
		{name: "negative", bindings: good, inputs: map[string]string{"order_id": "-1"}, errorHas: "positive 64-bit integer"},
		{name: "not integer", bindings: good, inputs: map[string]string{"order_id": "5.5"}, errorHas: "positive 64-bit integer"},
		{name: "overflow", bindings: good, inputs: map[string]string{"order_id": "9223372036854775808"}, errorHas: "positive 64-bit integer"},
		{name: "missing binding", bindings: nil, inputs: map[string]string{"order_id": "1001"}, errorHas: "explicit header input binding"},
		{name: "ambiguous bindings", bindings: append(append([]realization.HeaderDetailContractQueryInputBinding{}, good...), good[0]), inputs: map[string]string{"order_id": "1001"}, errorHas: "explicit header input binding"},
		{name: "missing attribute", bindings: []realization.HeaderDetailContractQueryInputBinding{{Input: "order_id"}}, inputs: map[string]string{"order_id": "1001"}, errorHas: "explicit header input binding"},
		{name: "missing input name", bindings: []realization.HeaderDetailContractQueryInputBinding{{SourceAttribute: "sales.order.order_id"}}, inputs: map[string]string{"order_id": "1001"}, errorHas: "explicit header input binding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runtimeHeaderSourceFromBinding(tc.bindings, tc.inputs); err == nil || !strings.Contains(err.Error(), tc.errorHas) {
				t.Fatalf("expected error containing %q, got %v", tc.errorHas, err)
			}
		})
	}
}

func TestRuntimeHeaderBindingIsCheckedBeforeDatabaseConnection(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	world := loaded.World
	found := false
	for i := range world.HeaderDetailContractQueryBindings {
		binding := &world.HeaderDetailContractQueryBindings[i]
		if binding.Application == "order_entry" && binding.Pattern == "order_entry.order_lines" {
			if len(binding.Inputs) != 1 {
				t.Fatalf("expected one declared header input: %+v", binding.Inputs)
			}
			binding.Inputs[0].SourceAttribute = "sales.order.no_such_attribute"
			found = true
		}
	}
	if !found {
		t.Fatal("reference Workspace has no header-detail Query binding")
	}
	t.Setenv("TEI_VIEW_PG_DSN", "")
	_, err = readDestinationOrderLines(world, "order_entry", "order_entry.order_lines", "development",
		map[string]string{"order_id": "1001"})
	if err == nil || !strings.Contains(err.Error(), "sales.order.no_such_attribute") {
		t.Fatalf("malformed Semantic binding was not rejected: %v", err)
	}
	if strings.Contains(err.Error(), "TEI_VIEW_PG_DSN") {
		t.Fatalf("database connection was required before validating the source binding: %v", err)
	}
}
