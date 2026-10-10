package languagereference

import "testing"

func TestPipelineLoweringAndComposition(t *testing.T) {
	catalog := []Operator{
		{ID: "std.op.where", Version: "1", InputType: "rows/order", OutputType: "rows/order", Effect: "pure"},
		{ID: "std.op.expand", Version: "1", InputType: "rows/order", OutputType: "rows/line", Effect: "pure"},
		{ID: "std.op.summarize", Version: "1", InputType: "rows/line", OutputType: "rows/total", Effect: "pure"},
	}
	input, calls, err := ParsePipeline("rows/order |> std.op.where |> std.op.expand |> std.op.summarize", catalog)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Compose(input, calls, catalog)
	if err != nil || result != "rows/total" {
		t.Fatalf("got %s: %v", result, err)
	}
}
func TestPipelineRejectsUnresolvedSyntaxAndMeaning(t *testing.T) {
	catalog := []Operator{{ID: "std.op.where", Version: "1", InputType: "rows/order", OutputType: "rows/order", Effect: "pure"}}
	sources := []string{
		"rows/order",
		"rows/order |>",
		"rows/order |> std.op.unknown",
		"rows/other |> std.op.where",
		"rows/order |> std.op.where extra",
	}
	for _, src := range sources {
		t.Run(src, func(t *testing.T) {
			_, _, err := ParsePipeline(src, catalog)
			if err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}
