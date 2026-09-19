package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestBuildReplayPlanBindsIntakeReceiptAndDerivesSixUnexecutedUnits(t *testing.T) {
	intake, receipt, intakeSHA, receiptSHA := testInputs(t)

	got, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA)
	if err != nil {
		t.Fatal(err)
	}
	if got.EvidenceNature != "first-web-target-r7b-replay-plan" {
		t.Fatalf("evidenceNature=%q", got.EvidenceNature)
	}
	if got.SourceExecutionIntakeSHA256 != intakeSHA ||
		got.SourceExecutionIntakeReceiptSHA256 != receiptSHA ||
		got.ReplayTargetBuildRevision != expectedTargetRevision {
		t.Fatalf("replay plan lost intake / receipt / Target identity: %+v", got)
	}
	if len(got.Units) != 6 {
		t.Fatalf("unit count=%d want=6", len(got.Units))
	}

	expected := expectedBehaviors()
	for _, unit := range got.Units {
		spec := expected[unit.BehaviorID]
		if unit.RequestFixtureID != spec.RequestFixtureID ||
			unit.TargetLocalInteraction != spec.TargetLocalAction ||
			unit.ExpectedHTTPStatus != spec.ExpectedHTTPStatus {
			t.Fatalf("Target-local recipe drifted for %q: %+v", unit.BehaviorID, unit)
		}
		if unit.ReplayTargetBuildRevision != unit.OriginTargetBuildRevision ||
			unit.RequiredDistinctRuns != 2 ||
			unit.ReplayExecutionStatus != "not-executed" ||
			unit.RepeatabilityStatus != "not-observed" ||
			unit.CoverageClass != "behavior-unit" {
			t.Fatalf("replay plan fabricated execution or changed replay Target: %+v", unit)
		}
	}
}

func TestBuildReplayPlanRejectsReceiptNotBoundToExactIntakeBytes(t *testing.T) {
	intake, receipt, intakeSHA, receiptSHA := testInputs(t)
	receipt.ExecutionIntakeArtifactSHA256 = strings.Repeat("f", 64)
	if receipt.ExecutionIntakeArtifactSHA256 == intakeSHA {
		t.Fatal("test digest unexpectedly equals intake digest")
	}
	if _, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA); err == nil {
		t.Fatal("accepted intake receipt not bound to exact intake bytes")
	}
}

func TestBuildReplayPlanRejectsReceiptThatClaimsReplayAlreadyOccurred(t *testing.T) {
	intake, receipt, intakeSHA, receiptSHA := testInputs(t)
	receipt.ReplayExecuted = true
	if _, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA); err == nil {
		t.Fatal("accepted intake receipt that already claimed replay execution")
	}
}

func TestBuildReplayPlanRejectsGateCOrExecutionStateDrift(t *testing.T) {
	t.Run("target-revision", func(t *testing.T) {
		intake, receipt, intakeSHA, receiptSHA := testInputs(t)
		intake.TargetBuildRevision = "different-T"
		if _, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA); err == nil {
			t.Fatal("accepted intake with Target revision drift")
		}
	})

	t.Run("already-executed", func(t *testing.T) {
		intake, receipt, intakeSHA, receiptSHA := testInputs(t)
		intake.Candidates[0].ReplayExecutionStatus = "passed"
		if _, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA); err == nil {
			t.Fatal("accepted intake that already claimed replay execution")
		}
	})

	t.Run("invalid-origin-digest", func(t *testing.T) {
		intake, receipt, intakeSHA, receiptSHA := testInputs(t)
		intake.Candidates[0].RawArtifactSHA256 = "not-sha256"
		if _, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA); err == nil {
			t.Fatal("accepted origin raw artifact without exact SHA-256 identity")
		}
	})
}

func TestBuildReplayPlanRejectsMissingDuplicateOrUnknownBehavior(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		intake, receipt, intakeSHA, receiptSHA := testInputs(t)
		intake.Candidates = intake.Candidates[1:]
		if _, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA); err == nil {
			t.Fatal("accepted incomplete behavior set")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		intake, receipt, intakeSHA, receiptSHA := testInputs(t)
		intake.Candidates = append(intake.Candidates, intake.Candidates[0])
		if _, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA); err == nil {
			t.Fatal("accepted duplicate behavior")
		}
	})

	t.Run("unknown", func(t *testing.T) {
		intake, receipt, intakeSHA, receiptSHA := testInputs(t)
		intake.Candidates[0].BehaviorID = "first-web/unknown"
		if _, err := buildReplayPlan(intake, receipt, intakeSHA, receiptSHA); err == nil {
			t.Fatal("accepted unknown behavior")
		}
	})
}

