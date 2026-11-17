package syntax

import (
	"strings"
	"testing"
)

// A file in the semantic root that starts with a declaration this product
// defines elsewhere is in the wrong place, not misspelled. Naming the place
// turns "expected \"namespace\"" into an instruction (#4169).
func TestSemanticRootNamesThePlacementOfAnotherDeclaration(t *testing.T) {
	for keyword, want := range map[string]string{
		"pattern":     "applications/",
		"data":        "semantic/",
		"query":       "applications/",
		"contract":    "databases/",
		"environment": "environments/",
	} {
		src := keyword + " Thing {\n}\n"
		_, err := NewParser(src).ParseSemanticFile()
		if err == nil {
			t.Fatalf("%q at the top of a semantic file was accepted", keyword)
		}
		message, ok := ErrorMessage(err)
		if !ok {
			t.Fatalf("%q: error has no message", keyword)
		}
		if strings.Contains(message, `expected "namespace"`) && !strings.Contains(message, "cannot be declared") {
			t.Errorf("%q kept the generic message: %s", keyword, message)
		}
		if !strings.Contains(message, want) {
			t.Errorf("%q does not name where it belongs (%q): %s", keyword, want, message)
		}
	}
}

// A word this product does not declare keeps the generic parse message, so the
// new branch does not swallow a genuine misspelling.
func TestSemanticRootKeepsTheGenericMessageForAnUnknownWord(t *testing.T) {
	message, ok := ErrorMessage(mustParseErr(t, "frobnicate Thing\n"))
	if !ok {
		t.Fatal("error has no message")
	}
	if !strings.Contains(message, `expected "namespace"`) {
		t.Fatalf("an unknown word no longer gets the generic message: %s", message)
	}
	if strings.Contains(message, "cannot be declared") {
		t.Errorf("an unknown word claims a placement: %s", message)
	}
}

func mustParseErr(t *testing.T, src string) error {
	t.Helper()
	_, err := NewParser(src).ParseSemanticFile()
	if err == nil {
		t.Fatalf("source was accepted: %q", src)
	}
	return err
}

// `namespace` is itself a keyword in the declaration table, so the placement
// check would claim "namespace starts a semantic namespace, which cannot be
// declared..." for a CORRECT file if it ran without excluding it. This pins
// that exclusion: a valid file must still be accepted.
func TestSemanticRootAcceptsItsOwnKeyword(t *testing.T) {
	if _, err := NewParser("namespace equip\n\ndata Loan {\n}\n").ParseSemanticFile(); err != nil {
		t.Fatalf("a valid semantic file was refused: %v", err)
	}
}
