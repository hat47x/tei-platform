package application

import "testing"

// PatternKinds and Valid must stay one answer. Two copies would let the parser
// advertise a kind it rejects, or reject a kind it never mentions.
func TestPatternKindsAndValidAgree(t *testing.T) {
	for _, kind := range PatternKinds {
		if !PatternKind(kind).Valid() {
			t.Errorf("PatternKinds lists %q but Valid() rejects it", kind)
		}
	}
	// Every declared constant must appear in the list, so adding a kind without
	// listing it is caught here rather than in a confusing author error.
	declared := []PatternKind{PatternBrowse, PatternRecord, PatternReference, PatternCommand, PatternHeaderDetail}
	if len(declared) != len(PatternKinds) {
		t.Fatalf("PatternKinds has %d entries but %d kinds are declared; update both together",
			len(PatternKinds), len(declared))
	}
	seen := map[string]bool{}
	for _, kind := range PatternKinds {
		seen[kind] = true
	}
	for _, kind := range declared {
		if !seen[string(kind)] {
			t.Errorf("declared kind %q is missing from PatternKinds", kind)
		}
	}
	for _, bogus := range []string{"", "bogus", "Browse", "browse "} {
		if PatternKind(bogus).Valid() {
			t.Errorf("Valid() accepted %q", bogus)
		}
	}
}
