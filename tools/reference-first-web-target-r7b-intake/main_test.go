package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestBuildExecutionIntakePreservesActualEvidenceWithoutExecutingReplay(t *testing.T) {
	handoff, receipt, handoffDigest, receiptDigest := testInputs(t)
	got, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest)
	if err != nil {
		t.Fatal(err)
	}
	if got.EvidenceNature != "first-web-target-r7b-execution-intake" {
		t.Fatalf("evidenceNature=%q", got.EvidenceNature)
	}
	if got.SourceHandoffArtifactSHA256 != handoffDigest ||
		got.AcquisitionReceiptSHA256 != receiptDigest ||
		got.HandoffToolRepositoryRevision != receipt.HandoffToolRepositoryRevision {
		t.Fatalf("execution intake lost source Evidence identity: %+v", got)
	}
	if len(got.Candidates) != 6 {
		t.Fatalf("candidate count=%d, want 6", len(got.Candidates))
	}
	for _, candidate := range got.Candidates {
		if candidate.ReplayExecutionStatus != "not-executed" {
			t.Fatalf("intake fabricated replay execution: %+v", candidate)
		}
		if candidate.RepeatabilityStatus != "not-observed" {
			t.Fatalf("intake fabricated repeatability Evidence: %+v", candidate)
		}
		if candidate.HandoffArtifactSHA256 != handoffDigest {
			t.Fatalf("candidate lost handoff identity: %+v", candidate)
		}
	}
}

func TestBuildExecutionIntakeRequiresReceiptBoundToExactHandoffBytes(t *testing.T) {
	handoff, receipt, handoffDigest, receiptDigest := testInputs(t)
	receipt.HandoffArtifactSHA256 = strings.Repeat("f", 64)
	if _, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest); err == nil {
		t.Fatal("accepted receipt whose handoff digest did not match exact handoff bytes")
	}
}

func TestBuildExecutionIntakeRequiresCleanQualifiedAcquisition(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*acquisitionReceipt)
	}{
		{
			name: "repository-not-clean",
			mutate: func(receipt *acquisitionReceipt) {
				receipt.RepositoryPostRunClean = false
			},
		},
		{
			name: "source-qualification-not-passed",
			mutate: func(receipt *acquisitionReceipt) {
				receipt.SourceQualificationPassed = false
			},
		},
		{
			name: "tool-revision-changed-during-run",
			mutate: func(receipt *acquisitionReceipt) {
				receipt.RepositoryPostRunRevision = strings.Repeat("9", 40)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handoff, receipt, handoffDigest, receiptDigest := testInputs(t)
			tc.mutate(&receipt)
			if _, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest); err == nil {
				t.Fatalf("accepted invalid acquisition receipt: %s", tc.name)
			}
		})
	}
}

func TestBuildExecutionIntakeRequiresExactActualGateCProvenance(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*handoffSummary)
	}{
		{
			name: "target-revision",
			mutate: func(handoff *handoffSummary) {
				handoff.TargetBuildRevision = "different-T"
			},
		},
		{
			name: "visual-run",
			mutate: func(handoff *handoffSummary) {
				handoff.VisualRun = "different-visual-run"
			},
		},
		{
			name: "semantic-summary-digest",
			mutate: func(handoff *handoffSummary) {
				handoff.SemanticFunctionalSummarySHA256 = strings.Repeat("a", 64)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handoff, receipt, handoffDigest, receiptDigest := testInputs(t)
			tc.mutate(&handoff)
			if _, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest); err == nil {
				t.Fatalf("accepted mismatched actual Gate C provenance: %s", tc.name)
			}
		})
	}
}

func TestBuildExecutionIntakeRequiresCompleteUnexecutedSixBehaviorSet(t *testing.T) {
	t.Run("missing-behavior", func(t *testing.T) {
		handoff, receipt, handoffDigest, receiptDigest := testInputs(t)
		handoff.Candidates = handoff.Candidates[1:]
		if _, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest); err == nil {
			t.Fatal("accepted incomplete behavior set")
		}
	})

	t.Run("duplicate-behavior", func(t *testing.T) {
		handoff, receipt, handoffDigest, receiptDigest := testInputs(t)
		handoff.Candidates = append(handoff.Candidates, handoff.Candidates[0])
		if _, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest); err == nil {
			t.Fatal("accepted duplicate behavior")
		}
	})

	t.Run("already-executed", func(t *testing.T) {
		handoff, receipt, handoffDigest, receiptDigest := testInputs(t)
		handoff.Candidates[0].ReplayStatus = "passed"
		if _, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest); err == nil {
			t.Fatal("accepted handoff that claimed replay was already executed")
		}
	})

	t.Run("invalid-raw-artifact-digest", func(t *testing.T) {
		handoff, receipt, handoffDigest, receiptDigest := testInputs(t)
		handoff.Candidates[0].RawArtifactSHA256 = "not-sha256"
		if _, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest); err == nil {
			t.Fatal("accepted behavior without exact raw artifact SHA-256")
		}
	})
}

