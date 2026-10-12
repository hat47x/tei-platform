package transform_test

import (
	"strings"
	"testing"

	"tei/internal/plugin/transform"
)

func TestRequirementAssertionsRejectInvalidValuesAndPreserveValidInputs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		op     transform.Op
		raw    string
		valid  bool
	}{
		{name: "canonical single segment", op: transform.OpAssertCanonicalID, raw: `"tei"`, valid: true},
		{name: "canonical dotted id", op: transform.OpAssertCanonicalID, raw: `"tei.base_1"`, valid: true},
		{name: "noncanonical spaces", op: transform.OpAssertCanonicalID, raw: `" tei.base "`},
		{name: "noncanonical uppercase", op: transform.OpAssertCanonicalID, raw: `"tei.Base"`},
		{name: "noncanonical empty", op: transform.OpAssertCanonicalID, raw: `""`},
		{name: "canonical expects string", op: transform.OpAssertCanonicalID, raw: `1`},
		{name: "positive one", op: transform.OpAssertPositiveInt, raw: `1`, valid: true},
		{name: "positive larger number", op: transform.OpAssertPositiveInt, raw: `12`, valid: true},
		{name: "zero", op: transform.OpAssertPositiveInt, raw: `0`},
		{name: "negative", op: transform.OpAssertPositiveInt, raw: `-2`},
		{name: "fraction", op: transform.OpAssertPositiveInt, raw: `1.5`},
		{name: "integer decimal", op: transform.OpAssertPositiveInt, raw: `1.0`},
		{name: "scientific notation", op: transform.OpAssertPositiveInt, raw: `1e0`},
		{name: "integer expects number", op: transform.OpAssertPositiveInt, raw: `"1"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := uniqueByTestSource()
			source.Inputs = []transform.InputSpec{{Name: "doc", Kind: transform.InputObject}}
			input := transform.Expr{Op: transform.OpInput, Name: "doc"}
			field := transform.Expr{Op: transform.OpSelectField, From: &input, Field: "value"}
			source.Program.Assertions = []transform.Expr{{Op: tc.op, From: &field}}
			source.Program.Output = input
			doc, err := transform.ParseJSON([]byte(`{"value":` + tc.raw + "}"))
			if err != nil {
				t.Fatal(err)
			}
			got, err := transform.Evaluate(source, map[string]transform.Value{"doc": doc})
			if (err == nil) != tc.valid {
				t.Fatalf("op %s value %s expected valid=%t, got %v", tc.op, tc.raw, tc.valid, err)
			}
			if tc.valid && !transform.Equal(got.Output, doc) {
				t.Fatal("assertion changed its input")
			}
			if !tc.valid && err != nil && !strings.Contains(err.Error(), string(tc.op)) {
				t.Fatalf("assertion error did not identify operation: %v", err)
			}
			invalidSource := source
			invalidSource.Program.Assertions = []transform.Expr{{Op: tc.op}}
			if err := invalidSource.Validate(); err == nil {
				t.Fatal("missing assertion operand accepted")
			}
		})
	}
}
