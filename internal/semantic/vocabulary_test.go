package semantic

import "testing"

// TypeKinds and Cardinalities are quoted by their diagnostics, so each list and
// its Valid must be one answer (#4094).
func TestSemanticVocabularyListsAndValidAgree(t *testing.T) {
	typeKinds := []TypeKind{TypePrimitive, TypeAlias}
	if len(typeKinds) != len(TypeKinds) {
		t.Fatalf("TypeKinds has %d entries but %d are declared", len(TypeKinds), len(typeKinds))
	}
	for _, v := range TypeKinds {
		if !v.Valid() {
			t.Errorf("TypeKinds lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []TypeKind{"", "Primitive", "structure"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted type kind %q", bad)
		}
	}

	cardinalities := []Cardinality{CardinalityOne, CardinalityMany}
	if len(cardinalities) != len(Cardinalities) {
		t.Fatalf("Cardinalities has %d entries but %d are declared", len(Cardinalities), len(cardinalities))
	}
	for _, v := range Cardinalities {
		if !v.Valid() {
			t.Errorf("Cardinalities lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []Cardinality{"", "several", "One"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted cardinality %q", bad)
		}
	}
}
