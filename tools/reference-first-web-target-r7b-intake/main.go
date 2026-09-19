package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
)

const (
	expectedGateCJudgmentRevision = "d34edb9273f5f138abcbc23f4dcb000dcb5cdf20"
	expectedBaselineRevision      = "e5231839a5dcc62cce1181a2731fcb7c70d16736"
	expectedTargetRevision        = "0ba87a4e0a96f70a198b03ff7652fb00e3826abe"
	expectedSemanticRun           = "first-web-target-0ba87a4e0a96"
	expectedVisualRun             = "visual-T"
	expectedReadinessSHA256       = "b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c"
	expectedSemanticSHA256        = "78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b"
)

type acquisitionReceipt struct {
	EvidenceNature                        string `json:"evidenceNature"`
	GateCJudgmentRecordRepositoryRevision string `json:"gateCJudgmentRecordRepositoryRevision"`
	RecordedTargetBuildRevision           string `json:"recordedTargetBuildRevision"`
	HandoffToolRepositoryRevision         string `json:"handoffToolRepositoryRevision"`
	RepositoryPostRunRevision             string `json:"repositoryPostRunRevision"`
	RepositoryPostRunClean                bool   `json:"repositoryPostRunClean"`
	SourceQualificationPassed             bool   `json:"sourceQualificationPassed"`
	GateCReadinessArtifactSHA256          string `json:"gateCReadinessArtifactSHA256"`
	SemanticFunctionalSummarySHA256       string `json:"semanticFunctionalSummarySHA256"`
	HandoffArtifact                       string `json:"handoffArtifact"`
	HandoffArtifactSHA256                 string `json:"handoffArtifactSHA256"`
}

type handoffCandidate struct {
	BehaviorID                      string `json:"behaviorId"`
	SemanticOperation               string `json:"semanticOperation"`
	TargetRealizationRef            string `json:"targetRealizationRef"`
	RawArtifactFile                 string `json:"rawArtifactFile"`
	RawArtifactSHA256               string `json:"rawArtifactSHA256"`
	ObservationClass                string `json:"observationClass"`
	ReplayStatus                    string `json:"replayStatus"`
	CoverageClass                   string `json:"coverageClass"`
	HandoverBaselineID              string `json:"handoverBaselineId"`
	BaselineRepositoryRevision      string `json:"baselineRepositoryRevision"`
	TargetBuildRevision             string `json:"targetBuildRevision"`
	SemanticFunctionalRun           string `json:"semanticFunctionalRun"`
	VisualRun                       string `json:"visualRun"`
	GateCReadinessArtifactSHA256    string `json:"gateCReadinessArtifactSHA256"`
	SemanticFunctionalSummarySHA256 string `json:"semanticFunctionalSummarySHA256"`
}

type handoffSummary struct {
	EvidenceNature                  string             `json:"evidenceNature"`
	HandoverBaselineID              string             `json:"handoverBaselineId"`
	BaselineRepositoryRevision      string             `json:"baselineRepositoryRevision"`
	TargetBuildRevision             string             `json:"targetBuildRevision"`
	TargetAdapter                   string             `json:"targetAdapter"`
	Host                            string             `json:"host"`
	SemanticFunctionalRun           string             `json:"semanticFunctionalRun"`
	VisualRun                       string             `json:"visualRun"`
	GateCReadinessArtifactSHA256    string             `json:"gateCReadinessArtifactSHA256"`
	SemanticFunctionalSummarySHA256 string             `json:"semanticFunctionalSummarySHA256"`
	Candidates                      []handoffCandidate `json:"candidates"`
	AuthorityLimits                 []string           `json:"authorityLimits"`
}

type behaviorSpec struct {
	SemanticOperation    string
	TargetRealizationRef string
	RawArtifactFile      string
	ObservationClass     string
}

