package syntax

import (
	"strings"
	"testing"

	"tei/internal/application"
)

// A pattern kind written in the application body must say where patterns go. The
// generic "expected database, uses, or binds" cannot tell a misplaced pattern
// apart from a typo, which is what a designer hit in round 60 (#4128).
func TestPatternKindInApplicationBodyNamesItsPlacement(t *testing.T) {
	for _, kind := range application.PatternKinds {
		src := "application order_entry {\n    uses semantic sales\n    " + kind + " one {\n        data sales.order\n    }\n}"
		_, err := NewParser(src).ParseApplication()
		if err == nil {
			t.Fatalf("%s: expected a parse error", kind)
		}
		message, ok := ErrorMessage(err)
		if !ok {
			t.Fatalf("%s: error has no message", kind)
		}
		if !strings.Contains(message, `"`+kind+`"`) {
			t.Errorf("%s: message does not quote the kind: %s", kind, message)
		}
		if !strings.Contains(message, "pattern <application>.<id> "+kind) {
			t.Errorf("%s: message does not show the declaration form: %s", kind, message)
		}
		if strings.Contains(message, "expected database, uses, or binds") {
			t.Errorf("%s: message is still the generic one: %s", kind, message)
		}
	}
}

// An unknown token is not a misplaced pattern, so it keeps the generic message.
func TestUnknownApplicationBodyTokenKeepsTheGenericMessage(t *testing.T) {
	src := "application order_entry { frobnicate }"
	_, err := NewParser(src).ParseApplication()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	message, ok := ErrorMessage(err)
	if !ok || message != "expected database, uses, or binds" {
		t.Fatalf("message=%q ok=%v", message, ok)
	}
}

// The range must still point at the offending token, not at the whole block.
func TestPatternPlacementErrorRangeCoversTheKind(t *testing.T) {
	src := "application order_entry {\n    record one {\n    }\n}"
	_, err := NewParser(src).ParseApplication()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	start, end, ok := ErrorRange(err)
	if !ok {
		t.Fatalf("error has no source range: %T %v", err, err)
	}
	if got := src[start.Offset:end.Offset]; got != "record" {
		t.Fatalf("range text=%q want %q", got, "record")
	}
}
