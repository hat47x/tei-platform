package main

import (
	"errors"
	"testing"

	"tei/tools/internal/repositoryevidence"
)

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

func TestRunWithRechecksSourceBeforeReturningLaneFailure(t *testing.T) {
	t.Setenv("TEI_TEST_POSTGRES_DSN", "postgres://evidence-only")

	postCalls := 0
	laneCalls := 0
	result, err := runWith(
		false,
		func() (repositoryevidence.Source, error) {
			return repositoryevidence.Source{RepositoryRevision: "R", WorkingTreeClean: true}, nil
		},
		func(initial string) (repositoryevidence.PostRun, error) {
			postCalls++
			if initial != "R" {
				t.Fatalf("InspectPost initial = %q, want R", initial)
			}
			return cleanPostRun("R"), nil
		},
		func(spec laneSpec, candidate string) (laneEvidence, error) {
			laneCalls++
			return laneEvidence{Name: spec.Name}, errors.New("lane failed")
		},
		fakeGoVersion,
	)
	if err == nil {
		t.Fatal("runWith() error = nil, want lane failure")
	}
	if laneCalls != 1 {
		t.Fatalf("lane calls = %d, want 1", laneCalls)
	}
	if postCalls != 1 {
		t.Fatalf("post-run source checks = %d, want 1", postCalls)
	}
	if !result.PostRunRepository.Checked {
		t.Fatal("postRunRepository.checked = false, want true after lane failure")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision = true on lane failure, want false")
	}
}

func TestRunWithStopsBeforeNextLaneWhenSourceDrifts(t *testing.T) {
	t.Setenv("TEI_TEST_POSTGRES_DSN", "postgres://evidence-only")

	laneCalls := 0
	result, err := runWith(
		false,
		func() (repositoryevidence.Source, error) {
			return repositoryevidence.Source{RepositoryRevision: "R", WorkingTreeClean: true}, nil
		},
		func(string) (repositoryevidence.PostRun, error) {
			return repositoryevidence.PostRun{
				Checked:            true,
				RepositoryRevision: "R2",
				RevisionStable:     false,
				WorkingTreeClean:   true,
			}, nil
		},
		func(spec laneSpec, candidate string) (laneEvidence, error) {
			laneCalls++
			return passedLane(spec, candidate), nil
		},
		fakeGoVersion,
	)
	if err == nil {
		t.Fatal("runWith() error = nil, want source drift failure")
	}
	if laneCalls != 1 {
		t.Fatalf("lane calls = %d, want 1", laneCalls)
	}
	if result.AllLanesSameRevision {
		t.Fatal("allLanesSameRevision = true after source drift, want false")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision = true after source drift, want false")
	}
}

func TestRunWithMarksReproducibleOnlyAfterThreeLanesAndFinalSourceCheck(t *testing.T) {
	t.Setenv("TEI_TEST_POSTGRES_DSN", "postgres://evidence-only")

	postCalls := 0
	laneCalls := 0
	result, err := runWith(
		true,
		func() (repositoryevidence.Source, error) {
			return repositoryevidence.Source{RepositoryRevision: "R", WorkingTreeClean: true}, nil
		},
		func(string) (repositoryevidence.PostRun, error) {
			postCalls++
			return cleanPostRun("R"), nil
		},
		func(spec laneSpec, candidate string) (laneEvidence, error) {
			laneCalls++
			return passedLane(spec, candidate), nil
		},
		fakeGoVersion,
	)
	if err != nil {
		t.Fatalf("runWith() error = %v", err)
	}
	if laneCalls != 3 {
		t.Fatalf("lane calls = %d, want 3", laneCalls)
	}
	if postCalls != 4 {
		t.Fatalf("post-run source checks = %d, want 4 (each lane + final)", postCalls)
	}
	if !result.AllLanesSameRevision {
		t.Fatal("allLanesSameRevision = false, want true")
	}
	if !result.ReproducibleRevision {
		t.Fatal("reproducibleRevision = false, want true")
	}
	if !result.BroadRequested {
		t.Fatal("broadRequested = false, want true")
	}
}

func cleanPostRun(revision string) repositoryevidence.PostRun {
	return repositoryevidence.PostRun{
		Checked:            true,
		RepositoryRevision: revision,
		RevisionStable:     true,
		WorkingTreeClean:   true,
	}
}

func passedLane(spec laneSpec, revision string) laneEvidence {
	return laneEvidence{
		Name:                 spec.Name,
		ExpectedNature:       spec.ExpectedNature,
		ObservedNature:       spec.ExpectedNature,
		RepositoryRevision:   revision,
		ReproducibleRevision: true,
		Passed:               true,
	}
}

func fakeGoVersion(name string, args ...string) (string, error) {
	if name != "go" || len(args) != 1 || args[0] != "version" {
		return "", errors.New("unexpected command")
	}
	return "go version go1.23.2 linux/amd64", nil
}