type executionIntakeCandidate struct {
	BehaviorID                      string `json:"behaviorId"`
	SemanticOperation               string `json:"semanticOperation"`
	TargetRealizationRef            string `json:"targetRealizationRef"`
	RawArtifactFile                 string `json:"rawArtifactFile"`
	RawArtifactSHA256               string `json:"rawArtifactSHA256"`
	ObservationClass                string `json:"observationClass"`
	ReplayExecutionStatus           string `json:"replayExecutionStatus"`
	RepeatabilityStatus             string `json:"repeatabilityStatus"`
	HandoffArtifactSHA256           string `json:"handoffArtifactSHA256"`
	GateCReadinessArtifactSHA256    string `json:"gateCReadinessArtifactSHA256"`
	SemanticFunctionalSummarySHA256 string `json:"semanticFunctionalSummarySHA256"`
	HandoverBaselineID              string `json:"handoverBaselineId"`
	BaselineRepositoryRevision      string `json:"baselineRepositoryRevision"`
	TargetBuildRevision             string `json:"targetBuildRevision"`
	SemanticFunctionalRun           string `json:"semanticFunctionalRun"`
	VisualRun                       string `json:"visualRun"`
}

type executionIntakeSummary struct {
	EvidenceNature                        string                     `json:"evidenceNature"`
	SourceHandoffArtifact                 string                     `json:"sourceHandoffArtifact"`
	SourceHandoffArtifactSHA256           string                     `json:"sourceHandoffArtifactSHA256"`
	AcquisitionReceiptSHA256              string                     `json:"acquisitionReceiptSHA256"`
	HandoffToolRepositoryRevision         string                     `json:"handoffToolRepositoryRevision"`
	GateCJudgmentRecordRepositoryRevision string                     `json:"gateCJudgmentRecordRepositoryRevision"`
	HandoverBaselineID                    string                     `json:"handoverBaselineId"`
	BaselineRepositoryRevision            string                     `json:"baselineRepositoryRevision"`
	TargetBuildRevision                   string                     `json:"targetBuildRevision"`
	TargetAdapter                         string                     `json:"targetAdapter"`
	Host                                  string                     `json:"host"`
	SemanticFunctionalRun                 string                     `json:"semanticFunctionalRun"`
	VisualRun                             string                     `json:"visualRun"`
	GateCReadinessArtifactSHA256          string                     `json:"gateCReadinessArtifactSHA256"`
	SemanticFunctionalSummarySHA256       string                     `json:"semanticFunctionalSummarySHA256"`
	Candidates                            []executionIntakeCandidate `json:"candidates"`
	AuthorityLimits                       []string                   `json:"authorityLimits"`
}

