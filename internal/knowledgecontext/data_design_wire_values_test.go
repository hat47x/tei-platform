package knowledgecontext

import (
	"encoding/json"
	"testing"

	"tei/internal/sourceindex"
)

// The data-design projection revisions and candidate kinds are wire values a
// reader branches on. They are used through constants, so the literals are pinned
// here to make a rename a visible change (#4032).
func TestDataDesignWireRevisionsAreFixed(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"context projection", dataDesignProjectionRevision, "tei.knowledge.data-design/v7"},
		{"candidate review projection", dataDesignCandidateReviewProjectionRevision, "tei.knowledge.data-design-candidate-review/v1"},
		{"business identity candidate kind", DataDesignCandidateKindBusinessIdentity, "business-identity"},
		{"functional dependency candidate kind", DataDesignCandidateKindFunctionalDependency, "functional-dependency"},
	}
	for _, testCase := range cases {
		if testCase.got != testCase.want {
			t.Errorf("%s = %q, want %q", testCase.name, testCase.got, testCase.want)
		}
	}
}

// A candidate fingerprint is computed over a body stamped with the candidate's
// schema identifier, so a change to that identifier would silently change every
// stored fingerprint. This test recomputes the fingerprint from the literal and
// compares it with the production function, which pins the identifier to its
// value (#4032).
func TestCandidateFingerprintsUseThePublishedSchema(t *testing.T) {
	type stamped struct {
		Kind      string `json:"kind"`
		Candidate any    `json:"candidate"`
	}
	recompute := func(kind string, candidate any) string {
		encoded, err := json.Marshal(stamped{Kind: kind, Candidate: candidate})
		if err != nil {
			t.Fatal(err)
		}
		return sourceindex.Fingerprint(encoded)
	}

	businessIdentity := recompute("tei.knowledge.data-design.business-identity-candidate/v2", DataDesignBusinessIdentityCandidate{})
	if got := businessIdentityCandidateFingerprint(DataDesignBusinessIdentityCandidate{}); got != businessIdentity {
		t.Fatalf("business identity fingerprint=%q, want %q", got, businessIdentity)
	}
	dependency := recompute("tei.knowledge.data-design.functional-dependency-candidate/v2", DataDesignFunctionalDependencyCandidate{})
	if got := functionalDependencyCandidateFingerprint(DataDesignFunctionalDependencyCandidate{}); got != dependency {
		t.Fatalf("functional dependency fingerprint=%q, want %q", got, dependency)
	}
	if businessIdentity == dependency {
		t.Fatal("the two candidate kinds produced the same fingerprint")
	}
}
