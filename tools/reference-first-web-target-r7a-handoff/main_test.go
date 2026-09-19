package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildHandoffPreservesReadyGateCProvenanceWithoutClaimingReplay(t *testing.T) {
	readiness, semantic, evidence := testInputs()

	got, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs())
	if err != nil {
		t.Fatal(err)
	}
	if got.EvidenceNature != "first-web-target-r7a-replayable-behavior-handoff" {
		t.Fatalf("evidenceNature=%q", got.EvidenceNature)
	}
	if got.VisualRun != readiness.VisualRun ||
		got.GateCReadinessArtifactSHA256 != evidence.GateCReadinessArtifactSHA256 ||
		got.SemanticFunctionalSummarySHA256 != evidence.SemanticFunctionalSummarySHA256 {
		t.Fatalf("handoff lost final Evidence identity: %+v", got)
	}
	if len(got.Candidates) != len(firstWebBehaviorSpecs()) {
		t.Fatalf("candidate count=%d", len(got.Candidates))
	}
	for _, candidate := range got.Candidates {
		if candidate.ReplayStatus != "not-executed" {
			t.Fatalf("handoff fabricated replay execution: %+v", candidate)
		}
		if candidate.CoverageClass != "behavior-unit" {
			t.Fatalf("handoff fabricated broader coverage: %+v", candidate)
		}
		if candidate.VisualRun != readiness.VisualRun ||
			candidate.GateCReadinessArtifactSHA256 != evidence.GateCReadinessArtifactSHA256 ||
			candidate.SemanticFunctionalSummarySHA256 != evidence.SemanticFunctionalSummarySHA256 {
			t.Fatalf("candidate lost final Evidence identity: %+v", candidate)
		}
		if !isSHA256(candidate.RawArtifactSHA256) {
			t.Fatalf("candidate raw artifact digest=%q", candidate.RawArtifactSHA256)
		}
	}
}

func TestBuildHandoffRequiresCompleteFinalReadinessShape(t *testing.T) {
	t.Run("gate-c-not-ready", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		readiness.GateCReady = false
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted Gate C not-ready input")
		}
	})

	t.Run("missing-required-check", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		readiness.Checks = readiness.Checks[1:]
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted incomplete readiness check set")
		}
	})

	t.Run("unknown-check", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		readiness.Checks[0].Name = "test-only-unknown"
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted unknown readiness check")
		}
	})

	t.Run("duplicate-check", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		readiness.Checks[len(readiness.Checks)-1] = readiness.Checks[0]
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted duplicate readiness check")
		}
	})

	t.Run("missing-visual-run", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		readiness.VisualRun = ""
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted readiness without Visual run")
		}
	})

	t.Run("blocking-unresolved", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		readiness.Unresolved = append(readiness.Unresolved, unresolvedDisposition{
			ID:       "test-only-blocker",
			Blocking: true,
		})
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted blocking unresolved")
		}
	})
}

func TestBuildHandoffRequiresExactEvidenceAndRawArtifactIdentity(t *testing.T) {
	t.Run("semantic-run-mismatch", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		semantic.RunIdentity = "different-run"
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted Semantic / Functional run mismatch")
		}
	})

	t.Run("target-revision-mismatch", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		semantic.TargetBuildRevision = "different-T"
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted Target revision mismatch")
		}
	})

	t.Run("missing-readiness-artifact-digest", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		evidence.GateCReadinessArtifactSHA256 = ""
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted unpinned readiness artifact")
		}
	})

	t.Run("invalid-semantic-summary-digest", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		evidence.SemanticFunctionalSummarySHA256 = "not-a-sha256"
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted invalid Semantic / Functional summary digest")
		}
	})

	t.Run("missing-raw-artifact-digest", func(t *testing.T) {
		readiness, semantic, evidence := testInputs()
		for i := range semantic.Artifacts {
			if semantic.Artifacts[i].File == "submit-success.json" {
				semantic.Artifacts[i].SHA256 = ""
			}
		}
		if _, err := buildHandoff(readiness, semantic, evidence, firstWebBehaviorSpecs()); err == nil {
			t.Fatal("handoff accepted behavior without exact raw artifact digest")
		}
	})
}

