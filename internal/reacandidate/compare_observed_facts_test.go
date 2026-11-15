package reacandidate

import "testing"

// A producer may reuse an external occurrence ID and declared Source digest
// while reporting a different relation fact. Candidate identity is not a
// substitute for observed facts or for current Owner acceptance.
func TestCompareCandidateSetsSameSourceChangedObservedRelation(t *testing.T) {
	before := mustGenerate(t, defaultEvidence())
	relationID := ""
	relationKind := ""
	for _, c := range before.Candidates {
		if c.Kind == CandidateKindCrossReference {
			relationID = c.CandidateID
			relationKind = c.Facts.ObservedKind
			break
		}
	}
	if relationID == "" || relationKind == "" {
		t.Fatal("test fixture needs one directed relation with a reported kind")
	}

	changedKind := "requires"
	if changedKind == relationKind {
		changedKind = "imports"
	}
	tests := []struct {
		name           string
		reuseAttempt   bool
		changeSource   bool
		wantStatus     string
	}{
		{"different-attempt-same-source", false, false, ComparisonObservedFactsChanged},
		{"reused-attempt-same-source", true, false, ComparisonObservedFactsChanged},
		{"changed-source-and-facts", false, true, ComparisonRevisionChanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := defaultEvidence()
			fixture.edges[0].rel = changedKind
			if !tt.reuseAttempt {
				fixture.evidenceID = "ev-minimal-facts-rerun"
			}
			if tt.changeSource {
				fixture.edges[0].artifact = digestB
			}
			after := mustGenerate(t, fixture)

			var beforeRelation, afterRelation *StructureCandidate
			for i := range before.Candidates {
				if before.Candidates[i].CandidateID == relationID {
					beforeRelation = &before.Candidates[i]
					break
				}
			}
			for i := range after.Candidates {
				if after.Candidates[i].CandidateID == relationID {
					afterRelation = &after.Candidates[i]
					break
				}
			}
			if beforeRelation == nil || afterRelation == nil {
				t.Fatalf("same external occurrence must retain candidate ID %q", relationID)
			}
			if beforeRelation.Facts.ObservedKind == afterRelation.Facts.ObservedKind {
				t.Fatalf("test relation facts did not change: %q", relationKind)
			}
			if beforeRelation.SourceRevision.ArtifactPath != afterRelation.SourceRevision.ArtifactPath {
				t.Fatal("test unexpectedly moved the source path")
			}
			if !tt.changeSource && beforeRelation.SourceRevision.Fingerprint != afterRelation.SourceRevision.Fingerprint {
				t.Fatal("test unexpectedly changed the declared source digest")
			}

			comparison := CompareCandidateSets(before, after)
			if comparison.Effect != EffectReadOnly {
				t.Fatalf("comparison effect = %q", comparison.Effect)
			}
			found := 0
			for _, entry := range comparison.Entries {
				if entry.CandidateID != relationID {
					continue
				}
				found++
				if entry.Status != tt.wantStatus {
					t.Fatalf("relation status = %q, want %q", entry.Status, tt.wantStatus)
				}
				for _, want := range []string{
					"observed-structural-facts-changed-under-reused-candidate-id",
					"source-backed-re-evaluation-is-required-before-meaning-or-edit",
				} {
					if !hasString(entry.Notes, want) {
						t.Fatalf("missing %q in %v", want, entry.Notes)
					}
				}
				if tt.changeSource && !hasString(entry.Notes, "fresh-owner-review-is-required") {
					t.Fatalf("source revision change lost review requirement: %v", entry.Notes)
				}
			}
			if found != 1 {
				t.Fatalf("relation entries = %d, want exactly one", found)
			}
		})
	}
}
