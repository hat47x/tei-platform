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

type replayableBehaviorR7ATargetDifferenceHandoff struct {
	ApplicationID string
	TargetAdapter string
	Host          string
}

type replayableBehaviorR7AGateCReadiness struct {
	EvidenceNature             string
	HandoverBaselineID         string
	BaselineRepositoryRevision string
	TargetBuildRevision        string
	TargetAdapter              string
	Host                       string
	SemanticFunctionalRun      string
	VisualRun                  string
	GateCReady                 bool
	Checks                     []replayableBehaviorR7ACheck
	Unresolved                 []replayableBehaviorR7AUnresolved
	TargetDifferenceHandoff    replayableBehaviorR7ATargetDifferenceHandoff
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
	GateCReady                 bool
	Artifacts                  []replayableBehaviorR7AArtifactDigest
}

type replayableBehaviorR7AEntryEvidence struct {
	GateCReadinessArtifactSHA256    string
	SemanticFunctionalSummarySHA256 string
}

type replayableBehaviorR7AEntryBundle struct {
	Readiness replayableBehaviorR7AGateCReadiness
	Semantic  replayableBehaviorR7ASemanticFunctionalSummary
	Evidence  replayableBehaviorR7AEntryEvidence
}

type replayableBehaviorR7ASpec struct {
	BehaviorID           string
	SemanticOperation    string
	TargetRealizationRef string
	RawArtifactFile      string
	ObservationClass     string
}

type replayableBehaviorR7ACandidate struct {
	BehaviorID                      string
	SemanticOperation               string
	TargetRealizationRef            string
	RawArtifactFile                 string
	RawArtifactSHA256               string
	ObservationClass                string
	ReplayStatus                    string
	CoverageClass                   string
	HandoverBaselineID              string
	BaselineRepositoryRevision      string
	TargetBuildRevision             string
	SemanticFunctionalRun           string
	VisualRun                       string
	GateCReadinessArtifactSHA256    string
	SemanticFunctionalSummarySHA256 string
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

func TestReplayableBehaviorR7ARejectsIncompleteOperationalHandoffState(t *testing.T) {
	t.Run("no-readiness-checks", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Readiness.Checks = nil
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted Gate C readiness without explicit checks")
		}
	})

	t.Run("unnamed-readiness-check", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Readiness.Checks[0].Name = ""
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted unnamed Gate C readiness check")
		}
	})

	t.Run("semantic-summary-declares-gate-c-ready", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Semantic.GateCReady = true
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted Semantic / Functional summary that bypasses final Gate C readiness")
		}
	})

	t.Run("target-difference-handoff-mismatch", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Readiness.TargetDifferenceHandoff.ApplicationID = "different_application"
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted Gate C target-difference handoff for a different Application")
		}
	})

	t.Run("incomplete-unrelated-artifact-digest", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Semantic.Artifacts = append(bundle.Semantic.Artifacts, replayableBehaviorR7AArtifactDigest{
			File: "extra-same-run-artifact.json",
		})
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A silently ignored an incomplete same-run artifact digest")
		}
	})

	t.Run("empty-provenance", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Readiness.TargetBuildRevision = ""
		bundle.Semantic.TargetBuildRevision = ""
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted empty provenance merely because both inputs matched")
		}
	})
}