func TestReadJSONWithSHA256PinsExactInputBytes(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")
	if err := os.WriteFile(first, []byte("{\"passed\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("{ \"passed\": true }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var firstValue passState
	firstDigest, err := readJSONWithSHA256(first, &firstValue)
	if err != nil {
		t.Fatal(err)
	}
	var secondValue passState
	secondDigest, err := readJSONWithSHA256(second, &secondValue)
	if err != nil {
		t.Fatal(err)
	}
	if !firstValue.Passed || !secondValue.Passed {
		t.Fatal("JSON values did not decode")
	}
	if firstDigest == secondDigest {
		t.Fatal("different exact JSON bytes produced the same handoff identity")
	}
	if !isSHA256(firstDigest) || !isSHA256(secondDigest) {
		t.Fatalf("computed digests are not SHA-256: %q %q", firstDigest, secondDigest)
	}
}

func TestReplayableBehaviorCandidateDoesNotLeakMotorOrReadinessAuthority(t *testing.T) {
	typ := reflect.TypeOf(replayableBehaviorCandidate{})
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
			t.Fatalf("candidate leaked forbidden field %q", forbidden)
		}
	}
}

func testInputs() (gateCReadiness, semanticFunctionalSummary, entryEvidence) {
	const (
		baseline = "gate-b-to-c-handover-baseline-b"
		r        = "test-only-R"
		tRev     = "test-only-T"
		run      = "test-only-semantic-functional-run"
		visual   = "test-only-visual-run"
	)

	checkNames := []string{
		"same-B-R-T-and-target-identity",
		"source-semantic-functional-passed",
		"visual-fidelity-separately-passed",
		"canonical-world-invariant-R-to-T",
		"web-target-authority-boundary-passed",
		"standard-custom-shared-command-boundary-passed",
		"result-execution-error-observation-boundaries-preserved",
		"target-difference-handoff-recoverable",
		"all-unresolved-explicitly-classified",
		"no-blocking-unresolved",
	}
	checks := make([]readinessCheck, 0, len(checkNames))
	for _, name := range checkNames {
		checks = append(checks, readinessCheck{Name: name, Passed: true})
	}

	files := []string{
		"read.json",
		"submit-success.json",
		"submit-authentication-denied.json",
		"submit-authorization-denied.json",
		"submit-validation-not-satisfied.json",
		"submit-execution-error.json",
	}
	artifacts := make([]artifactDigest, 0, len(files))
	for index, file := range files {
		digit := string(rune('a' + index))
		artifacts = append(artifacts, artifactDigest{
			File:   file,
			SHA256: strings.Repeat(digit, 64),
		})
	}

	readiness := gateCReadiness{
		EvidenceNature:             "first-web-target-gate-c-readiness-audit",
		HandoverBaselineID:         baseline,
		BaselineRepositoryRevision: r,
		TargetBuildRevision:        tRev,
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
		SemanticFunctionalRun:      run,
		VisualRun:                  visual,
		Checks:                     checks,
		TargetDifferenceHandoff: readinessTargetDifference{
			ApplicationID: "order_entry",
			TargetAdapter: "reference-first-web-target",
			Host:          "go-net-http",
		},
		GateCReady: true,
	}
	semantic := semanticFunctionalSummary{
		EvidenceNature:             "first-web-target-semantic-functional-evidence",
		RunIdentity:                run,
		HandoverBaselineID:         baseline,
		BaselineRepositoryRevision: r,
		TargetBuildRevision:        tRev,
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
		SourceQualification:        passState{Passed: true},
		SemanticConformance:        passState{Passed: true},
		FunctionalEvidence:         passState{Passed: true},
		Artifacts:                  artifacts,
		GateCReady:                 false,
	}
	evidence := entryEvidence{
		GateCReadinessArtifactSHA256:    strings.Repeat("1", 64),
		SemanticFunctionalSummarySHA256: strings.Repeat("2", 64),
	}
	return readiness, semantic, evidence
}
