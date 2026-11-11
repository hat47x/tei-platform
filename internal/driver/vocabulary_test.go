package driver

import "testing"

// ObservationEvidenceNature says where an observation came from. A reader must
// be able to tell a synthetic observation from a real one, so every value has to
// be named here (#4032).
func TestObservationEvidenceNaturesAreCovered(t *testing.T) {
	natures := []ObservationEvidenceNature{
		ObservationEvidenceSynthetic, ObservationEvidenceExternalObserved, ObservationEvidenceManagedHere,
	}
	want := []string{"synthetic", "external-observed", "managed-here"}
	if len(natures) != len(want) {
		t.Fatalf("a nature was dropped: %d values", len(natures))
	}
	for i, nature := range natures {
		if string(nature) != want[i] {
			t.Fatalf("natures[%d]=%q want %q", i, nature, want[i])
		}
	}
	// The three values must stay distinguishable: a synthetic observation that
	// looked like a real one would be an over-claim.
	if natures[0] == natures[1] || natures[1] == natures[2] || natures[0] == natures[2] {
		t.Fatal("two observation natures share a value")
	}
}
