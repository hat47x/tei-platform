package languagereference

import "testing"

func TestPipelineSyntaxSeparatesParsingFromContracts(t *testing.T) {
	input := "  rows/order  |>  std.where  |>  std.expand  "
	got, err := ParsePipelineSyntax(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Input.ID != "rows/order" || len(got.Operators) != 2 {
		t.Fatalf("unexpected syntax %#v", got)
	}
	for _, tok := range append([]PipelineToken{got.Input}, got.Operators...) {
		if input[tok.Span.Start:tok.Span.End] != tok.ID {
			t.Fatalf("incorrect span %#v", tok)
		}
	}
}

func TestPipelineSyntaxUTF8ByteSpans(t *testing.T) {
	input := "　rows/order |> std.where"
	got, err := ParsePipelineSyntax(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Input.Span.Start != len("　") {
		t.Fatalf("expected UTF-8 byte offset %d, got %d", len("　"), got.Input.Span.Start)
	}
	if input[got.Operators[0].Span.Start:got.Operators[0].Span.End] != "std.where" {
		t.Fatalf("incorrect operator span %#v", got.Operators[0])
	}
}

func TestPipelineSyntaxRejectsBrokenTokens(t *testing.T) {
	for _, src := range []string{"rows/order", " |> std.where", "rows/order |>", "rows/order |> std .where", "rows/order |> std.where |>"} {
		if _, err := ParsePipelineSyntax(src); err == nil {
			t.Errorf("accepted invalid syntax %q", src)
		}
	}
}
