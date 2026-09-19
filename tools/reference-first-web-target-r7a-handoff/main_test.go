package main

import (
	"os"
	"reflect"
	"testing"
)

func TestBuildHandoffPinsFinalReadinessProvenance(t *testing.T) {
	readiness, semantic := testInputs()

	got, err := buildHandoff(
		readiness,
		semantic,
		"sha256-readiness",
		"sha256-semantic",
		firstWebBehaviorSpecs(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.VisualRun != "visual-T" ||
		got.GateCReadinessArtifactSHA256 != "sha256-readiness" ||
		got.SemanticFunctionalSummarySHA256 != "sha256-semantic" {
		t.Fatalf("R7A handoff lost final readiness provenance: %+v", got)
	}
	if len(got.Candidates) != len(firstWebBehaviorSpecs()) {
		t.Fatalf(
			"candidate count=%d want=%d",
			len(got.Candidates),
			len(firstWebBehaviorSpecs()),
		)
	}
	for _, candidate := range got.Candidates {
		if candidate.ReplayStatus != "not-executed" ||
			candidate.CoverageClass != "behavior-unit" {
			t.Fatalf("R7A handoff fabricated replay result: %+v", candidate)
		}
		if candidate.VisualRun != got.VisualRun ||
			candidate.GateCReadinessArtifactSHA256 != got.GateCReadinessArtifactSHA256 ||
			candidate.SemanticFunctionalSummarySHA256 != got.SemanticFunctionalSummarySHA256 {
			t.Fatalf("candidate lost handoff provenance: %+v", candidate)
		}
	}
}

func TestBuildHandoffRejectsIncompleteFinalReadinessCheckSet(t *testing.T) {
	readiness, semantic := testInputs()
	readiness.Checks = readiness.Checks[1:]

	if _, err := buildHandoff(
		readiness,
		semantic,
		"sha256-readiness",
		"sha256-semantic",
		firstWebBehaviorSpecs(),
	); err == nil {
		t.Fatal("accepted incomplete final Gate C readiness check set")
	}
}

func TestBuildHandoffRejectsMissingInputArtifactDigest(t *testing.T) {
	readiness, semantic := testInputs()

	if _, err := buildHandoff(
		readiness,
		semantic,
		"",
		"sha256-semantic",
		firstWebBehaviorSpecs(),
	); err == nil {
		t.Fatal("accepted missing final readiness exact-bytes digest")
	}
}

func TestReadJSONWithDigestUsesExactInputBytes(t *testing.T) {
	file, err := os.CreateTemp("", "r7a-handoff-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	raw := []byte("{\n  \"passed\": true\n}\n")
	if _, err := file.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	var got passState
	firstDigest, err := readJSONWithDigest(file.Name(), &got)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Passed || firstDigest == "" {
		t.Fatalf("failed to decode or fingerprint exact input bytes: %+v %q", got, firstDigest)
	}

	raw = []byte("{\"passed\":true}\n")
	if err := os.WriteFile(file.Name(), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var second passState
	secondDigest, err := readJSONWithDigest(file.Name(), &second)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest == secondDigest {
		t.Fatal("exact-bytes digest ignored byte-level representation change")
	}
}

func TestHandoffShapeHasNoReplaySuccessOrCanonicalMotorShortcut(t *testing.T) {
	typ := reflect.TypeOf(handoffSummary{})
	for _, forbidden := range []string{
		"ReplayPassed",
		"WorkflowPassed",
		"FunctionalPassed",
		"Selector",
		"DOMSelector",
		"Coordinate",
		"ClickSequence",
		"CanonicalRoute",
		"CanonicalComponent",
		"NextAction",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("handoff introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func testInputs() (gateCReadiness, semanticFunctionalSummary) {
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

	artifacts := make([]artifactDigest, 0, len(firstWebBehaviorSpecs()))
	for _, spec := range firstWebBehaviorSpecs() {
		artifacts = append(artifacts, artifactDigest{
			File:   spec.RawArtifactFile,
			SHA256: "sha256-" + spec.RawArtifactFile,
		})
	}

	readiness := gateCReadiness{
		EvidenceNature:             "first-web-target-gate-c-readiness-audit",
		HandoverBaselineID:         "gate-b-to-c-handover-baseline-b",
		BaselineRepositoryRevision: "test-R",
		TargetBuildRevision:        "test-T",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
		SemanticFunctionalRun:      "test-run",
		VisualRun:                  "visual-T",
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
		RunIdentity:                "test-run",
		HandoverBaselineID:         "gate-b-to-c-handover-baseline-b",
		BaselineRepositoryRevision: "test-R",
		TargetBuildRevision:        "test-T",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
		SourceQualification:        passState{Passed: true},
		SemanticConformance:        passState{Passed: true},
		FunctionalEvidence:         passState{Passed: true},
		Artifacts:                  artifacts,
		GateCReady:                 false,
	}
	return readiness, semantic
}
