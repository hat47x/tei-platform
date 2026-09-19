package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type replayableBehaviorR7ACheck struct {
	Name   string
	Passed bool
}

type replayableBehaviorR7AUnresolved struct {
	ID       string
	Blocking bool
}

type replayableBehaviorR7AGateCReadiness struct {
	EvidenceNature             string
	HandoverBaselineID         string
	BaselineRepositoryRevision string
	TargetBuildRevision        string
	TargetAdapter              string
	Host                       string
	SemanticFunctionalRun      string
	GateCReady                 bool
	Checks                     []replayableBehaviorR7ACheck
	Unresolved                 []replayableBehaviorR7AUnresolved
}

type replayableBehaviorR7AArtifactDigest struct {
	File   string
	SHA256 string
}

type replayableBehaviorR7ASemanticFunctionalSummary struct {
	EvidenceNature             string
	RunIdentity                string
	HandoverBaselineID         string
	BaselineRepositoryRevision string
	TargetBuildRevision        string
	TargetAdapter              string
	Host                       string
	SourceQualificationPassed  bool
	SemanticConformancePassed  bool
	FunctionalEvidencePassed   bool
	Artifacts                  []replayableBehaviorR7AArtifactDigest
}

type replayableBehaviorR7AEntryBundle struct {
	Readiness replayableBehaviorR7AGateCReadiness
	Semantic  replayableBehaviorR7ASemanticFunctionalSummary
}

type replayableBehaviorR7ASpec struct {
	BehaviorID           string
	SemanticOperation    string
	TargetRealizationRef string
	RawArtifactFile      string
	ObservationClass     string
}

type replayableBehaviorR7ACandidate struct {
	BehaviorID                 string
	SemanticOperation          string
	TargetRealizationRef       string
	RawArtifactFile            string
	RawArtifactSHA256          string
	ObservationClass           string
	ReplayStatus               string
	CoverageClass              string
	HandoverBaselineID         string
	BaselineRepositoryRevision string
	TargetBuildRevision        string
	SemanticFunctionalRun      string
}

func TestReplayableBehaviorR7ARequiresActualGateCReadinessBeforeCandidateDerivation(t *testing.T) {
	bundle := replayableBehaviorR7ATestBundle()
	bundle.Readiness.GateCReady = false

	if _, err := replayableBehaviorR7ADeriveCandidates(
		bundle,
		replayableBehaviorR7AFirstWebSpecs(),
	); err == nil {
		t.Fatal("R7A accepted Gate C input that is not ready")
	}
}

func TestReplayableBehaviorR7ARequiresSameBRTRunAndRawArtifactDigests(t *testing.T) {
	t.Run("semantic-run-mismatch", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Semantic.RunIdentity = "different-semantic-functional-run"

		if _, err := replayableBehaviorR7ADeriveCandidates(
			bundle,
			replayableBehaviorR7AFirstWebSpecs(),
		); err == nil {
			t.Fatal("R7A accepted mismatched Semantic / Functional run identity")
		}
	})

	t.Run("target-revision-mismatch", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Semantic.TargetBuildRevision = "different-T"

		if _, err := replayableBehaviorR7ADeriveCandidates(
			bundle,
			replayableBehaviorR7AFirstWebSpecs(),
		); err == nil {
			t.Fatal("R7A accepted mismatched Target build revision")
		}
	})

	t.Run("raw-artifact-digest-missing", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		for i := range bundle.Semantic.Artifacts {
			if bundle.Semantic.Artifacts[i].File == "submit-success.json" {
				bundle.Semantic.Artifacts[i].SHA256 = ""
			}
		}

		if _, err := replayableBehaviorR7ADeriveCandidates(
			bundle,
			replayableBehaviorR7AFirstWebSpecs(),
		); err == nil {
			t.Fatal("R7A accepted behavior without exact raw artifact digest")
		}
	})
}

