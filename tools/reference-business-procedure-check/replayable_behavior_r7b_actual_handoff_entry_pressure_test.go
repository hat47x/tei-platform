package main

import (
	"fmt"
	"sort"
	"testing"
)

type replayableBehaviorR7BRecordedGateC struct {
	HandoverBaselineID               string
	BaselineRepositoryRevision       string
	TargetBuildRevision              string
	TargetAdapter                    string
	Host                             string
	SemanticFunctionalRun            string
	SemanticFunctionalSummarySHA256  string
	GateCReadinessArtifactSHA256     string
	GateCJudgmentRecordRepositorySHA string
	GateCReady                       bool
}

type replayableBehaviorR7BHandoffCandidate struct {
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

type replayableBehaviorR7AHandoffArtifact struct {
	EvidenceNature                  string
	HandoverBaselineID              string
	BaselineRepositoryRevision      string
	TargetBuildRevision             string
	TargetAdapter                   string
	Host                            string
	SemanticFunctionalRun           string
	VisualRun                       string
	GateCReadinessArtifactSHA256    string
	SemanticFunctionalSummarySHA256 string
	Candidates                      []replayableBehaviorR7BHandoffCandidate
}

type replayableBehaviorR7BEntry struct {
	GateCRecord           replayableBehaviorR7BRecordedGateC
	Handoff               replayableBehaviorR7AHandoffArtifact
	HandoffArtifactSHA256 string
}

type replayableBehaviorR7BExecutionIntake struct {
	BehaviorID                      string
	SemanticOperation               string
	TargetRealizationRef            string
	RawArtifactFile                 string
	RawArtifactSHA256               string
	ObservationClass                string
	ReplayExecutionStatus           string
	RepeatabilityStatus             string
	HandoffArtifactSHA256           string
	GateCReadinessArtifactSHA256    string
	SemanticFunctionalSummarySHA256 string
	HandoverBaselineID              string
	BaselineRepositoryRevision      string
	TargetBuildRevision             string
	SemanticFunctionalRun           string
	VisualRun                       string
}

type replayableBehaviorR7BExpectedBehavior struct {
	SemanticOperation    string
	TargetRealizationRef string
	RawArtifactFile      string
	ObservationClass     string
}

func TestReplayableBehaviorR7BCurrentRecordedStateRemainsBlockedWithoutActualHandoffArtifact(t *testing.T) {
	entry := replayableBehaviorR7BEntry{
		GateCRecord: replayableBehaviorR7BActualGateCRecord(),
	}
	if _, err := replayableBehaviorR7BDeriveExecutionIntake(entry); err == nil {
		t.Fatal("R7B started without an actual R7A handoff artifact")
	}
}

func TestReplayableBehaviorR7BRequiresExactActualGateCProvenance(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*replayableBehaviorR7BEntry)
	}{
		{
			name: "target-build-revision",
			mutate: func(entry *replayableBehaviorR7BEntry) {
				entry.Handoff.TargetBuildRevision = "different-T"
			},
		},
		{
			name: "semantic-functional-summary-digest",
			mutate: func(entry *replayableBehaviorR7BEntry) {
				entry.Handoff.SemanticFunctionalSummarySHA256 = "different-summary-digest"
			},
		},
		{
			name: "gate-c-readiness-digest",
			mutate: func(entry *replayableBehaviorR7BEntry) {
				entry.Handoff.GateCReadinessArtifactSHA256 = "different-readiness-digest"
			},
		},
		{
			name: "visual-run",
			mutate: func(entry *replayableBehaviorR7BEntry) {
				entry.Handoff.VisualRun = "different-visual-run"
			},
		},
		{
			name: "empty-handoff-artifact-digest",
			mutate: func(entry *replayableBehaviorR7BEntry) {
				entry.HandoffArtifactSHA256 = ""
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := replayableBehaviorR7BTestEntry()
			tc.mutate(&entry)
			if _, err := replayableBehaviorR7BDeriveExecutionIntake(entry); err == nil {
				t.Fatalf("R7B accepted mismatched or unpinned actual handoff input: %s", tc.name)
			}
		})
	}
}