func main() {
	var (
		handoffPath = flag.String("r7a-handoff", "", "R7A replayable behavior handoff JSON")
		receiptPath = flag.String("r7a-receipt", "", "R7A handoff acquisition receipt JSON")
	)
	flag.Parse()
	if *handoffPath == "" || *receiptPath == "" {
		fatalf("-r7a-handoff and -r7a-receipt are required")
	}

	var handoff handoffSummary
	handoffDigest, err := readJSONWithSHA256(*handoffPath, &handoff)
	if err != nil {
		fatalf("%v", err)
	}
	var receipt acquisitionReceipt
	receiptDigest, err := readJSONWithSHA256(*receiptPath, &receipt)
	if err != nil {
		fatalf("%v", err)
	}

	intake, err := buildExecutionIntake(handoff, receipt, handoffDigest, receiptDigest)
	if err != nil {
		fatalf("%v", err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(intake); err != nil {
		fatalf("encode R7B execution intake: %v", err)
	}
}

func buildExecutionIntake(
	handoff handoffSummary,
	receipt acquisitionReceipt,
	handoffDigest,
	receiptDigest string,
) (executionIntakeSummary, error) {
	if !isSHA256(handoffDigest) || !isSHA256(receiptDigest) {
		return executionIntakeSummary{}, fmt.Errorf("R7B input artifact identity is incomplete")
	}
	if err := validateReceipt(receipt, handoffDigest); err != nil {
		return executionIntakeSummary{}, err
	}
	if err := validateHandoff(handoff, receipt); err != nil {
		return executionIntakeSummary{}, err
	}

	expected := expectedBehaviors()
	seen := make(map[string]struct{}, len(expected))
	candidates := make([]executionIntakeCandidate, 0, len(handoff.Candidates))
	for _, candidate := range handoff.Candidates {
		want, ok := expected[candidate.BehaviorID]
		if !ok {
			return executionIntakeSummary{}, fmt.Errorf("unknown First Web Target behavior %q", candidate.BehaviorID)
		}
		if _, duplicate := seen[candidate.BehaviorID]; duplicate {
			return executionIntakeSummary{}, fmt.Errorf("duplicate First Web Target behavior %q", candidate.BehaviorID)
		}
		seen[candidate.BehaviorID] = struct{}{}
		if candidate.SemanticOperation != want.SemanticOperation ||
			candidate.TargetRealizationRef != want.TargetRealizationRef ||
			candidate.RawArtifactFile != want.RawArtifactFile ||
			candidate.ObservationClass != want.ObservationClass {
			return executionIntakeSummary{}, fmt.Errorf("behavior %q does not match the R7A contract", candidate.BehaviorID)
		}
		if !isSHA256(candidate.RawArtifactSHA256) {
			return executionIntakeSummary{}, fmt.Errorf("behavior %q raw artifact identity is not SHA-256", candidate.BehaviorID)
		}
		if candidate.ReplayStatus != "not-executed" || candidate.CoverageClass != "behavior-unit" {
			return executionIntakeSummary{}, fmt.Errorf("behavior %q carries replay or coverage state that is not valid for R7B entry", candidate.BehaviorID)
		}
		if candidate.HandoverBaselineID != handoff.HandoverBaselineID ||
			candidate.BaselineRepositoryRevision != handoff.BaselineRepositoryRevision ||
			candidate.TargetBuildRevision != handoff.TargetBuildRevision ||
			candidate.SemanticFunctionalRun != handoff.SemanticFunctionalRun ||
			candidate.VisualRun != handoff.VisualRun ||
			candidate.GateCReadinessArtifactSHA256 != handoff.GateCReadinessArtifactSHA256 ||
			candidate.SemanticFunctionalSummarySHA256 != handoff.SemanticFunctionalSummarySHA256 {
			return executionIntakeSummary{}, fmt.Errorf("behavior %q does not preserve exact R7A handoff provenance", candidate.BehaviorID)
		}

		candidates = append(candidates, executionIntakeCandidate{
			BehaviorID:                      candidate.BehaviorID,
			SemanticOperation:               candidate.SemanticOperation,
			TargetRealizationRef:            candidate.TargetRealizationRef,
			RawArtifactFile:                 candidate.RawArtifactFile,
			RawArtifactSHA256:               candidate.RawArtifactSHA256,
			ObservationClass:                candidate.ObservationClass,
			ReplayExecutionStatus:           "not-executed",
			RepeatabilityStatus:             "not-observed",
			HandoffArtifactSHA256:           handoffDigest,
			GateCReadinessArtifactSHA256:    handoff.GateCReadinessArtifactSHA256,
			SemanticFunctionalSummarySHA256: handoff.SemanticFunctionalSummarySHA256,
			HandoverBaselineID:              handoff.HandoverBaselineID,
			BaselineRepositoryRevision:      handoff.BaselineRepositoryRevision,
			TargetBuildRevision:             handoff.TargetBuildRevision,
			SemanticFunctionalRun:           handoff.SemanticFunctionalRun,
			VisualRun:                       handoff.VisualRun,
		})
	}
	if len(seen) != len(expected) {
		return executionIntakeSummary{}, fmt.Errorf("R7A handoff behavior set is incomplete")
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].BehaviorID < candidates[j].BehaviorID
	})

	return executionIntakeSummary{
		EvidenceNature:                        "first-web-target-r7b-execution-intake",
		SourceHandoffArtifact:                 receipt.HandoffArtifact,
		SourceHandoffArtifactSHA256:           handoffDigest,
		AcquisitionReceiptSHA256:              receiptDigest,
		HandoffToolRepositoryRevision:         receipt.HandoffToolRepositoryRevision,
		GateCJudgmentRecordRepositoryRevision: receipt.GateCJudgmentRecordRepositoryRevision,
		HandoverBaselineID:                    handoff.HandoverBaselineID,
		BaselineRepositoryRevision:            handoff.BaselineRepositoryRevision,
		TargetBuildRevision:                   handoff.TargetBuildRevision,
		TargetAdapter:                         handoff.TargetAdapter,
		Host:                                  handoff.Host,
		SemanticFunctionalRun:                 handoff.SemanticFunctionalRun,
		VisualRun:                             handoff.VisualRun,
		GateCReadinessArtifactSHA256:          handoff.GateCReadinessArtifactSHA256,
		SemanticFunctionalSummarySHA256:       handoff.SemanticFunctionalSummarySHA256,
		Candidates:                            candidates,
		AuthorityLimits: []string{
			"execution-intake-does-not-execute-replay",
			"replay-execution-status-starts-not-executed",
			"repeatability-status-starts-not-observed",
			"behavior-unit-does-not-imply-workflow-coverage",
			"target-realization-ref-is-not-canonical-semantic-identity",
			"selectors-coordinates-and-click-sequences-are-not-canonicalized",
			"gate-c-ready-and-handoff-acquired-do-not-imply-replay-success",
		},
	}, nil
}

