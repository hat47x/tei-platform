package languagereference

import "testing"

func TestPinnedLoweringPreservesStepSpans(t *testing.T) {
	src := "rows/order |> std.where |> std.expand"
	syntax, err := ParsePipelineSyntax(src)
	if err != nil {
		t.Fatal(err)
	}
	where := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	expand := OperatorContract{ID: "std.expand", Version: "1", Fingerprint: "v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/line"}
	steps, err := LowerPinnedPipeline(syntax, []OperatorContract{where, expand}, []OperatorContract{expand, where})
	if err != nil {
		t.Fatal(err)
	}
	for i, step := range steps {
		if src[step.Span.Start:step.Span.End] != syntax.Operators[i].ID {
			t.Errorf("lost source span at step %d: %#v", i, step.Span)
		}
	}
	if steps[1].InputType != "rows/order" || steps[1].Contract.OutputType != "rows/line" {
		t.Fatalf("typed lowering mismatch: %#v", steps[1])
	}
}

func TestPinnedLoweringFailsClosed(t *testing.T) {
	src := "rows/order |> std.where"
	syntax, err := ParsePipelineSyntax(src)
	if err != nil {
		t.Fatal(err)
	}
	where := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	for _, tc := range []struct {
		name    string
		pins    []OperatorContract
		catalog []OperatorContract
	}{
		{"no pin", nil, []OperatorContract{where}},
		{"no matching catalog", []OperatorContract{where}, nil},
		{"different source operator", []OperatorContract{{ID: "std.other", Version: "1", Fingerprint: "v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}}, []OperatorContract{where}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := LowerPinnedPipeline(syntax, tc.pins, tc.catalog); err == nil {
				t.Fatal("expected fail-closed lowering")
			}
		})
	}
}
