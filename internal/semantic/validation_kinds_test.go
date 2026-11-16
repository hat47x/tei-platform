package semantic

import "testing"

// ValidationKinds is the single source both Valid() and the diagnostics read.
// A kind that Valid() accepts but the list omits would be accepted and never
// named; a kind in the list that Valid() rejects would be named and refused.
func TestValidationKindsIsTheSingleSource(t *testing.T) {
	if len(ValidationKinds) == 0 {
		t.Fatal("ValidationKinds is empty")
	}
	seen := map[ValidationKind]bool{}
	for _, kind := range ValidationKinds {
		if seen[kind] {
			t.Errorf("%q appears twice in ValidationKinds", kind)
		}
		seen[kind] = true
		if !kind.Valid() {
			t.Errorf("%q is listed but Valid() rejects it", kind)
		}
	}
	// Every declared kind must be in the list: the five constants are the
	// vocabulary, so an omission makes one of them unusable.
	for _, kind := range []ValidationKind{
		ValidationPositive,
		ValidationRelationUnique,
		ValidationOperator,
		ValidationNotAfter,
		ValidationNoOverlap,
	} {
		if !seen[kind] {
			t.Errorf("%q is declared but missing from ValidationKinds", kind)
		}
	}
	// An unknown kind stays rejected.
	if ValidationKind("frobnicate").Valid() {
		t.Error(`ValidationKind("frobnicate").Valid() = true`)
	}
}