func validateReceipt(receipt acquisitionReceipt, handoffDigest string) error {
	if receipt.EvidenceNature != "first-web-target-r7a-handoff-acquisition-receipt" {
		return fmt.Errorf("unexpected R7A receipt evidenceNature %q", receipt.EvidenceNature)
	}
	if !receipt.RepositoryPostRunClean || !receipt.SourceQualificationPassed {
		return fmt.Errorf("R7A acquisition receipt does not record a clean qualified acquisition")
	}
	if receipt.GateCJudgmentRecordRepositoryRevision != expectedGateCJudgmentRevision ||
		receipt.RecordedTargetBuildRevision != expectedTargetRevision {
		return fmt.Errorf("R7A acquisition receipt does not point to the recorded actual Gate C determination")
	}
	if !isGitRevision(receipt.HandoffToolRepositoryRevision) ||
		receipt.RepositoryPostRunRevision != receipt.HandoffToolRepositoryRevision {
		return fmt.Errorf("R7A acquisition receipt tool revision is incomplete or changed during acquisition")
	}
	if receipt.GateCReadinessArtifactSHA256 != expectedReadinessSHA256 ||
		receipt.SemanticFunctionalSummarySHA256 != expectedSemanticSHA256 {
		return fmt.Errorf("R7A acquisition receipt does not preserve the recorded Gate C artifact identities")
	}
	if receipt.HandoffArtifact != "r7a-replayable-behavior-handoff.json" {
		return fmt.Errorf("unexpected R7A handoff artifact name %q", receipt.HandoffArtifact)
	}
	if !isSHA256(receipt.HandoffArtifactSHA256) || receipt.HandoffArtifactSHA256 != handoffDigest {
		return fmt.Errorf("R7A acquisition receipt handoff digest does not match exact handoff bytes")
	}
	return nil
}

func validateHandoff(handoff handoffSummary, receipt acquisitionReceipt) error {
	if handoff.EvidenceNature != "first-web-target-r7a-replayable-behavior-handoff" {
		return fmt.Errorf("unexpected R7A handoff evidenceNature %q", handoff.EvidenceNature)
	}
	if handoff.HandoverBaselineID != "B" ||
		handoff.BaselineRepositoryRevision != expectedBaselineRevision ||
		handoff.TargetBuildRevision != expectedTargetRevision ||
		handoff.TargetAdapter != "reference-first-web-target" ||
		handoff.Host != "go-net-http" ||
		handoff.SemanticFunctionalRun != expectedSemanticRun ||
		handoff.VisualRun != expectedVisualRun {
		return fmt.Errorf("R7A handoff does not share exact actual Gate C provenance")
	}
	if handoff.GateCReadinessArtifactSHA256 != expectedReadinessSHA256 ||
		handoff.SemanticFunctionalSummarySHA256 != expectedSemanticSHA256 ||
		handoff.GateCReadinessArtifactSHA256 != receipt.GateCReadinessArtifactSHA256 ||
		handoff.SemanticFunctionalSummarySHA256 != receipt.SemanticFunctionalSummarySHA256 {
		return fmt.Errorf("R7A handoff Evidence identities do not match the acquisition receipt")
	}
	if len(handoff.Candidates) != len(expectedBehaviors()) {
		return fmt.Errorf("R7A handoff behavior count=%d, want %d", len(handoff.Candidates), len(expectedBehaviors()))
	}
	return nil
}

func expectedBehaviors() map[string]behaviorSpec {
	return map[string]behaviorSpec{
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

func readJSONWithSHA256(path string, target any) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return "", fmt.Errorf("decode %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func isSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func isGitRevision(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
