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
	expectedBaselineRevision      = "e5231839a5dcc62cce1181a2731fcb7c70d16736"
	expectedTargetRevision        = "0ba87a4e0a96f70a198b03ff7652fb00e3826abe"
	expectedSemanticRun           = "first-web-target-0ba87a4e0a96"
	expectedVisualRun             = "visual-T"
	expectedReadinessSHA256       = "b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c"
	expectedSemanticSHA256        = "78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b"
	expectedGateCJudgmentRevision = "d34edb9273f5f138abcbc23f4dcb000dcb5cdf20"
)

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

type executionIntake struct {
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

type behaviorSpec struct {
	SemanticOperation    string
	TargetRealizationRef string
	RawArtifactFile      string
	ObservationClass     string
	RequestFixtureID     string
	TargetLocalAction    string
	ExpectedHTTPStatus   int
}

type replayUnit struct {
	BehaviorID                string `json:"behaviorId"`
	SemanticOperation         string `json:"semanticOperation"`
	TargetRealizationRef      string `json:"targetRealizationRef"`
	RequestFixtureID          string `json:"requestFixtureId"`
	TargetLocalInteraction    string `json:"targetLocalInteraction"`
	ExpectedHTTPStatus        int    `json:"expectedHTTPStatus"`
	ExpectedObservationClass  string `json:"expectedObservationClass"`
	OriginRawArtifactFile     string `json:"originRawArtifactFile"`
	OriginRawArtifactSHA256   string `json:"originRawArtifactSHA256"`
	OriginTargetBuildRevision string `json:"originTargetBuildRevision"`
	ReplayTargetBuildRevision string `json:"replayTargetBuildRevision"`
	RequiredDistinctRuns      int    `json:"requiredDistinctRuns"`
	ReplayExecutionStatus     string `json:"replayExecutionStatus"`
	RepeatabilityStatus       string `json:"repeatabilityStatus"`
	CoverageClass             string `json:"coverageClass"`
}

type replayPlan struct {
	EvidenceNature                    string       `json:"evidenceNature"`
	SourceExecutionIntakeSHA256       string       `json:"sourceExecutionIntakeSHA256"`
	SourceR7AHandoffSHA256            string       `json:"sourceR7AHandoffSHA256"`
	SourceR7AAcquisitionReceiptSHA256 string       `json:"sourceR7AAcquisitionReceiptSHA256"`
	ReplayTargetBuildRevision         string       `json:"replayTargetBuildRevision"`
	Units                             []replayUnit `json:"units"`
	AuthorityLimits                   []string     `json:"authorityLimits"`
}

func main() {
	intakePath := flag.String("r7b-intake", "", "R7B execution-intake JSON")
	flag.Parse()
	if *intakePath == "" {
		fatalf("-r7b-intake is required")
	}

	var intake executionIntake
	intakeSHA, err := readJSONWithSHA256(*intakePath, &intake)
	if err != nil {
		fatalf("%v", err)
	}
	plan, err := buildReplayPlan(intake, intakeSHA)
	if err != nil {
		fatalf("%v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(plan); err != nil {
		fatalf("encode R7B replay plan: %v", err)
	}
}

func buildReplayPlan(intake executionIntake, intakeSHA string) (replayPlan, error) {
	if !isSHA256(intakeSHA) {
		return replayPlan{}, fmt.Errorf("R7B execution intake exact-byte identity is invalid")
	}
	if err := validateIntake(intake); err != nil {
		return replayPlan{}, err
	}

	expected := expectedBehaviors()
	if len(intake.Candidates) != len(expected) {
		return replayPlan{}, fmt.Errorf(
			"R7B execution intake behavior count=%d, want %d",
			len(intake.Candidates),
			len(expected),
		)
	}

	seen := map[string]struct{}{}
	units := make([]replayUnit, 0, len(intake.Candidates))
	for _, candidate := range intake.Candidates {
		spec, ok := expected[candidate.BehaviorID]
		if !ok {
			return replayPlan{}, fmt.Errorf("unknown R7B behavior %q", candidate.BehaviorID)
		}
		if _, duplicate := seen[candidate.BehaviorID]; duplicate {
			return replayPlan{}, fmt.Errorf("duplicate R7B behavior %q", candidate.BehaviorID)
		}
		seen[candidate.BehaviorID] = struct{}{}

		if candidate.SemanticOperation != spec.SemanticOperation ||
			candidate.TargetRealizationRef != spec.TargetRealizationRef ||
			candidate.RawArtifactFile != spec.RawArtifactFile ||
			candidate.ObservationClass != spec.ObservationClass {
			return replayPlan{}, fmt.Errorf("behavior %q does not match the R7B intake contract", candidate.BehaviorID)
		}
		if !isSHA256(candidate.RawArtifactSHA256) ||
			candidate.RawArtifactSHA256 == stringsOfZeroes() {
			return replayPlan{}, fmt.Errorf("behavior %q origin raw artifact identity is invalid", candidate.BehaviorID)
		}
		if candidate.ReplayExecutionStatus != "not-executed" ||
			candidate.RepeatabilityStatus != "not-observed" {
			return replayPlan{}, fmt.Errorf("behavior %q entered replay plan with fabricated execution state", candidate.BehaviorID)
		}
		if candidate.HandoffArtifactSHA256 != intake.SourceHandoffArtifactSHA256 ||
			candidate.GateCReadinessArtifactSHA256 != intake.GateCReadinessArtifactSHA256 ||
			candidate.SemanticFunctionalSummarySHA256 != intake.SemanticFunctionalSummarySHA256 ||
			candidate.HandoverBaselineID != intake.HandoverBaselineID ||
			candidate.BaselineRepositoryRevision != intake.BaselineRepositoryRevision ||
			candidate.TargetBuildRevision != intake.TargetBuildRevision ||
			candidate.SemanticFunctionalRun != intake.SemanticFunctionalRun ||
			candidate.VisualRun != intake.VisualRun {
			return replayPlan{}, fmt.Errorf("behavior %q does not preserve exact R7B intake provenance", candidate.BehaviorID)
		}

		units = append(units, replayUnit{
			BehaviorID:                candidate.BehaviorID,
			SemanticOperation:         candidate.SemanticOperation,
			TargetRealizationRef:      candidate.TargetRealizationRef,
			RequestFixtureID:          spec.RequestFixtureID,
			TargetLocalInteraction:    spec.TargetLocalAction,
			ExpectedHTTPStatus:        spec.ExpectedHTTPStatus,
			ExpectedObservationClass:  candidate.ObservationClass,
			OriginRawArtifactFile:     candidate.RawArtifactFile,
			OriginRawArtifactSHA256:   candidate.RawArtifactSHA256,
			OriginTargetBuildRevision: candidate.TargetBuildRevision,
			ReplayTargetBuildRevision: candidate.TargetBuildRevision,
			RequiredDistinctRuns:      2,
			ReplayExecutionStatus:     "not-executed",
			RepeatabilityStatus:       "not-observed",
			CoverageClass:             "behavior-unit",
		})
	}
	if len(seen) != len(expected) {
		return replayPlan{}, fmt.Errorf("R7B replay plan behavior set is incomplete")
	}
	sort.Slice(units, func(i, j int) bool {
		return units[i].BehaviorID < units[j].BehaviorID
	})

	return replayPlan{
		EvidenceNature:                    "first-web-target-r7b-replay-plan",
		SourceExecutionIntakeSHA256:       intakeSHA,
		SourceR7AHandoffSHA256:            intake.SourceHandoffArtifactSHA256,
		SourceR7AAcquisitionReceiptSHA256: intake.AcquisitionReceiptSHA256,
		ReplayTargetBuildRevision:         intake.TargetBuildRevision,
		Units:                             units,
		AuthorityLimits: []string{
			"replay-plan-does-not-execute-replay",
			"request-fixture-id-is-target-local-not-canonical",
			"target-local-interaction-is-not-canonical-semantic-identity",
			"replay-target-is-pinned-to-origin-target-build",
			"two-required-runs-do-not-imply-they-have-executed",
			"behavior-unit-does-not-imply-workflow-coverage",
			"credentials-selectors-coordinates-and-click-sequences-are-not-plan-identity",
		},
	}, nil
}

func validateIntake(intake executionIntake) error {
	if intake.EvidenceNature != "first-web-target-r7b-execution-intake" {
		return fmt.Errorf("unexpected R7B intake evidenceNature %q", intake.EvidenceNature)
	}
	if intake.SourceHandoffArtifact != "r7a-replayable-behavior-handoff.json" ||
		!isSHA256(intake.SourceHandoffArtifactSHA256) ||
		!isSHA256(intake.AcquisitionReceiptSHA256) ||
		!isGitRevision(intake.HandoffToolRepositoryRevision) {
		return fmt.Errorf("R7B intake source identity is incomplete")
	}
	if intake.GateCJudgmentRecordRepositoryRevision != expectedGateCJudgmentRevision ||
		intake.HandoverBaselineID != "B" ||
		intake.BaselineRepositoryRevision != expectedBaselineRevision ||
		intake.TargetBuildRevision != expectedTargetRevision ||
		intake.TargetAdapter != "reference-first-web-target" ||
		intake.Host != "go-net-http" ||
		intake.SemanticFunctionalRun != expectedSemanticRun ||
		intake.VisualRun != expectedVisualRun ||
		intake.GateCReadinessArtifactSHA256 != expectedReadinessSHA256 ||
		intake.SemanticFunctionalSummarySHA256 != expectedSemanticSHA256 {
		return fmt.Errorf("R7B intake does not preserve exact recorded Gate C provenance")
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
			RequestFixtureID:     "gate-c/read-from-0830-row-0",
			TargetLocalAction:    "GET /scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0",
			ExpectedHTTPStatus:   200,
		},
		"first-web/command-success": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:submit-success",
			RawArtifactFile:      "submit-success.json",
			ObservationClass:     "business-result-plus-separate-observation",
			RequestFixtureID:     "gate-c/command-success",
			TargetLocalAction:    "POST /scenario/submit fixture=gate-c/command-success",
			ExpectedHTTPStatus:   200,
		},
		"first-web/authentication-denied": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:authentication-denied",
			RawArtifactFile:      "submit-authentication-denied.json",
			ObservationClass:     "business-denial-authentication",
			RequestFixtureID:     "gate-c/authentication-denied",
			TargetLocalAction:    "POST /scenario/submit fixture=gate-c/authentication-denied",
			ExpectedHTTPStatus:   200,
		},
		"first-web/authorization-denied": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:authorization-denied",
			RawArtifactFile:      "submit-authorization-denied.json",
			ObservationClass:     "business-denial-authorization",
			RequestFixtureID:     "gate-c/authorization-denied",
			TargetLocalAction:    "POST /scenario/submit fixture=gate-c/authorization-denied",
			ExpectedHTTPStatus:   200,
		},
		"first-web/validation-not-satisfied": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:validation-not-satisfied",
			RawArtifactFile:      "submit-validation-not-satisfied.json",
			ObservationClass:     "validation-not-satisfied",
			RequestFixtureID:     "gate-c/validation-not-satisfied",
			TargetLocalAction:    "POST /scenario/submit fixture=gate-c/validation-not-satisfied",
			ExpectedHTTPStatus:   200,
		},
		"first-web/execution-error": {
			SemanticOperation:    "order-entry.command-submit",
			TargetRealizationRef: "first-web-target:execution-error",
			RawArtifactFile:      "submit-execution-error.json",
			ObservationClass:     "command-invocation-execution-error",
			RequestFixtureID:     "gate-c/execution-error",
			TargetLocalAction:    "POST /scenario/submit fixture=gate-c/execution-error",
			ExpectedHTTPStatus:   422,
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

func stringsOfZeroes() string {
	return "0000000000000000000000000000000000000000000000000000000000000000"
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
