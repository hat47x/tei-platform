package generatorselection

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

// #2585 PFC-06: many fields are only supported for text-like primitives. The
// contract validator must reject the other combinations instead of accepting
// them statically and failing on the first runtime element.
func TestRecordProjectionManySupportMatrix(t *testing.T) {
	base := Contract{
		Schema: ContractSchema, ID: "sui.card.pfc_many", Version: "1", Status: "reference-local",
		Output: TypeSpec{Primitive: "std.text"},
	}
	withMany := func(primitive string) Contract {
		contract := base
		contract.Inputs = []InputSpec{{Attribute: "sui.card.tag", Primitive: primitive}}
		contract.Projection = &ProjectionSpec{Kind: "record", Fields: []ProjectionField{
			{Name: "tags", Input: "tags", Attribute: "sui.card.tag", Primitive: primitive, Many: true},
		}}
		return contract
	}
	for _, primitive := range []string{"std.text", "std.identifier"} {
		if err := validateContract(withMany(primitive)); err != nil {
			t.Fatalf("many %s should be supported: %v", primitive, err)
		}
	}
	for _, primitive := range []string{"std.integer", "std.decimal", "std.boolean"} {
		err := validateContract(withMany(primitive))
		if err == nil {
			t.Fatalf("many %s was accepted although the runtime only handles strings", primitive)
		}
	}
}

// #2585 PFC-07: a many field preserves emptiness, duplicates and order. Set
// uniqueness is a business contract, not part of this projection.
func TestRecordProjectionPFCManySemantics(t *testing.T) {
	contractPath := filepath.Join("testdata", "pfc-many", "contract.json")
	out := filepath.Join(t.TempDir(), "candidate")
	if _, err := Generate(contractPath, "text-record-template", out); err != nil {
		t.Fatal(err)
	}
	binary := candidateBinaryPath(filepath.Join(t.TempDir(), "candidate"))
	if err := buildCandidate(context.Background(), out, binary); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		raw       string
		want      Result
		wantError bool
	}{
		{name: "missing", raw: `{}`, want: Result{State: "missing", Text: `["tags"]`}},
		{name: "unresolved", raw: `{"tags":null}`, want: Result{State: "unresolved"}},
		{name: "empty", raw: `{"tags":[]}`, want: Result{State: "present", Text: `{"tags":[]}`}},
		{name: "duplicate", raw: `{"tags":["c1","c1"]}`, want: Result{State: "present", Text: `{"tags":["c1","c1"]}`}},
		{name: "order", raw: `{"tags":["b","a"]}`, want: Result{State: "present", Text: `{"tags":["b","a"]}`}},
		// A non-string element is rejected at runtime with exit code 2.
		{name: "non-string-element", raw: `{"tags":["a",1]}`, wantError: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, _, err := runCandidate(context.Background(), binary, json.RawMessage(testCase.raw), ExecutionOptions{})
			if testCase.wantError {
				if err == nil {
					t.Fatalf("non-string element was accepted: %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("got=%+v want=%+v", got, testCase.want)
			}
		})
	}
}
