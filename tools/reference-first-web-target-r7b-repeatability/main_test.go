package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestAggregateRepeatabilityRequiresTwoDistinctAuditedRuns(t *testing.T) {
	plan, _, planBytes, receiptBytes := testPlanReceipt(t)
	run1 := testRun(t, plan, planBytes, receiptBytes, "run-1", "process-1", "a")
	run2 := testRun(t, plan, planBytes, receiptBytes, "run-2", "process-2", "b")
	got, err := aggregateRepeatability(planBytes, receiptBytes, [][]byte{run1, run2})
	if err != nil {
		t.Fatal(err)
	}
	if !got.RepeatabilityObserved || got.RequiredDistinctRuns != 2 || got.ObservedDistinctRuns != 2 {
		t.Fatalf("summary=%+v", got)
	}
	if got.CoverageClass != "behavior-unit" || got.DeepWorkflowCovered {
		t.Fatalf("coverage inflated: %+v", got)
	}
	if len(got.Behaviors) != 6 || len(got.Runs) != 2 {
		t.Fatalf("summary counts=%+v", got)
	}
	for _, b := range got.Behaviors {
		if !b.Repeatable || b.DistinctRuns != 2 {
			t.Fatalf("behavior=%+v", b)
		}
	}
}

func TestAggregateRepeatabilityDoesNotRequireRawDigestEquality(t *testing.T) {
	plan, _, planBytes, receiptBytes := testPlanReceipt(t)
	run1 := testRun(t, plan, planBytes, receiptBytes, "run-1", "process-1", "first")
	run2 := testRun(t, plan, planBytes, receiptBytes, "run-2", "process-2", "second")
	var a, b runObservation
	if err := json.Unmarshal(run1, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(run2, &b); err != nil {
		t.Fatal(err)
	}
	if a.Observations[0].RawArtifactSHA256 == b.Observations[0].RawArtifactSHA256 {
		t.Fatal("test digests unexpectedly equal")
	}
	if _, err := aggregateRepeatability(planBytes, receiptBytes, [][]byte{run1, run2}); err != nil {
		t.Fatal(err)
	}
}

func TestAggregateRepeatabilityRejectsDuplicateRunOrProcessIdentity(t *testing.T) {
	plan, _, planBytes, receiptBytes := testPlanReceipt(t)
	cases := []struct{ name, run2, proc2 string }{
		{"run", "run-1", "process-2"},
		{"process", "run-2", "process-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run1 := testRun(t, plan, planBytes, receiptBytes, "run-1", "process-1", "a")
			run2 := testRun(t, plan, planBytes, receiptBytes, tc.run2, tc.proc2, "b")
			if _, err := aggregateRepeatability(planBytes, receiptBytes, [][]byte{run1, run2}); err == nil {
				t.Fatal("accepted duplicate identity")
			}
		})
	}
}

func TestAggregateRepeatabilityRejectsRunBoundaryDrift(t *testing.T) {
	plan, _, planBytes, receiptBytes := testPlanReceipt(t)
	cases := []struct {
		name   string
		mutate func(*runObservation)
	}{
		{"target", func(r *runObservation) { r.ReplayTargetBuildRevision = strings.Repeat("9", 40) }},
		{"listen", func(r *runObservation) { r.TargetListenScope = "0.0.0.0" }},
		{"status", func(r *runObservation) { r.Observations[0].ObservedHTTPStatus = 500 }},
		{"class", func(r *runObservation) { r.Observations[0].ObservationClass = "other" }},
		{"semantic-false", func(r *runObservation) { r.Observations[0].SemanticValidationPassed = false }},
		{"raw-sha", func(r *runObservation) { r.Observations[0].RawArtifactSHA256 = "bad" }},
		{"missing-behavior", func(r *runObservation) { r.Observations = r.Observations[1:] }},
		{"duplicate-behavior", func(r *runObservation) { r.Observations[1] = r.Observations[0] }},
		{"plan-binding", func(r *runObservation) { r.SourceReplayPlanSHA256 = strings.Repeat("f", 64) }},
		{"receipt-binding", func(r *runObservation) { r.SourceReplayPlanReceiptSHA256 = strings.Repeat("e", 64) }},
	}
	run1 := testRun(t, plan, planBytes, receiptBytes, "run-1", "process-1", "a")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run2 := testRun(t, plan, planBytes, receiptBytes, "run-2", "process-2", "b")
			var r runObservation
			if err := json.Unmarshal(run2, &r); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&r)
			run2 = marshal(t, r)
			if _, err := aggregateRepeatability(planBytes, receiptBytes, [][]byte{run1, run2}); err == nil {
				t.Fatalf("accepted %s drift", tc.name)
			}
		})
	}
}

