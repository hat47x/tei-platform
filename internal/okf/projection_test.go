package okf

import (
	"strings"
	"testing"
)

// TestProjectionIsCandidateOnly covers the single most important property of the
// projection: nothing it emits is an acceptance, a decision, or a grant.
func TestProjectionIsCandidateOnly(t *testing.T) {
	projection := Project(readFixture(t))

	if projection.Admission != "candidate" {
		t.Fatalf("projection admission = %q, want candidate", projection.Admission)
	}
	if projection.Profile != projectionProfileReadOnly {
		t.Fatalf("profile = %q", projection.Profile)
	}
	if len(projection.Candidates) == 0 {
		t.Fatal("the fixture must produce at least one candidate")
	}
	for _, candidate := range projection.Candidates {
		if candidate.Admission != "candidate" {
			t.Errorf("candidate %s admission = %q", candidate.ProposedID, candidate.Admission)
		}
		if !strings.HasPrefix(candidate.ProposedID, "candidate:") {
			t.Errorf("candidate ID %q is not visibly a proposal", candidate.ProposedID)
		}
		if candidate.Basis == "" {
			t.Errorf("candidate %s has no basis", candidate.ProposedID)
		}
		if candidate.Source.Digest == "" {
			t.Errorf("candidate %s has no source digest", candidate.ProposedID)
		}
		if len(candidate.RequiresReview) == 0 {
			t.Errorf("candidate %s claims no missing qualification", candidate.ProposedID)
		}
	}
	if len(projection.NotProposed) == 0 {
		t.Fatal("the projection must state what it does not propose")
	}
	for _, item := range projection.NotProposed {
		if strings.Contains(strings.ToLower(item), "accept") && projection.Admission == "candidate" {
			// The item is listed precisely because it is not produced.
			continue
		}
	}
}

// TestProjectionCoversTheFixtureShape covers one candidate of each kind the fixture
// can support, and the per-concept loss entries.
func TestProjectionCoversTheFixtureShape(t *testing.T) {
	bundle := readFixture(t)
	projection := Project(bundle)

	kinds := map[CandidateKind]int{}
	for _, candidate := range projection.Candidates {
		kinds[candidate.Kind]++
	}
	for _, concept := range bundle.Concepts {
		if kinds[CandidateSubject] < 1 {
			t.Fatal("no subject candidate")
		}
		_ = concept
	}
	if kinds[CandidateSubject] != 3 {
		t.Fatalf("subject candidates = %d, want one per concept", kinds[CandidateSubject])
	}
	// data/order declares one source entry.
	if kinds[CandidateSourceKind] != 1 {
		t.Fatalf("source candidates = %d, want 1", kinds[CandidateSourceKind])
	}
	// functions/submit and business/submit link to data/order.
	if kinds[CandidateRelation] != 2 {
		t.Fatalf("relation candidates = %d, want 2", kinds[CandidateRelation])
	}
	if kinds[CandidateComputation] != 0 {
		t.Fatalf("the fixture has no Attested Computation, got %d", kinds[CandidateComputation])
	}

	if len(projection.Losses) == 0 {
		t.Fatal("the projection must carry a loss manifest")
	}
	hasBodyLoss := false
	for _, loss := range projection.Losses {
		if strings.Contains(loss.Information, "body prose") {
			hasBodyLoss = true
		}
		if loss.Reason == "" || loss.ReturnTo == "" {
			t.Errorf("loss %q lacks a reason or a return path", loss.Information)
		}
	}
	if !hasBodyLoss {
		t.Fatal("the projection must state that the body prose is not translated")
	}
}

// TestProjectionRelationTargetsAreConcepts covers the relation rule: a link becomes
// a candidate only when it resolves to another concept, never on a guess.
func TestProjectionRelationTargetsAreConcepts(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"functions/submit.md": replaceInFile(t, "functions/submit.md",
			"A documentation-only example of submitting an [order](../data/order.md).",
			"A documentation-only example of submitting an [order](../data/order.md).\n\nSee [broken](../missing/other.md)."),
	})
	projection := Project(readFixture(t, mutated))
	for _, candidate := range projection.Candidates {
		if candidate.Kind != CandidateRelation {
			continue
		}
		if candidate.Source.Target == "../missing/other.md" {
			t.Fatal("a broken link must not become a relation candidate")
		}
	}
}