func TestReplayableBehaviorR7ACandidatesPreserveSemanticAndTargetRealizationSeparately(t *testing.T) {
	bundle := replayableBehaviorR7ATestBundle()
	specs := replayableBehaviorR7AFirstWebSpecs()

	got, err := replayableBehaviorR7ADeriveCandidates(bundle, specs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(specs) {
		t.Fatalf("R7A candidate count=%d, want %d", len(got), len(specs))
	}

	for i := range got {
		if got[i].SemanticOperation != specs[i].SemanticOperation {
			t.Fatalf("R7A semantic operation drifted: got=%q want=%q", got[i].SemanticOperation, specs[i].SemanticOperation)
		}
		if got[i].TargetRealizationRef != specs[i].TargetRealizationRef {
			t.Fatalf("R7A target realization drifted: got=%q want=%q", got[i].TargetRealizationRef, specs[i].TargetRealizationRef)
		}
		if got[i].RawArtifactSHA256 == "" {
			t.Fatalf("R7A candidate lost raw artifact fingerprint: %+v", got[i])
		}
		if got[i].ReplayStatus != "not-executed" {
			t.Fatalf("R7A source pressure fabricated replay execution status: %+v", got[i])
		}
		if got[i].CoverageClass != "behavior-unit" {
			t.Fatalf("R7A candidate fabricated broader workflow coverage: %+v", got[i])
		}
	}
}

func TestReplayableBehaviorR7ADoesNotTreatSelectorOrSurfaceMotorIdentityAsCanonical(t *testing.T) {
	typ := reflect.TypeOf(replayableBehaviorR7ACandidate{})
	for _, forbidden := range []string{
		"Selector",
		"DOMSelector",
		"Coordinate",
		"ClickSequence",
		"CanonicalSelector",
		"CanonicalRoute",
		"CanonicalComponent",
		"WorkflowPassed",
		"ReplayPassed",
		"FunctionalPassed",
		"GateCReady",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R7A candidate leaked forbidden field %q", forbidden)
		}
	}
}

func TestReplayableBehaviorR7ABlockingUnresolvedDoesNotEnterReplay(t *testing.T) {
	bundle := replayableBehaviorR7ATestBundle()
	bundle.Readiness.Unresolved = append(
		bundle.Readiness.Unresolved,
		replayableBehaviorR7AUnresolved{
			ID:       "test-only-blocking-unresolved",
			Blocking: true,
		},
	)

	if _, err := replayableBehaviorR7ADeriveCandidates(
		bundle,
		replayableBehaviorR7AFirstWebSpecs(),
	); err == nil {
		t.Fatal("R7A accepted blocking unresolved despite Gate C entry contract")
	}
}

func replayableBehaviorR7ADeriveCandidates(
	bundle replayableBehaviorR7AEntryBundle,
	specs []replayableBehaviorR7ASpec,
) ([]replayableBehaviorR7ACandidate, error) {
	r := bundle.Readiness
	s := bundle.Semantic

	if r.EvidenceNature != "first-web-target-gate-c-readiness-audit" {
		return nil, fmt.Errorf("unexpected Gate C evidence nature %q", r.EvidenceNature)
	}
	if !r.GateCReady {
		return nil, fmt.Errorf("Gate C is not ready")
	}
	for _, check := range r.Checks {
		if !check.Passed {
			return nil, fmt.Errorf("Gate C readiness check %q is not passed", check.Name)
		}
	}
	for _, unresolved := range r.Unresolved {
		if unresolved.Blocking {
			return nil, fmt.Errorf("blocking unresolved %q remains", unresolved.ID)
		}
	}

	if s.EvidenceNature != "first-web-target-semantic-functional-evidence" {
		return nil, fmt.Errorf("unexpected Semantic / Functional evidence nature %q", s.EvidenceNature)
	}
	if !s.SourceQualificationPassed ||
		!s.SemanticConformancePassed ||
		!s.FunctionalEvidencePassed {
		return nil, fmt.Errorf("Semantic / Functional Evidence is not fully passed")
	}
	if r.SemanticFunctionalRun != s.RunIdentity {
		return nil, fmt.Errorf("Semantic / Functional run identity mismatch")
	}
	if r.HandoverBaselineID != s.HandoverBaselineID ||
		r.BaselineRepositoryRevision != s.BaselineRepositoryRevision ||
		r.TargetBuildRevision != s.TargetBuildRevision ||
		r.TargetAdapter != s.TargetAdapter ||
		r.Host != s.Host {
		return nil, fmt.Errorf("R7A input does not share exact B/R/T/Target provenance")
	}

	digests := map[string]string{}
	for _, artifact := range s.Artifacts {
		if artifact.File == "" || artifact.SHA256 == "" {
			continue
		}
		if _, exists := digests[artifact.File]; exists {
			return nil, fmt.Errorf("duplicate raw artifact digest for %q", artifact.File)
		}
		digests[artifact.File] = artifact.SHA256
	}

	seen := map[string]struct{}{}
	var out []replayableBehaviorR7ACandidate
	for _, spec := range specs {
		if spec.BehaviorID == "" ||
			spec.SemanticOperation == "" ||
			spec.TargetRealizationRef == "" ||
			spec.RawArtifactFile == "" ||
			spec.ObservationClass == "" {
			return nil, fmt.Errorf("incomplete replay behavior spec: %+v", spec)
		}
		if _, exists := seen[spec.BehaviorID]; exists {
			return nil, fmt.Errorf("duplicate behavior identity %q", spec.BehaviorID)
		}
		seen[spec.BehaviorID] = struct{}{}

		digest := digests[spec.RawArtifactFile]
		if digest == "" {
			return nil, fmt.Errorf("raw artifact %q has no exact digest", spec.RawArtifactFile)
		}

		out = append(out, replayableBehaviorR7ACandidate{
			BehaviorID:                 spec.BehaviorID,
			SemanticOperation:          spec.SemanticOperation,
			TargetRealizationRef:       spec.TargetRealizationRef,
			RawArtifactFile:            spec.RawArtifactFile,
			RawArtifactSHA256:          digest,
			ObservationClass:           spec.ObservationClass,
			ReplayStatus:               "not-executed",
			CoverageClass:              "behavior-unit",
			HandoverBaselineID:         r.HandoverBaselineID,
			BaselineRepositoryRevision: r.BaselineRepositoryRevision,
			TargetBuildRevision:        r.TargetBuildRevision,
			SemanticFunctionalRun:      r.SemanticFunctionalRun,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].BehaviorID < out[j].BehaviorID
	})
	return out, nil
}