func TestAggregateRepeatabilityRejectsPlanOrReceiptDrift(t *testing.T) {
	plan, receipt, planBytes, receiptBytes := testPlanReceipt(t)
	runs := [][]byte{testRun(t, plan, planBytes, receiptBytes, "run-1", "process-1", "a"), testRun(t, plan, planBytes, receiptBytes, "run-2", "process-2", "b")}
	t.Run("receipt-plan-sha", func(t *testing.T) {
		r := receipt
		r.ReplayPlanArtifactSHA256 = strings.Repeat("f", 64)
		if _, err := aggregateRepeatability(planBytes, marshal(t, r), runs); err == nil {
			t.Fatal("accepted unbound receipt")
		}
	})
	t.Run("receipt-already-observed", func(t *testing.T) {
		r := receipt
		r.RepeatabilityObserved = true
		if _, err := aggregateRepeatability(planBytes, marshal(t, r), runs); err == nil {
			t.Fatal("accepted prior repeatability claim")
		}
	})
	t.Run("plan-required-runs", func(t *testing.T) {
		p := plan
		p.Units = append([]replayUnit(nil), plan.Units...)
		for i := range p.Units {
			p.Units[i].RequiredDistinctRuns = 3
		}
		pBytes := marshal(t, p)
		r := receipt
		r.ReplayPlanArtifactSHA256 = sha256Hex(pBytes)
		rBytes := marshal(t, r)
		mutatedRuns := [][]byte{
			testRun(t, p, pBytes, rBytes, "run-1", "process-1", "a"),
			testRun(t, p, pBytes, rBytes, "run-2", "process-2", "b"),
			testRun(t, p, pBytes, rBytes, "run-3", "process-3", "c"),
		}
		if _, err := aggregateRepeatability(pBytes, rBytes, mutatedRuns); err == nil {
			t.Fatal("accepted RequiredDistinctRuns drift")
		}
	})
	t.Run("plan-http-status", func(t *testing.T) {
		p := plan
		p.Units = append([]replayUnit(nil), plan.Units...)
		p.Units[0].ExpectedHTTPStatus = 201
		pBytes := marshal(t, p)
		r := receipt
		r.ReplayPlanArtifactSHA256 = sha256Hex(pBytes)
		rBytes := marshal(t, r)
		mutatedRuns := [][]byte{
			testRun(t, p, pBytes, rBytes, "run-1", "process-1", "a"),
			testRun(t, p, pBytes, rBytes, "run-2", "process-2", "b"),
		}
		if _, err := aggregateRepeatability(pBytes, rBytes, mutatedRuns); err == nil {
			t.Fatal("accepted expected HTTP drift")
		}
	})
	t.Run("plan-observation-class", func(t *testing.T) {
		p := plan
		p.Units = append([]replayUnit(nil), plan.Units...)
		p.Units[0].ExpectedObservationClass = "other-class"
		pBytes := marshal(t, p)
		r := receipt
		r.ReplayPlanArtifactSHA256 = sha256Hex(pBytes)
		rBytes := marshal(t, r)
		mutatedRuns := [][]byte{
			testRun(t, p, pBytes, rBytes, "run-1", "process-1", "a"),
			testRun(t, p, pBytes, rBytes, "run-2", "process-2", "b"),
		}
		if _, err := aggregateRepeatability(pBytes, rBytes, mutatedRuns); err == nil {
			t.Fatal("accepted expected Observation class drift")
		}
	})
}

func TestAggregateRepeatabilityRejectsUnknownPlanBehavior(t *testing.T) {
	plan, receipt, _, _ := testPlanReceipt(t)
	plan.Units = append([]replayUnit(nil), plan.Units...)
	plan.Units[0].BehaviorID = "first-web/unknown"
	planBytes := marshal(t, plan)
	receipt.ReplayPlanArtifactSHA256 = sha256Hex(planBytes)
	receiptBytes := marshal(t, receipt)
	runs := [][]byte{
		testRun(t, plan, planBytes, receiptBytes, "run-1", "process-1", "a"),
		testRun(t, plan, planBytes, receiptBytes, "run-2", "process-2", "b"),
	}
	if _, err := aggregateRepeatability(planBytes, receiptBytes, runs); err == nil {
		t.Fatal("accepted unknown replay-plan behavior")
	}
}

