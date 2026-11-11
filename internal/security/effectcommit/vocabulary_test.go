package effectcommit

import "testing"

// Provenance labels where evidence came from and ErrorClass says why a commit
// failed. Both are read by a caller deciding what to report, so every value is
// named here (#4032).
func TestEvidenceVocabulariesAreCovered(t *testing.T) {
	t.Run("provenance", func(t *testing.T) {
		values := []Provenance{
			ProvenanceUnspecified, ProvenanceSynthetic, ProvenanceRealPG, ProvenanceProvisionalGrant,
		}
		want := []string{"unspecified", "synthetic", "real-pg", "provisional-grant"}
		if len(values) != len(want) {
			t.Fatalf("a provenance value was dropped: %d values", len(values))
		}
		for i, value := range values {
			if string(value) != want[i] {
				t.Fatalf("values[%d]=%q want %q", i, value, want[i])
			}
		}
		// A real PostgreSQL observation must not share its label with a
		// synthetic one, or the "real" claim would be unusable.
		if ProvenanceRealPG == ProvenanceSynthetic {
			t.Fatal("real and synthetic provenance share a value")
		}
	})

	t.Run("error classes", func(t *testing.T) {
		classes := []ErrorClass{
			ErrorClassInvalidRequest, ErrorClassMissingProvider, ErrorClassCommitUncertain,
			ErrorClassContext, ErrorClassExecution,
		}
		want := []string{"invalid-request", "missing-provider", "commit-uncertain", "context", "execution-error"}
		if len(classes) != len(want) {
			t.Fatalf("an error class was dropped: %d values", len(classes))
		}
		seen := map[string]bool{}
		for i, class := range classes {
			if string(class) != want[i] {
				t.Fatalf("classes[%d]=%q want %q", i, class, want[i])
			}
			if seen[string(class)] {
				t.Fatalf("error class %q is declared twice", class)
			}
			seen[string(class)] = true
		}
	})
}