// TestProjectionOnAttestedComputation covers the computation candidate: the contract
// is proposed and the execution is explicitly outside the candidate.
func TestProjectionOnAttestedComputation(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"index.md": "# Concepts\n\n* [Computation](computation.md) - a declared computation\n",
		"computation.md": "---\ntype: Attested Computation\ntitle: Revenue\nruntime: python\n" +
			"parameters:\n  - { name: year, type: integer, required: true }\n" +
			"executor:\n  resource: run.md\n  receipt: [job_id, result]\n" +
			"attester:\n  resource: attester.md\n---\n\n# Computation\n\n    SELECT 1\n",
		"run.md":      "---\ntype: Skill\ntitle: runner\n---\n\nrun instructions\n",
		"attester.md": "---\ntype: Attester\ntitle: attester\n---\n\ncheck instructions\n",
	})
	bundle, err := Read(dir, Options{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	projection := Project(bundle)

	found := false
	for _, candidate := range projection.Candidates {
		if candidate.Kind != CandidateComputation {
			continue
		}
		found = true
		if candidate.Source.ConceptID != "computation" {
			t.Fatalf("computation candidate source = %q", candidate.Source.ConceptID)
		}
		if !strings.Contains(candidate.Basis, "python") {
			t.Fatalf("basis = %q, want the declared runtime", candidate.Basis)
		}
		for _, missing := range candidate.RequiresReview {
			switch {
			case strings.Contains(missing, "run"), strings.Contains(missing, "receipt"), strings.Contains(missing, "attester"), strings.Contains(missing, "re-read"):
			default:
				t.Errorf("unexpected requirement %q", missing)
			}
		}
	}
	if !found {
		t.Fatal("an attested computation must produce a contract candidate")
	}
	// The executor and attester files are concepts in this bundle, so they also
	// get subject candidates; the point is that the projection never claims a run.
	for _, candidate := range projection.Candidates {
		if strings.Contains(candidate.ProposedID, "receipt") && candidate.Kind != CandidateComputation {
			t.Errorf("unexpected receipt-shaped candidate %q", candidate.ProposedID)
		}
	}
}

// TestProjectionLossesCarryTrustAndStaleness covers the separation rule: a document
// label must appear in the loss manifest rather than in a candidate.
func TestProjectionLossesCarryTrustAndStaleness(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md",
			"status: draft",
			"status: deprecated\nverified: { by: human:someone, at: 2026-06-25T09:00:00Z }\nstale_after: 2026-12-31T00:00:00Z"),
	})
	projection := Project(readFixture(t, mutated))

	for _, candidate := range projection.Candidates {
		if strings.Contains(candidate.Basis, "human-reviewed") {
			t.Fatal("a trust tier must not appear as a candidate basis")
		}
	}

	var sawTrust, sawStaleness, sawStatus bool
	for _, loss := range projection.Losses {
		if loss.ConceptID != "data/order" {
			continue
		}
		switch {
		case strings.Contains(loss.Information, "trust tier"):
			sawTrust = true
		case strings.Contains(loss.Information, "staleness"):
			sawStaleness = true
		case strings.Contains(loss.Information, "lifecycle status"):
			sawStatus = true
		}
	}
	if !sawTrust {
		t.Fatal("the trust tier must be reported as a loss, not projected")
	}
	if !sawStaleness {
		t.Fatal("staleness must be reported as a loss, not projected")
	}
	if !sawStatus {
		t.Fatal("the deprecated status must be reported as a loss, not projected")
	}
}

// TestProjectionIsDeterministic covers ordering: the same bytes produce the same
// candidate order, so a diff of two projections is meaningful.
func TestProjectionIsDeterministic(t *testing.T) {
	first := Project(readFixture(t))
	second := Project(readFixture(t))
	if len(first.Candidates) != len(second.Candidates) {
		t.Fatalf("candidate counts differ: %d / %d", len(first.Candidates), len(second.Candidates))
	}
	for i := range first.Candidates {
		if first.Candidates[i].ProposedID != second.Candidates[i].ProposedID {
			t.Fatalf("candidate %d differs: %q / %q", i, first.Candidates[i].ProposedID, second.Candidates[i].ProposedID)
		}
	}
	if first.Revision.Value != second.Revision.Value {
		t.Fatal("the projection must carry the revision it was derived from")
	}
}

// TestProjectionReadsNothingExtra covers the non-effect rule at the API level: the
// projection consumes an already-read bundle, so calling it cannot read a file the
// bundle read did not.
func TestProjectionReadsNothingExtra(t *testing.T) {
	bundle := readFixture(t)
	projection := Project(bundle)
	for _, candidate := range projection.Candidates {
		if candidate.Source.Path == "" {
			t.Errorf("candidate %s has no source path", candidate.ProposedID)
		}
		known := false
		for _, concept := range bundle.Concepts {
			if concept.Path == candidate.Source.Path {
				known = true
			}
		}
		if !known {
			t.Errorf("candidate %s names %q, which is not a concept in the read bundle", candidate.ProposedID, candidate.Source.Path)
		}
	}
}