func TestAggregateRepeatabilityNeedsAtLeastRequiredRuns(t *testing.T) {
	plan, _, planBytes, receiptBytes := testPlanReceipt(t)
	run := testRun(t, plan, planBytes, receiptBytes, "run-1", "process-1", "a")
	if _, err := aggregateRepeatability(planBytes, receiptBytes, [][]byte{run}); err == nil {
		t.Fatal("one run was accepted as repeatability")
	}
}

func testPlanReceipt(t *testing.T) (replayPlan, replayPlanReceipt, []byte, []byte) {
	t.Helper()
	ids := []string{"first-web/authentication-denied", "first-web/authorization-denied", "first-web/command-success", "first-web/execution-error", "first-web/read-only", "first-web/validation-not-satisfied"}
	classes := map[string]string{
		"first-web/authentication-denied":    "business-denial-authentication",
		"first-web/authorization-denied":     "business-denial-authorization",
		"first-web/command-success":          "business-result-plus-separate-observation",
		"first-web/execution-error":          "command-invocation-execution-error",
		"first-web/read-only":                "read-result",
		"first-web/validation-not-satisfied": "validation-not-satisfied",
	}
	statuses := map[string]int{"first-web/execution-error": 422}
	units := make([]replayUnit, 0, len(ids))
	target := strings.Repeat("a", 40)
	for _, id := range ids {
		status := statuses[id]
		if status == 0 {
			status = 200
		}
		units = append(units, replayUnit{BehaviorID: id, ExpectedHTTPStatus: status, ExpectedObservationClass: classes[id], ReplayTargetBuildRevision: target, RequiredDistinctRuns: 2, ReplayExecutionStatus: "not-executed", RepeatabilityStatus: "not-observed", CoverageClass: "behavior-unit"})
	}
	plan := replayPlan{EvidenceNature: "first-web-target-r7b-replay-plan", SourceExecutionIntakeSHA256: strings.Repeat("1", 64), SourceExecutionIntakeReceiptSHA256: strings.Repeat("2", 64), HandoverBaselineID: "B", BaselineRepositoryRevision: strings.Repeat("b", 40), ReplayTargetBuildRevision: target, TargetAdapter: "reference-first-web-target", Host: "go-net-http", Units: units}
	planBytes := marshal(t, plan)
	planSHA := sha256Hex(planBytes)
	receipt := replayPlanReceipt{EvidenceNature: "first-web-target-r7b-replay-plan-acquisition-receipt", ReplayPlanToolRepositoryRevision: strings.Repeat("c", 40), RepositoryPostRunRevision: strings.Repeat("c", 40), RepositoryPostRunClean: true, SourceQualificationPassed: true, SourceExecutionIntakeSHA256: plan.SourceExecutionIntakeSHA256, SourceExecutionIntakeReceiptSHA256: plan.SourceExecutionIntakeReceiptSHA256, ReplayPlanArtifact: "r7b-replay-plan.json", ReplayPlanArtifactSHA256: planSHA, ReplayExecuted: false, RepeatabilityObserved: false}
	return plan, receipt, planBytes, marshal(t, receipt)
}

func testRun(t *testing.T, plan replayPlan, planBytes, receiptBytes []byte, runID, processID, salt string) []byte {
	t.Helper()
	obs := make([]behaviorObservation, 0, len(plan.Units))
	for _, u := range plan.Units {
		sum := sha256.Sum256([]byte(salt + u.BehaviorID))
		obs = append(obs, behaviorObservation{BehaviorID: u.BehaviorID, ObservedHTTPStatus: u.ExpectedHTTPStatus, ObservationClass: u.ExpectedObservationClass, SemanticValidationPassed: true, RawArtifactFile: u.BehaviorID + ".json", RawArtifactSHA256: hex.EncodeToString(sum[:])})
	}
	r := runObservation{EvidenceNature: "first-web-target-r7b-run-observation", RunIdentity: runID, TargetProcessInstanceIdentity: processID, SourceReplayPlanSHA256: sha256Hex(planBytes), SourceReplayPlanReceiptSHA256: sha256Hex(receiptBytes), HandoverBaselineID: plan.HandoverBaselineID, BaselineRepositoryRevision: plan.BaselineRepositoryRevision, ReplayTargetBuildRevision: plan.ReplayTargetBuildRevision, TargetAdapter: plan.TargetAdapter, Host: plan.Host, TargetListenScope: "loopback", Observations: obs}
	return marshal(t, r)
}

func marshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}