func TestReplayableBehaviorR7BRequiresCompleteSixBehaviorHandoff(t *testing.T) {
	t.Run("missing-behavior", func(t *testing.T) {
		entry := replayableBehaviorR7BTestEntry()
		entry.Handoff.Candidates = entry.Handoff.Candidates[1:]
		if _, err := replayableBehaviorR7BDeriveExecutionIntake(entry); err == nil {
			t.Fatal("R7B accepted incomplete First Web Target behavior handoff")
		}
	})

	t.Run("duplicate-behavior", func(t *testing.T) {
		entry := replayableBehaviorR7BTestEntry()
		entry.Handoff.Candidates = append(entry.Handoff.Candidates, entry.Handoff.Candidates[0])
		if _, err := replayableBehaviorR7BDeriveExecutionIntake(entry); err == nil {
			t.Fatal("R7B accepted duplicate behavior identity")
		}
	})

	t.Run("wrong-raw-artifact", func(t *testing.T) {
		entry := replayableBehaviorR7BTestEntry()
		entry.Handoff.Candidates[0].RawArtifactFile = "different.json"
		if _, err := replayableBehaviorR7BDeriveExecutionIntake(entry); err == nil {
			t.Fatal("R7B accepted behavior mapped to the wrong raw artifact")
		}
	})

	t.Run("already-executed-status", func(t *testing.T) {
		entry := replayableBehaviorR7BTestEntry()
		entry.Handoff.Candidates[0].ReplayStatus = "passed"
		if _, err := replayableBehaviorR7BDeriveExecutionIntake(entry); err == nil {
			t.Fatal("R7B accepted source handoff that fabricated prior replay execution")
		}
	})
}

func TestReplayableBehaviorR7BExecutionIntakePreservesEvidenceWithoutFabricatingExecution(t *testing.T) {
	entry := replayableBehaviorR7BTestEntry()
	got, err := replayableBehaviorR7BDeriveExecutionIntake(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("R7B execution intake count=%d, want 6", len(got))
	}
	for _, item := range got {
		if item.ReplayExecutionStatus != "not-executed" {
			t.Fatalf("R7B fabricated replay execution status: %+v", item)
		}
		if item.RepeatabilityStatus != "not-observed" {
			t.Fatalf("R7B fabricated repeatability result: %+v", item)
		}
		if item.HandoffArtifactSHA256 != entry.HandoffArtifactSHA256 ||
			item.GateCReadinessArtifactSHA256 != entry.GateCRecord.GateCReadinessArtifactSHA256 ||
			item.SemanticFunctionalSummarySHA256 != entry.GateCRecord.SemanticFunctionalSummarySHA256 {
			t.Fatalf("R7B lost actual Evidence provenance: %+v", item)
		}
	}
}

