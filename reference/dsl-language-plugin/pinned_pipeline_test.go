package languagereference

import "testing"

func TestPinnedPipelineRoundTrip(t *testing.T) {
	where := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "where-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	expand := OperatorContract{ID: "std.expand", Version: "1", Fingerprint: "expand-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/line"}
	source := "rows/order |> std.where |> std.expand"
	for _, catalog := range [][]OperatorContract{{where, expand}, {expand, where}} {
		input, calls, err := ParsePinnedPipeline(source, []OperatorContract{where, expand}, catalog)
		if err != nil {
			t.Fatal(err)
		}
		out, err := ComposePinned(input, calls, catalog)
		if err != nil || out != "rows/line" {
			t.Fatalf("got %q: %v", out, err)
		}
	}
}

func TestPinnedPipelineFailsClosed(t *testing.T) {
	pin := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "where-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	tests := []struct {
		name, source   string
		pins, catalog []OperatorContract
	}{
		{"no pins", "rows/order |> std.where", nil, []OperatorContract{pin}},
		{"wrong identity", "rows/order |> std.other", []OperatorContract{pin}, []OperatorContract{pin}},
		{"wrong input", "rows/other |> std.where", []OperatorContract{pin}, []OperatorContract{pin}},
		{"unknown fingerprint", "rows/order |> std.where", []OperatorContract{pin}, []OperatorContract{{ID: pin.ID, Version: pin.Version, Fingerprint: "different", Effect: pin.Effect, InputType: pin.InputType, OutputType: pin.OutputType}}},
		{"duplicate", "rows/order |> std.where", []OperatorContract{pin}, []OperatorContract{pin, pin}},
		{"trailing pipe", "rows/order |> std.where |>", []OperatorContract{pin, pin}, []OperatorContract{pin}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ParsePinnedPipeline(tc.source, tc.pins, tc.catalog)
			if err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestPinnedPipelineCoexistingVersions(t *testing.T) {
	v1 := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "where-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	v2 := OperatorContract{ID: "std.where", Version: "2", Fingerprint: "where-v2", Effect: "pure", InputType: "rows/order", OutputType: "rows/filtered"}
	for _, catalog := range [][]OperatorContract{{v1, v2}, {v2, v1}} {
		input, calls, err := ParsePinnedPipeline("rows/order |> std.where", []OperatorContract{v1}, catalog)
		if err != nil {
			t.Fatal(err)
		}
		out, err := ComposePinned(input, calls, catalog)
		if err != nil || out != "rows/order" {
			t.Fatalf("pin drift: output %q, %v", out, err)
		}
	}
	if _, _, err := ParsePinnedPipeline("rows/order |> std.where", []OperatorContract{v1}, []OperatorContract{v2}); err == nil {
		t.Fatal("version was silently upgraded")
	}
}
