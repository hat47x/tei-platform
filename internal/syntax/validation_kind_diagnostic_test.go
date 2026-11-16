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

// Each kind parses with the body it requires. A kind rejected here would be
// named by the diagnostic but unusable, so the vocabulary has to be accepted,
// not merely listed.
func TestKnownValidationKindIsAccepted(t *testing.T) {
	// The body grammar accepts only `attribute` and `relation`. Every kind
	// requires an attribute; relation-unique additionally requires a relation.
	bodies := map[semantic.ValidationKind]string{
		semantic.ValidationPositive:       "    attribute probe.thing.id\n",
		semantic.ValidationRelationUnique: "    attribute probe.thing.id\n    relation probe.thing.items\n",
		semantic.ValidationOperator:       "    attribute probe.thing.id\n",
		semantic.ValidationNotAfter:       "    attribute probe.thing.starts_on\n",
		semantic.ValidationNoOverlap:      "    attribute probe.thing.starts_on\n",
	}

	for _, kind := range semantic.ValidationKinds {
		body, ok := bodies[kind]
		if !ok {
			t.Errorf("kind %q has no fixture; add the body it requires", kind)
			continue
		}
		src := "namespace probe\n\nvalidation probe.ok " + string(kind) + " {\n" + body + "}\n"
		if _, err := NewParser(src).ParseSemanticFile(); err != nil {
			t.Errorf("kind %q was rejected: %v", kind, err)
		}
	}
}
