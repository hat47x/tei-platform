package semantic

import "testing"

// ValidationNoOverlap is a declared kind, and a kind that Valid() rejects is
// unreachable through authoring (#4032).
func TestValidationNoOverlapIsAValidKind(t *testing.T) {
	if !ValidationNoOverlap.Valid() {
		t.Fatalf("ValidationNoOverlap.Valid() = false, want true")
	}
	if got := string(ValidationNoOverlap); got != "no-overlap" {
		t.Errorf("ValidationNoOverlap = %q, want no-overlap", got)
	}
	// A kind has to be distinguishable from its neighbours: the overlap rule is
	// not the uniqueness rule.
	if ValidationNoOverlap == ValidationRelationUnique {
		t.Error("ValidationNoOverlap and ValidationRelationUnique are the same kind")
	}
	if ValidationNoOverlap == ValidationNotAfter {
		t.Error("ValidationNoOverlap and ValidationNotAfter are the same kind")
	}
}