func replayableBehaviorR7BDeriveExecutionIntake(
	entry replayableBehaviorR7BEntry,
) ([]replayableBehaviorR7BExecutionIntake, error) {
	actual := replayableBehaviorR7BActualGateCRecord()
	gate := entry.GateCRecord
	handoff := entry.Handoff

	if gate != actual {
		return nil, fmt.Errorf("R7B Gate C record does not match the recorded actual Gate C determination")
	}
	if !gate.GateCReady {
		return nil, fmt.Errorf("actual Gate C record is not Ready")
	}
	if entry.HandoffArtifactSHA256 == "" {
		return nil, fmt.Errorf("actual R7A handoff artifact is not acquired or not fingerprinted")
	}
	if handoff.EvidenceNature != "first-web-target-r7a-replayable-behavior-handoff" {
		return nil, fmt.Errorf("unexpected R7A handoff evidence nature %q", handoff.EvidenceNature)
	}
	if handoff.HandoverBaselineID != gate.HandoverBaselineID ||
		handoff.BaselineRepositoryRevision != gate.BaselineRepositoryRevision ||
		handoff.TargetBuildRevision != gate.TargetBuildRevision ||
		handoff.TargetAdapter != gate.TargetAdapter ||
		handoff.Host != gate.Host ||
		handoff.SemanticFunctionalRun != gate.SemanticFunctionalRun ||
		handoff.VisualRun == "" ||
		handoff.GateCReadinessArtifactSHA256 != gate.GateCReadinessArtifactSHA256 ||
		handoff.SemanticFunctionalSummarySHA256 != gate.SemanticFunctionalSummarySHA256 {
		return nil, fmt.Errorf("R7A handoff does not share the exact actual Gate C provenance")
	}

	expected := replayableBehaviorR7BExpectedFirstWebBehaviors()
	if len(handoff.Candidates) != len(expected) {
		return nil, fmt.Errorf("R7A handoff behavior count=%d, want %d", len(handoff.Candidates), len(expected))
	}

	seen := map[string]struct{}{}
	out := make([]replayableBehaviorR7BExecutionIntake, 0, len(handoff.Candidates))
	for _, candidate := range handoff.Candidates {
		want, ok := expected[candidate.BehaviorID]
		if !ok {
			return nil, fmt.Errorf("unknown First Web Target behavior %q", candidate.BehaviorID)
		}
		if _, duplicate := seen[candidate.BehaviorID]; duplicate {
			return nil, fmt.Errorf("duplicate First Web Target behavior %q", candidate.BehaviorID)
		}
		seen[candidate.BehaviorID] = struct{}{}

		if candidate.SemanticOperation != want.SemanticOperation ||
			candidate.TargetRealizationRef != want.TargetRealizationRef ||
			candidate.RawArtifactFile != want.RawArtifactFile ||
			candidate.ObservationClass != want.ObservationClass {
			return nil, fmt.Errorf("behavior %q does not match the R7A contract", candidate.BehaviorID)
		}
		if candidate.RawArtifactSHA256 == "" ||
			candidate.VisualRun == "" ||
			candidate.VisualRun != handoff.VisualRun {
			return nil, fmt.Errorf("behavior %q lacks exact Evidence provenance", candidate.BehaviorID)
		}
		if candidate.ReplayStatus != "not-executed" ||
			candidate.CoverageClass != "behavior-unit" {
			return nil, fmt.Errorf("behavior %q carries fabricated replay or coverage state", candidate.BehaviorID)
		}
		if candidate.HandoverBaselineID != gate.HandoverBaselineID ||
			candidate.BaselineRepositoryRevision != gate.BaselineRepositoryRevision ||
			candidate.TargetBuildRevision != gate.TargetBuildRevision ||
			candidate.SemanticFunctionalRun != gate.SemanticFunctionalRun ||
			candidate.GateCReadinessArtifactSHA256 != gate.GateCReadinessArtifactSHA256 ||
			candidate.SemanticFunctionalSummarySHA256 != gate.SemanticFunctionalSummarySHA256 {
			return nil, fmt.Errorf("behavior %q does not preserve the exact actual Evidence chain", candidate.BehaviorID)
		}

		out = append(out, replayableBehaviorR7BExecutionIntake{
			BehaviorID:                      candidate.BehaviorID,
			SemanticOperation:               candidate.SemanticOperation,
			TargetRealizationRef:            candidate.TargetRealizationRef,
			RawArtifactFile:                 candidate.RawArtifactFile,
			RawArtifactSHA256:               candidate.RawArtifactSHA256,
			ObservationClass:                candidate.ObservationClass,
			ReplayExecutionStatus:           "not-executed",
			RepeatabilityStatus:             "not-observed",
			HandoffArtifactSHA256:           entry.HandoffArtifactSHA256,
			GateCReadinessArtifactSHA256:    gate.GateCReadinessArtifactSHA256,
			SemanticFunctionalSummarySHA256: gate.SemanticFunctionalSummarySHA256,
			HandoverBaselineID:              gate.HandoverBaselineID,
			BaselineRepositoryRevision:      gate.BaselineRepositoryRevision,
			TargetBuildRevision:             gate.TargetBuildRevision,
			SemanticFunctionalRun:           gate.SemanticFunctionalRun,
			VisualRun:                       candidate.VisualRun,
		})
	}
	if len(seen) != len(expected) {
		return nil, fmt.Errorf("R7A handoff behavior set is incomplete")
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].BehaviorID < out[j].BehaviorID
	})
	return out, nil
}

