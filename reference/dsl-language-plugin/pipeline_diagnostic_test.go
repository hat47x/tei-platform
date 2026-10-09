package languagereference

import (
	"errors"
	"testing"
)

func TestPipelineSyntaxDiagnosticSpan(t *testing.T) {
	source := "　rows/order |> std.where |> bad name"
	_, err := ParsePipelineSyntax(source)
	if err == nil {
		t.Fatal("expected invalid operator")
	}
	var diagnostic *PipelineDiagnostic
	if !errors.As(err, &diagnostic) {
		t.Fatalf("expected located diagnostic, got %T: %v", err, err)
	}
	if diagnostic.Step != 1 || source[diagnostic.Span.Start:diagnostic.Span.End] != "bad name" {
		t.Fatalf("wrong location: %+v", diagnostic)
	}
}

func TestPinnedLoweringDiagnosticPreservesCauseAndSpan(t *testing.T) {
	source := "　rows/order |> std.where |> std.expand"
	syntax, err := ParsePipelineSyntax(source)
	if err != nil {
		t.Fatal(err)
	}
	where := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "where-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	expand := OperatorContract{ID: "std.expand", Version: "1", Fingerprint: "expand-v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/line"}
	_, err = LowerPinnedPipeline(syntax, []OperatorContract{where, expand}, []OperatorContract{where})
	if !errors.Is(err, ErrOperatorMissing) {
		t.Fatalf("expected missing contract cause, got %v", err)
	}
	var diagnostic *PipelineDiagnostic
	if !errors.As(err, &diagnostic) {
		t.Fatalf("expected located diagnostic, got %T: %v", err, err)
	}
	if diagnostic.Step != 1 || source[diagnostic.Span.Start:diagnostic.Span.End] != "std.expand" {
		t.Fatalf("wrong failure location: %+v", diagnostic)
	}
}

func TestPinnedLoweringMismatchDiagnostic(t *testing.T) {
	source := "rows/order |> std.where"
	syntax, err := ParsePipelineSyntax(source)
	if err != nil {
		t.Fatal(err)
	}
	pin := OperatorContract{ID: "std.other", Version: "1", Fingerprint: "v1", Effect: "pure", InputType: "rows/order", OutputType: "rows/order"}
	_, err = LowerPinnedPipeline(syntax, []OperatorContract{pin}, []OperatorContract{pin})
	if !errors.Is(err, ErrOperatorInvalid) {
		t.Fatalf("expected invalid contract, got %v", err)
	}
	var diagnostic *PipelineDiagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Step != 0 || source[diagnostic.Span.Start:diagnostic.Span.End] != "std.where" {
		t.Fatalf("wrong failure location: %+v, %v", diagnostic, err)
	}
}
