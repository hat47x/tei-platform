package syntax

import (
	"strings"
	"testing"

	"tei/internal/semantic"
)

// The accepted kinds are a closed vocabulary of five words. "unsupported" alone
// makes the author guess; the diagnostic has to name them, the same way the
// plugin and type diagnostics do.
func TestUnknownValidationKindNamesTheVocabulary(t *testing.T) {
	src := "namespace probe\n\nvalidation probe.thing_id_ok frobnicate\n"
	_, err := NewParser(src).ParseSemanticFile()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	message, ok := ErrorMessage(err)
	if !ok {
		t.Fatal("error has no message")
	}
	if !strings.Contains(message, "valid values are") {
		t.Fatalf("the diagnostic does not name the accepted kinds: %s", message)
	}
	for _, kind := range semantic.ValidationKinds {
		if !strings.Contains(message, string(kind)) {
			t.Errorf("the diagnostic omits %q: %s", kind, message)
		}
	}
	// It still names what was written.
	if !strings.Contains(message, "frobnicate") {
		t.Errorf("the diagnostic does not name the rejected word: %s", message)
	}
}

// A legacy .tei validation carries `attribute` and `relation` only, so only the
// kinds expressible from that body can be declared here. The rest are named by
// the vocabulary diagnostic and refused with a message pointing at .tei.md.
// (An earlier version of this test asserted all five parsed here; that was
// wrong, and the parser now refuses the three that never could.)
func TestKnownValidationKindIsAccepted(t *testing.T) {
	bodies := map[string]string{
		"positive":        "    attribute probe.thing.id\n",
		"relation-unique": "    attribute probe.thing.id\n    relation probe.thing.items\n",
	}
	for kind, body := range bodies {
		src := "namespace probe\n\nvalidation probe.ok " + kind + " {\n" + body + "}\n"
		if _, err := NewParser(src).ParseSemanticFile(); err != nil {
			t.Errorf("kind %q was rejected: %v", kind, err)
		}
	}
	for _, kind := range []string{"operator", "not-after", "no-overlap"} {
		src := "namespace probe\n\nvalidation probe.ok " + kind + " {\n    attribute probe.thing.id\n}\n"
		if _, err := NewParser(src).ParseSemanticFile(); err == nil {
			t.Errorf("kind %q was accepted although this surface cannot express it", kind)
		}
	}
}
