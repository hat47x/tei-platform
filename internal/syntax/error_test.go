package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseErrorRangeHighlightsUnexpectedToken(t *testing.T) {
	src := "application order_entry { unknown }"
	_, err := syntax.NewParser(src).ParseApplication()
	if err == nil {
		t.Fatal("expected parse error")
	}
	start, end, ok := syntax.ErrorRange(err)
	if !ok {
		t.Fatalf("error has no source range: %T %v", err, err)
	}
	if got := src[start.Offset:end.Offset]; got != "unknown" {
		t.Fatalf("range text=%q start=%+v end=%+v", got, start, end)
	}
	message, ok := syntax.ErrorMessage(err)
	if !ok || message != "expected database or uses" {
		t.Fatalf("message=%q ok=%v", message, ok)
	}
}

func TestLexerErrorRangeHighlightsUnexpectedByte(t *testing.T) {
	src := "application order_entry { @ }"
	_, err := syntax.NewParser(src).ParseApplication()
	if err == nil {
		t.Fatal("expected lexer error")
	}
	start, end, ok := syntax.ErrorRange(err)
	if !ok {
		t.Fatalf("error has no source range: %T %v", err, err)
	}
	if got := src[start.Offset:end.Offset]; got != "@" {
		t.Fatalf("range text=%q start=%+v end=%+v", got, start, end)
	}
}