func TestReplayableBehaviorR7ARequiresCompleteFinalReadinessShapeAndArtifactIdentity(t *testing.T) {
	t.Run("missing-required-readiness-check", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Readiness.Checks = bundle.Readiness.Checks[1:]
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted incomplete final Gate C readiness check set")
		}
	})

	t.Run("duplicate-readiness-check", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Readiness.Checks = append(bundle.Readiness.Checks, bundle.Readiness.Checks[0])
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted duplicate final Gate C readiness check")
		}
	})

	t.Run("unknown-readiness-check", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Readiness.Checks[0].Name = "test-only-unknown-readiness-check"
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted unknown final Gate C readiness check")
		}
	})

	t.Run("missing-visual-run", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Readiness.VisualRun = ""
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted final Gate C readiness without Visual run provenance")
		}
	})

	t.Run("missing-readiness-artifact-digest", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Evidence.GateCReadinessArtifactSHA256 = ""
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted unpinned final Gate C readiness artifact")
		}
	})

	t.Run("missing-semantic-summary-digest", func(t *testing.T) {
		bundle := replayableBehaviorR7ATestBundle()
		bundle.Evidence.SemanticFunctionalSummarySHA256 = ""
		if _, err := replayableBehaviorR7ADeriveCandidates(bundle, replayableBehaviorR7AFirstWebSpecs()); err == nil {
			t.Fatal("R7A accepted unpinned Semantic / Functional summary artifact")
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

	specByBehavior := map[string]replayableBehaviorR7ASpec{}
	for _, spec := range specs {
		specByBehavior[spec.BehaviorID] = spec
	}
	for i := range got {
		spec, ok := specByBehavior[got[i].BehaviorID]
		if !ok {
			t.Fatalf("R7A candidate has no source spec: %+v", got[i])
		}
		if got[i].SemanticOperation != spec.SemanticOperation {
			t.Fatalf("R7A semantic operation drifted: got=%q want=%q", got[i].SemanticOperation, spec.SemanticOperation)
		}
		if got[i].TargetRealizationRef != spec.TargetRealizationRef {
			t.Fatalf("R7A target realization drifted: got=%q want=%q", got[i].TargetRealizationRef, spec.TargetRealizationRef)
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
		if got[i].VisualRun != bundle.Readiness.VisualRun ||
			got[i].GateCReadinessArtifactSHA256 != bundle.Evidence.GateCReadinessArtifactSHA256 ||
			got[i].SemanticFunctionalSummarySHA256 != bundle.Evidence.SemanticFunctionalSummarySHA256 {
			t.Fatalf("R7A candidate lost final readiness provenance: %+v", got[i])
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
	if r.HandoverBaselineID == "" ||
		r.BaselineRepositoryRevision == "" ||
		r.TargetBuildRevision == "" ||
		r.TargetAdapter == "" ||
		r.Host == "" ||
		r.SemanticFunctionalRun == "" ||
		r.VisualRun == "" {
		return nil, fmt.Errorf("Gate C readiness provenance is incomplete")
	}
	if bundle.Evidence.GateCReadinessArtifactSHA256 == "" ||
		bundle.Evidence.SemanticFunctionalSummarySHA256 == "" {
		return nil, fmt.Errorf("R7A entry artifact identity is incomplete")
	}
	expectedChecks := replayableBehaviorR7AExpectedGateCReadinessChecks()
	if len(r.Checks) != len(expectedChecks) {
		return nil, fmt.Errorf("Gate C readiness check count=%d, want %d", len(r.Checks), len(expectedChecks))
	}
	seenChecks := map[string]struct{}{}
	for _, check := range r.Checks {
		if check.Name == "" || !check.Passed {
			return nil, fmt.Errorf("Gate C readiness check is not passed: %+v", check)
		}
		if _, required := expectedChecks[check.Name]; !required {
			return nil, fmt.Errorf("unknown Gate C readiness check %q", check.Name)
		}
		if _, duplicate := seenChecks[check.Name]; duplicate {
			return nil, fmt.Errorf("duplicate Gate C readiness check %q", check.Name)
		}
		seenChecks[check.Name] = struct{}{}
	}
	if len(seenChecks) != len(expectedChecks) {
		return nil, fmt.Errorf("Gate C readiness check set is incomplete")
	}
	for _, unresolved := range r.Unresolved {
		if unresolved.ID == "" {
			return nil, fmt.Errorf("Gate C readiness contains unnamed unresolved item")
		}
		if unresolved.Blocking {
			return nil, fmt.Errorf("blocking unresolved %q remains", unresolved.ID)
		}
	}
	if r.TargetDifferenceHandoff.ApplicationID != "order_entry" ||
		r.TargetDifferenceHandoff.TargetAdapter != r.TargetAdapter ||
		r.TargetDifferenceHandoff.Host != r.Host {
		return nil, fmt.Errorf("Gate C target-difference handoff does not describe the ready First Web Target")
	}

	if s.EvidenceNature != "first-web-target-semantic-functional-evidence" {
		return nil, fmt.Errorf("unexpected Semantic / Functional evidence nature %q", s.EvidenceNature)
	}
	if !s.SourceQualificationPassed ||
		!s.SemanticConformancePassed ||
		!s.FunctionalEvidencePassed {
		return nil, fmt.Errorf("Semantic / Functional Evidence is not fully passed")
	}
	if s.GateCReady {
		return nil, fmt.Errorf("Semantic / Functional summary must remain pre-Visual and must not declare Gate C Ready")
	}
	if s.RunIdentity == "" ||
		s.HandoverBaselineID == "" ||
		s.BaselineRepositoryRevision == "" ||
		s.TargetBuildRevision == "" ||
		s.TargetAdapter == "" ||
		s.Host == "" {
		return nil, fmt.Errorf("Semantic / Functional provenance is incomplete")
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
			return nil, fmt.Errorf("Semantic / Functional raw artifact digest is incomplete: %+v", artifact)
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
			BehaviorID:                      spec.BehaviorID,
			SemanticOperation:               spec.SemanticOperation,
			TargetRealizationRef:            spec.TargetRealizationRef,
			RawArtifactFile:                 spec.RawArtifactFile,
			RawArtifactSHA256:               digest,
			ObservationClass:                spec.ObservationClass,
			ReplayStatus:                    "not-executed",
			CoverageClass:                   "behavior-unit",
			HandoverBaselineID:              r.HandoverBaselineID,
			BaselineRepositoryRevision:      r.BaselineRepositoryRevision,
			TargetBuildRevision:             r.TargetBuildRevision,
			SemanticFunctionalRun:           r.SemanticFunctionalRun,
			VisualRun:                       r.VisualRun,
			GateCReadinessArtifactSHA256:    bundle.Evidence.GateCReadinessArtifactSHA256,
			SemanticFunctionalSummarySHA256: bundle.Evidence.SemanticFunctionalSummarySHA256,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].BehaviorID < out[j].BehaviorID
	})
	return out, nil
}

func replayableBehaviorR7AExpectedGateCReadinessChecks() map[string]struct{} {
	return map[string]struct{}{
		"same-B-R-T-and-target-identity":                          {},
		"source-semantic-functional-passed":                       {},
		"visual-fidelity-separately-passed":                       {},
		"canonical-world-invariant-R-to-T":                        {},
		"web-target-authority-boundary-passed":                    {},
		"standard-custom-shared-command-boundary-passed":          {},
		"result-execution-error-observation-boundaries-preserved": {},
		"target-difference-handoff-recoverable":                   {},
		"all-unresolved-explicitly-classified":                    {},
		"no-blocking-unresolved":                                  {},
	}
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
		{Name: "web-target-authority-boundary-passed", Passed: true},
		{Name: "standard-custom-shared-command-boundary-passed", Passed: true},
		{Name: "result-execution-error-observation-boundaries-preserved", Passed: true},
		{Name: "target-difference-handoff-recoverable", Passed: true},
		{Name: "all-unresolved-explicitly-classified", Passed: true},
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
			VisualRun:                  "test-only-visual-run",
			GateCReady:                 true,
			Checks:                     checks,
			TargetDifferenceHandoff: replayableBehaviorR7ATargetDifferenceHandoff{
				ApplicationID: "order_entry",
				TargetAdapter: "reference-first-web-target",
				Host:          "go-net-http",
			},
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
			GateCReady:                 false,
			Artifacts:                  artifacts,
		},
		Evidence: replayableBehaviorR7AEntryEvidence{
			GateCReadinessArtifactSHA256:    "test-only-sha256-gate-c-readiness-audit",
			SemanticFunctionalSummarySHA256: "test-only-sha256-semantic-functional-summary",
		},
	}
}
