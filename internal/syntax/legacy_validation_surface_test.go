package syntax

import (
	"strings"
	"testing"
)

// A legacy .tei validation carries an `attribute` and a `relation` only. Kinds
// that need a second attribute, a grouping, or an operator cannot be expressed
// there, so refusing them here points at a file the author can actually fix.
func TestLegacySurfaceRefusesKindsItCannotExpress(t *testing.T) {
	cases := map[string]string{
		"no-overlap": "second attribute",
		"not-after":  "second attribute",
		"operator":   "declared operator",
	}
	for kind, want := range cases {
		src := "namespace probe\n\nvalidation probe.rule " + kind + " {\n    attribute probe.thing.a\n}\n"
		_, err := NewParser(src).ParseSemanticFile()
		if err == nil {
			t.Fatalf("kind %q was accepted although this surface cannot express it", kind)
		}
		message, ok := ErrorMessage(err)
		if !ok {
			t.Fatalf("kind %q: error has no message", kind)
		}
		if !strings.Contains(message, want) {
			t.Errorf("kind %q: message does not name what it needs (%q): %s", kind, want, message)
		}
		if !strings.Contains(message, "tei.md") {
			t.Errorf("kind %q: message does not say where to declare it: %s", kind, message)
		}
	}
}

// The kinds this surface can carry keep working, so the refusal is not blanket.
func TestLegacySurfaceStillAcceptsKindsItCanExpress(t *testing.T) {
	// positive rejects a relation, so each kind gets the body it accepts.
	bodies := map[string]string{
		"positive":        "    attribute probe.thing.a\n",
		"relation-unique": "    attribute probe.thing.a\n    relation probe.thing.items\n",
	}
	for kind, body := range bodies {
		src := "namespace probe\n\nvalidation probe.rule " + kind + " {\n" + body + "}\n"
		if _, err := NewParser(src).ParseSemanticFile(); err != nil {
			t.Errorf("kind %q was refused although this surface can express it: %v", kind, err)
		}
	}
}