func replayableBehaviorR7AFirstWebSpecs() []replayableBehaviorR7ASpec {
	return []replayableBehaviorR7ASpec{
		{
			BehaviorID:           "first-web/read-only",
			SemanticOperation:    "order-entry.read-only-web-slice",
			TargetRealizationRef: "first-web-target:read",
			RawArtifactFile:      "read.json",
			ObservationClass:     "read-result",
		},
		{
			BehaviorID:           "first-web/command-success",
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:submit-success",
			RawArtifactFile:      "submit-success.json",
			ObservationClass:     "business-result-plus-separate-observation",
		},
		{
			BehaviorID:           "first-web/authentication-denied",
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:authentication-denied",
			RawArtifactFile:      "submit-authentication-denied.json",
			ObservationClass:     "business-denial-authentication",
		},
		{
			BehaviorID:           "first-web/authorization-denied",
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:authorization-denied",
			RawArtifactFile:      "submit-authorization-denied.json",
			ObservationClass:     "business-denial-authorization",
		},
		{
			BehaviorID:           "first-web/validation-not-satisfied",
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:validation-not-satisfied",
			RawArtifactFile:      "submit-validation-not-satisfied.json",
			ObservationClass:     "validation-not-satisfied",
		},
		{
			BehaviorID:           "first-web/execution-error",
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:execution-error",
			RawArtifactFile:      "submit-execution-error.json",
			ObservationClass:     "command-invocation-execution-error",
		},
	}
}

func replayableBehaviorR7ATestBundle() replayableBehaviorR7AEntryBundle {
	const (
		baseline = "gate-b-to-c-handover-baseline-b"
		r        = "test-only-R"
		t        = "test-only-T"
		run      = "test-only-semantic-functional-run"
	)

	files := []string{
		"read.json",
		"submit-success.json",
		"submit-authentication-denied.json",
		"submit-authorization-denied.json",
		"submit-validation-not-satisfied.json",
		"submit-execution-error.json",
	}
	var artifacts []replayableBehaviorR7AArtifactDigest
	for _, file := range files {
		artifacts = append(artifacts, replayableBehaviorR7AArtifactDigest{
			File:   file,
			SHA256: "test-only-sha256-" + file,
		})
	}

	checks := []replayableBehaviorR7ACheck{
		{Name: "same-B-R-T-and-target-identity", Passed: true},
		{Name: "source-semantic-functional-passed", Passed: true},
		{Name: "visual-fidelity-separately-passed", Passed: true},
		{Name: "canonical-world-invariant-R-to-T", Passed: true},
		{Name: "result-execution-error-observation-boundaries-preserved", Passed: true},
		{Name: "no-blocking-unresolved", Passed: true},
	}

	return replayableBehaviorR7AEntryBundle{
		Readiness: replayableBehaviorR7AGateCReadiness{
			EvidenceNature:             "first-web-target-gate-c-readiness-audit",
			HandoverBaselineID:         baseline,
			BaselineRepositoryRevision: r,
			TargetBuildRevision:        t,
			TargetAdapter:              "reference-first-web-target",
			Host:                       "go-net-http",
			SemanticFunctionalRun:      run,
			GateCReady:                 true,
			Checks:                     checks,
		},
		Semantic: replayableBehaviorR7ASemanticFunctionalSummary{
			EvidenceNature:             "first-web-target-semantic-functional-evidence",
			RunIdentity:                run,
			HandoverBaselineID:         baseline,
			BaselineRepositoryRevision: r,
			TargetBuildRevision:        t,
			TargetAdapter:              "reference-first-web-target",
			Host:                       "go-net-http",
			SourceQualificationPassed:  true,
			SemanticConformancePassed:  true,
			FunctionalEvidencePassed:   true,
			Artifacts:                  artifacts,
		},
	}
}