func TestReadJSONWithSHA256PinsExactInputBytes(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")
	if err := os.WriteFile(first, []byte("{\"evidenceNature\":\"x\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("{ \"evidenceNature\": \"x\" }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var a, b acquisitionReceipt
	firstDigest, err := readJSONWithSHA256(first, &a)
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := readJSONWithSHA256(second, &b)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest == secondDigest {
		t.Fatal("different exact bytes produced the same digest")
	}
	if !isSHA256(firstDigest) || !isSHA256(secondDigest) {
		t.Fatal("computed identity is not SHA-256")
	}
}

func TestExecutionIntakeShapeDoesNotClaimReplaySuccessOrCanonicalMotorIdentity(t *testing.T) {
	typ := reflect.TypeOf(executionIntakeSummary{})
	for _, forbidden := range []string{
		"ReplayPassed",
		"WorkflowPassed",
		"FunctionalPassed",
		"GateCReady",
		"Selector",
		"DOMSelector",
		"Coordinate",
		"ClickSequence",
		"CanonicalRoute",
		"CanonicalComponent",
		"NextAction",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("execution intake introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func testInputs(t *testing.T) (handoffSummary, acquisitionReceipt, string, string) {
	t.Helper()
	expected := expectedBehaviors()
	ids := make([]string, 0, len(expected))
	for id := range expected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	candidates := make([]handoffCandidate, 0, len(ids))
	for index, id := range ids {
		spec := expected[id]
		digit := string(rune('a' + index))
		candidates = append(candidates, handoffCandidate{
			BehaviorID:                      id,
			SemanticOperation:               spec.SemanticOperation,
			TargetRealizationRef:            spec.TargetRealizationRef,
			RawArtifactFile:                 spec.RawArtifactFile,
			RawArtifactSHA256:               strings.Repeat(digit, 64),
			ObservationClass:                spec.ObservationClass,
			ReplayStatus:                    "not-executed",
			CoverageClass:                   "behavior-unit",
			HandoverBaselineID:              "B",
			BaselineRepositoryRevision:      expectedBaselineRevision,
			TargetBuildRevision:             expectedTargetRevision,
			SemanticFunctionalRun:           expectedSemanticRun,
			VisualRun:                       expectedVisualRun,
			GateCReadinessArtifactSHA256:    expectedReadinessSHA256,
			SemanticFunctionalSummarySHA256: expectedSemanticSHA256,
		})
	}
	handoff := handoffSummary{
		EvidenceNature:                  "first-web-target-r7a-replayable-behavior-handoff",
		HandoverBaselineID:              "B",
		BaselineRepositoryRevision:      expectedBaselineRevision,
		TargetBuildRevision:             expectedTargetRevision,
		TargetAdapter:                   "reference-first-web-target",
		Host:                            "go-net-http",
		SemanticFunctionalRun:           expectedSemanticRun,
		VisualRun:                       expectedVisualRun,
		GateCReadinessArtifactSHA256:    expectedReadinessSHA256,
		SemanticFunctionalSummarySHA256: expectedSemanticSHA256,
		Candidates:                      candidates,
	}

	handoffBytes, err := json.MarshalIndent(handoff, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	handoffBytes = append(handoffBytes, '\n')
	handoffSum := sha256.Sum256(handoffBytes)
	handoffDigest := hex.EncodeToString(handoffSum[:])

	receipt := acquisitionReceipt{
		EvidenceNature:                        "first-web-target-r7a-handoff-acquisition-receipt",
		GateCJudgmentRecordRepositoryRevision: expectedGateCJudgmentRevision,
		RecordedTargetBuildRevision:           expectedTargetRevision,
		HandoffToolRepositoryRevision:         strings.Repeat("1", 40),
		RepositoryPostRunRevision:             strings.Repeat("1", 40),
		RepositoryPostRunClean:                true,
		SourceQualificationPassed:             true,
		GateCReadinessArtifactSHA256:          expectedReadinessSHA256,
		SemanticFunctionalSummarySHA256:       expectedSemanticSHA256,
		HandoffArtifact:                       "r7a-replayable-behavior-handoff.json",
		HandoffArtifactSHA256:                 handoffDigest,
	}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receiptBytes = append(receiptBytes, '\n')
	receiptSum := sha256.Sum256(receiptBytes)
	receiptDigest := hex.EncodeToString(receiptSum[:])
	return handoff, receipt, handoffDigest, receiptDigest
}
