package transform_test

import (
	"testing"

	"tei/internal/plugin/transform"
)

func TestOmitEmptyArrayFieldIsGenericAndNonDestructive(t *testing.T) {
	source := uniqueByTestSource()
	source.Inputs = []transform.InputSpec{{Name: "doc", Kind: transform.InputObject}}
	doc := transform.Expr{Op: transform.OpInput, Name: "doc"}
	source.Program.Assertions = nil
	source.Program.Output = transform.Expr{
		Op: transform.OpOmitEmptyArrayField, From: &doc, Field: "provides",
	}
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty array omitted", raw: `{"provides":[],"other":1}`, want: `{"other":1}`},
		{name: "nonempty array preserved", raw: `{"provides":["runtime.a"],"other":1}`, want: `{"provides":["runtime.a"],"other":1}`},
		{name: "absent field preserved", raw: `{"other":1}`, want: `{"other":1}`},
		{name: "null is not an empty array", raw: `{"provides":null,"other":1}`, want: `{"provides":null,"other":1}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, err := transform.ParseJSON([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			expected, err := transform.ParseJSON([]byte(tc.want))
			if err != nil {
				t.Fatal(err)
			}
			result, err := transform.Evaluate(source, map[string]transform.Value{"doc": in})
			if err != nil {
				t.Fatal(err)
			}
			if !transform.Equal(result.Output, expected) {
				t.Fatalf("unexpected empty-array omission: got %#v want %#v", result.Output, expected)
			}
			// The evaluator must not mutate caller-owned input Values.
			original, _ := transform.ParseJSON([]byte(tc.raw))
			if !transform.Equal(in, original) {
				t.Fatal("generic omission mutated its input")
			}
		})
	}
	invalid := source
	invalid.Program.Output.Field = ""
	if err := invalid.Validate(); err == nil {
		t.Fatal("missing field in omit-empty-array-field accepted by Source validator")
	}
}