func replayableBehaviorR7BActualGateCRecord() replayableBehaviorR7BRecordedGateC {
	return replayableBehaviorR7BRecordedGateC{
		HandoverBaselineID:               "B",
		BaselineRepositoryRevision:       "e5231839a5dcc62cce1181a2731fcb7c70d16736",
		TargetBuildRevision:              "0ba87a4e0a96f70a198b03ff7652fb00e3826abe",
		TargetAdapter:                    "reference-first-web-target",
		Host:                             "go-net-http",
		SemanticFunctionalRun:            "first-web-target-0ba87a4e0a96",
		SemanticFunctionalSummarySHA256:  "78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b",
		GateCReadinessArtifactSHA256:     "b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c",
		GateCJudgmentRecordRepositorySHA: "d34edb9273f5f138abcbc23f4dcb000dcb5cdf20",
		GateCReady:                       true,
	}
}

func replayableBehaviorR7BExpectedFirstWebBehaviors() map[string]replayableBehaviorR7BExpectedBehavior {
	return map[string]replayableBehaviorR7BExpectedBehavior{
		"first-web/read-only": {
			SemanticOperation:    "order-entry.read-only-web-slice",
			TargetRealizationRef: "first-web-target:read",
			RawArtifactFile:      "read.json",
			ObservationClass:     "read-result",
		},
		"first-web/command-success": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:submit-success",
			RawArtifactFile:      "submit-success.json",
			ObservationClass:     "business-result-plus-separate-observation",
		},
		"first-web/authentication-denied": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:authentication-denied",
			RawArtifactFile:      "submit-authentication-denied.json",
			ObservationClass:     "business-denial-authentication",
		},
		"first-web/authorization-denied": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:authorization-denied",
			RawArtifactFile:      "submit-authorization-denied.json",
			ObservationClass:     "business-denial-authorization",
		},
		"first-web/validation-not-satisfied": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:validation-not-satisfied",
			RawArtifactFile:      "submit-validation-not-satisfied.json",
			ObservationClass:     "validation-not-satisfied",
		},
		"first-web/execution-error": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:execution-error",
			RawArtifactFile:      "submit-execution-error.json",
			ObservationClass:     "command-invocation-execution-error",
		},
	}
}

func replayableBehaviorR7BTestEntry() replayableBehaviorR7BEntry {
	gate := replayableBehaviorR7BActualGateCRecord()
	expected := replayableBehaviorR7BExpectedFirstWebBehaviors()
	ids := make([]string, 0, len(expected))
	for id := range expected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	candidates := make([]replayableBehaviorR7BHandoffCandidate, 0, len(ids))
	for _, id := range ids {
		spec := expected[id]
		candidates = append(candidates, replayableBehaviorR7BHandoffCandidate{
			BehaviorID:                      id,
			SemanticOperation:               spec.SemanticOperation,
			TargetRealizationRef:            spec.TargetRealizationRef,
			RawArtifactFile:                 spec.RawArtifactFile,
			RawArtifactSHA256:               "test-only-sha256-" + id,
			ObservationClass:                spec.ObservationClass,
			ReplayStatus:                    "not-executed",
			CoverageClass:                   "behavior-unit",
			HandoverBaselineID:              gate.HandoverBaselineID,
			BaselineRepositoryRevision:      gate.BaselineRepositoryRevision,
			TargetBuildRevision:             gate.TargetBuildRevision,
			SemanticFunctionalRun:           gate.SemanticFunctionalRun,
			VisualRun:                       "visual-T",
			GateCReadinessArtifactSHA256:    gate.GateCReadinessArtifactSHA256,
			SemanticFunctionalSummarySHA256: gate.SemanticFunctionalSummarySHA256,
		})
	}

	return replayableBehaviorR7BEntry{
		GateCRecord: gate,
		Handoff: replayableBehaviorR7AHandoffArtifact{
			EvidenceNature:                  "first-web-target-r7a-replayable-behavior-handoff",
			HandoverBaselineID:              gate.HandoverBaselineID,
			BaselineRepositoryRevision:      gate.BaselineRepositoryRevision,
			TargetBuildRevision:             gate.TargetBuildRevision,
			TargetAdapter:                   gate.TargetAdapter,
			Host:                            gate.Host,
			SemanticFunctionalRun:           gate.SemanticFunctionalRun,
			VisualRun:                       "visual-T",
			GateCReadinessArtifactSHA256:    gate.GateCReadinessArtifactSHA256,
			SemanticFunctionalSummarySHA256: gate.SemanticFunctionalSummarySHA256,
			Candidates:                      candidates,
		},
		HandoffArtifactSHA256: "test-only-sha256-r7a-handoff-artifact",
	}
}
