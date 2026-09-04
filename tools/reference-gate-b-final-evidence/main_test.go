package main

import (
	"errors"
	"reflect"
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

func TestEvidenceArtifactFingerprintUsesExactBytes(t *testing.T) {
	const want = "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := evidenceArtifactFingerprint([]byte("abc")); got != want {
		t.Fatalf("evidenceArtifactFingerprint(abc)=%q want=%q", got, want)
	}
	if evidenceArtifactFingerprint([]byte("abc")) == evidenceArtifactFingerprint([]byte("abc\n")) {
		t.Fatal("末尾改行の異なるEvidence artifactを同じfingerprintとして扱っています")
	}
}

func TestDecodeLaneArtifactFingerprintsOnlyValidEvidenceJSON(t *testing.T) {
	content := []byte(`{"evidenceNature":"deterministic-repository","repositoryRevision":"R","reproducibleRevision":true}`)
	lane := laneEvidence{}
	envelope, err := decodeLaneArtifact(content, &lane)
	if err != nil {
		t.Fatalf("decodeLaneArtifact() error=%v", err)
	}
	if envelope.EvidenceNature != "deterministic-repository" || envelope.RepositoryRevision != "R" || !envelope.ReproducibleRevision {
		t.Fatalf("decoded envelope=%#v", envelope)
	}
	if lane.EvidenceArtifactFingerprint != evidenceArtifactFingerprint(content) {
		t.Fatalf("fingerprint=%q want=%q", lane.EvidenceArtifactFingerprint, evidenceArtifactFingerprint(content))
	}
	if lane.ObservedNature != envelope.EvidenceNature || lane.RepositoryRevision != envelope.RepositoryRevision || !lane.ReproducibleRevision {
		t.Fatalf("lane evidenceがdecoded envelopeと一致しません: %#v", lane)
	}

	invalidLane := laneEvidence{}
	if _, err := decodeLaneArtifact([]byte("not-json"), &invalidLane); err == nil {
		t.Fatal("非JSON stdoutをEvidence artifactとして受理しています")
	}
	if invalidLane.EvidenceArtifactFingerprint != "" {
		t.Fatalf("非JSON stdoutにEvidence artifact fingerprintを付与しています: %q", invalidLane.EvidenceArtifactFingerprint)
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

func TestLaneSpecsMatchCanonicalRunnerInterfaces(t *testing.T) {
	want := []laneSpec{
		{
			Name:           "deterministic-repository",
			ExpectedNature: "deterministic-repository",
			Args:           []string{"run", "./tools/reference-gate-b-repository-evidence"},
		},
		{
			Name:           "generated-artifact-reproducibility",
			ExpectedNature: "generated-artifact-reproducibility",
			Args:           []string{"run", "./tools/reference-generated-artifact-evidence"},
		},
		{
			Name:           "real-postgres-ephemeral-fixture",
			ExpectedNature: "real-postgres-ephemeral-fixture",
			Args:           []string{"run", "./tools/reference-postgres-header-detail-runtime"},
		},
	}
	if got := laneSpecs(false); !reflect.DeepEqual(got, want) {
		t.Fatalf("laneSpecs(false) = %#v, want %#v", got, want)
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

func TestRunWithStopsBeforeNextLaneWhenPostRunWasNotFullyChecked(t *testing.T) {
	t.Setenv("TEI_TEST_POSTGRES_DSN", "postgres://evidence-only")

	laneCalls := 0
	result, err := runWith(
		false,
		func() (repositoryevidence.Source, error) {
			return repositoryevidence.Source{RepositoryRevision: "R", WorkingTreeClean: true}, nil
		},
		func(string) (repositoryevidence.PostRun, error) {
			return repositoryevidence.PostRun{
				Checked:            false,
				RepositoryRevision: "R",
				RevisionStable:     true,
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
		t.Fatal("runWith() error = nil, want unchecked post-run failure")
	}
	if laneCalls != 1 {
		t.Fatalf("lane calls = %d, want 1", laneCalls)
	}
	if result.PostRunRepository.Checked {
		t.Fatal("postRunRepository.checked = true, want false")
	}
	if result.AllLanesSameRevision {
		t.Fatal("allLanesSameRevision = true with unchecked post-run state, want false")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision = true with unchecked post-run state, want false")
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