func TestReplayPlanDoesNotCarryCredentialMotorOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(replayUnit{})
	for _, forbidden := range []string{
		"LoginID",
		"Credential",
		"Password",
		"Selector",
		"DOMSelector",
		"Coordinate",
		"ClickSequence",
		"CanonicalRoute",
		"Authorized",
		"CanExecute",
		"ReplayPassed",
		"Repeatable",
		"NextAction",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("replay plan leaked forbidden field %q", forbidden)
		}
	}
}

func testInputs(t *testing.T) (
	executionIntake,
	intakeAcquisitionReceipt,
	string,
	string,
) {
	t.Helper()

	expected := expectedBehaviors()
	ids := make([]string, 0, len(expected))
	for id := range expected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	candidates := make([]executionIntakeCandidate, 0, len(ids))
	for _, id := range ids {
		spec := expected[id]
		sum := sha256.Sum256([]byte("origin-" + id))
		candidates = append(candidates, executionIntakeCandidate{
			BehaviorID:                      id,
			SemanticOperation:               spec.SemanticOperation,
			TargetRealizationRef:            spec.TargetRealizationRef,
			RawArtifactFile:                 spec.RawArtifactFile,
			RawArtifactSHA256:               hex.EncodeToString(sum[:]),
			ObservationClass:                spec.ObservationClass,
			ReplayExecutionStatus:           "not-executed",
			RepeatabilityStatus:             "not-observed",
			HandoffArtifactSHA256:           strings.Repeat("1", 64),
			GateCReadinessArtifactSHA256:    expectedReadinessSHA256,
			SemanticFunctionalSummarySHA256: expectedSemanticSHA256,
			HandoverBaselineID:              "B",
			BaselineRepositoryRevision:      expectedBaselineRevision,
			TargetBuildRevision:             expectedTargetRevision,
			SemanticFunctionalRun:           expectedSemanticRun,
			VisualRun:                       expectedVisualRun,
		})
	}

	intake := executionIntake{
		EvidenceNature:                        "first-web-target-r7b-execution-intake",
		SourceHandoffArtifact:                 "r7a-replayable-behavior-handoff.json",
		SourceHandoffArtifactSHA256:           strings.Repeat("1", 64),
		AcquisitionReceiptSHA256:              strings.Repeat("2", 64),
		HandoffToolRepositoryRevision:         strings.Repeat("3", 40),
		GateCJudgmentRecordRepositoryRevision: expectedGateCJudgmentRevision,
		HandoverBaselineID:                    "B",
		BaselineRepositoryRevision:            expectedBaselineRevision,
		TargetBuildRevision:                   expectedTargetRevision,
		TargetAdapter:                         "reference-first-web-target",
		Host:                                  "go-net-http",
		SemanticFunctionalRun:                 expectedSemanticRun,
		VisualRun:                             expectedVisualRun,
		GateCReadinessArtifactSHA256:          expectedReadinessSHA256,
		SemanticFunctionalSummarySHA256:       expectedSemanticSHA256,
		Candidates:                            candidates,
	}
	intakeBytes, err := json.MarshalIndent(intake, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	intakeBytes = append(intakeBytes, '\n')
	intakeSum := sha256.Sum256(intakeBytes)
	intakeSHA := hex.EncodeToString(intakeSum[:])

	receipt := intakeAcquisitionReceipt{
		EvidenceNature:                    "first-web-target-r7b-intake-acquisition-receipt",
		IntakeToolRepositoryRevision:      strings.Repeat("4", 40),
		RepositoryPostRunRevision:         strings.Repeat("4", 40),
		RepositoryPostRunClean:            true,
		SourceQualificationPassed:         true,
		SourceR7AHandoffSHA256:            intake.SourceHandoffArtifactSHA256,
		SourceR7AAcquisitionReceiptSHA256: intake.AcquisitionReceiptSHA256,
		ExecutionIntakeArtifact:           "r7b-execution-intake.json",
		ExecutionIntakeArtifactSHA256:     intakeSHA,
		ReplayExecuted:                    false,
		RepeatabilityObserved:             false,
	}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receiptBytes = append(receiptBytes, '\n')
	receiptSum := sha256.Sum256(receiptBytes)
	receiptSHA := hex.EncodeToString(receiptSum[:])

	return intake, receipt, intakeSHA, receiptSHA
}
