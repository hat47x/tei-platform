package syntax

import (
	"strings"
	"testing"

	"tei/internal/application"
)

// A pattern needs a kind before its body. Before this diagnostic named the
// accepted set, an author only saw `unexpected token "{"` and had to read the
// source or another workspace file to learn what a kind could be (#4078).
func TestPatternWithoutAKindNamesTheValidKinds(t *testing.T) {
	_, err := NewParser("pattern order_entry.revoke {\n    capability security.revoke\n}").ParseApplicationPattern()
	if err == nil {
		t.Fatal("a pattern with no kind parsed; it must be rejected")
	}
	message := err.Error()
	if !strings.Contains(message, "requires a kind") {
		t.Errorf("error does not explain that a kind is required: %v", err)
	}
	for _, kind := range application.PatternKinds {
		if !strings.Contains(message, kind) {
			t.Errorf("error does not name the valid kind %q: %v", kind, err)
		}
	}
}

// An unrecognised kind must also name the accepted set, so a typo is fixable
// without leaving the CLI.
func TestUnsupportedPatternKindNamesTheValidKinds(t *testing.T) {
	_, err := NewParser("pattern order_entry.revoke bogus {\n    capability security.revoke\n}").ParseApplicationPattern()
	if err == nil {
		t.Fatal("an unsupported kind parsed; it must be rejected")
	}
	message := err.Error()
	if !strings.Contains(message, `"bogus"`) {
		t.Errorf("error does not quote the rejected kind: %v", err)
	}
	for _, kind := range application.PatternKinds {
		if !strings.Contains(message, kind) {
			t.Errorf("error does not name the valid kind %q: %v", kind, err)
		}
	}
}

// A valid kind must still be accepted. Each kind then demands its own fields
// (browse/record need data, reference/header-detail need relation), so this
// checks only that the kind itself was not rejected: the added pre-check must
// not turn a valid kind into a diagnostic about kinds.
func TestValidPatternKindsAreNotRejectedAsKinds(t *testing.T) {
	for _, kind := range application.PatternKinds {
		source := "pattern order_entry.probe " + kind + " {\n    capability security.revoke\n}"
		_, err := NewParser(source).ParseApplicationPattern()
		if err == nil {
			continue // the kind was accepted and the body happened to satisfy it
		}
		message := err.Error()
		if strings.Contains(message, "kind") {
			t.Errorf("kind %q was rejected as a kind: %v", kind, err)
		}
	}
}
