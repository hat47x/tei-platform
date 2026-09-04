package main

import "testing"

func TestVerifyLaneEnvelopeAcceptsExpectedCandidateRevision(t *testing.T) {
	envelope := laneEnvelope{
		EvidenceNature:       "deterministic-repository",
		RepositoryRevision:   "abc123",
		ReproducibleRevision: true,
	}
	if err := verifyLaneEnvelope(envelope, "deterministic-repository", "abc123"); err != nil {
		t.Fatalf("verifyLaneEnvelope() error = %v", err)
	}
}

func TestVerifyLaneEnvelopeRejectsDifferentNature(t *testing.T) {
	envelope := laneEnvelope{
		EvidenceNature:       "generated-artifact-reproducibility",
		RepositoryRevision:   "abc123",
		ReproducibleRevision: true,
	}
	if err := verifyLaneEnvelope(envelope, "deterministic-repository", "abc123"); err == nil {
		t.Fatal("verifyLaneEnvelope() error = nil, want evidenceNature mismatch")
	}
}

func TestVerifyLaneEnvelopeRejectsDifferentRevision(t *testing.T) {
	envelope := laneEnvelope{
		EvidenceNature:       "deterministic-repository",
		RepositoryRevision:   "old-revision",
		ReproducibleRevision: true,
	}
	if err := verifyLaneEnvelope(envelope, "deterministic-repository", "candidate"); err == nil {
		t.Fatal("verifyLaneEnvelope() error = nil, want repositoryRevision mismatch")
	}
}

func TestVerifyLaneEnvelopeRejectsNonReproducibleEvidence(t *testing.T) {
	envelope := laneEnvelope{
		EvidenceNature:       "deterministic-repository",
		RepositoryRevision:   "abc123",
		ReproducibleRevision: false,
	}
	if err := verifyLaneEnvelope(envelope, "deterministic-repository", "abc123"); err == nil {
		t.Fatal("verifyLaneEnvelope() error = nil, want reproducibleRevision failure")
	}
}

func TestLaneSpecsKeepPostgresLast(t *testing.T) {
	specs := laneSpecs(false)
	if len(specs) != 3 {
		t.Fatalf("len(laneSpecs()) = %d, want 3", len(specs))
	}
	if got := specs[0].Name; got != "deterministic-repository" {
		t.Fatalf("first lane = %q, want deterministic-repository", got)
	}
	if got := specs[1].Name; got != "generated-artifact-reproducibility" {
		t.Fatalf("second lane = %q, want generated-artifact-reproducibility", got)
	}
	if got := specs[2].Name; got != "real-postgres-ephemeral-fixture" {
		t.Fatalf("last lane = %q, want real-postgres-ephemeral-fixture", got)
	}
}

func TestLaneSpecsForwardBroadOnlyToRepositoryEvidence(t *testing.T) {
	specs := laneSpecs(true)
	if got := specs[0].Args[len(specs[0].Args)-1]; got != "-broad" {
		t.Fatalf("repository lane last arg = %q, want -broad", got)
	}
	for _, spec := range specs[1:] {
		for _, arg := range spec.Args {
			if arg == "-broad" {
				t.Fatalf("%s unexpectedly received -broad", spec.Name)
			}
		}
	}
}

func TestLanesConvergeRequiresThreePassedLanesOnSameRevision(t *testing.T) {
	lanes := []laneEvidence{
		{Name: "deterministic-repository", RepositoryRevision: "R", ReproducibleRevision: true, Passed: true},
		{Name: "generated-artifact-reproducibility", RepositoryRevision: "R", ReproducibleRevision: true, Passed: true},
		{Name: "real-postgres-ephemeral-fixture", RepositoryRevision: "R", ReproducibleRevision: true, Passed: true},
	}
	if !lanesConverge(lanes, "R") {
		t.Fatal("lanesConverge() = false, want true")
	}

	lanes[2].RepositoryRevision = "R2"
	if lanesConverge(lanes, "R") {
		t.Fatal("lanesConverge() = true with mixed revisions, want false")
	}
}
