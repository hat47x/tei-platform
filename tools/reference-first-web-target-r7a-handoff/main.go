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

type passState struct {
	Passed bool `json:"passed"`
}

type artifactDigest struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type semanticFunctionalSummary struct {
	EvidenceNature             string           `json:"evidenceNature"`
	RunIdentity                string           `json:"runIdentity"`
	HandoverBaselineID         string           `json:"handoverBaselineId"`
	BaselineRepositoryRevision string           `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string           `json:"targetBuildRevision"`
	TargetAdapter              string           `json:"targetAdapter"`
	Host                       string           `json:"host"`
	SourceQualification        passState        `json:"sourceQualification"`
	SemanticConformance        passState        `json:"semanticConformance"`
	FunctionalEvidence         passState        `json:"functionalEvidence"`
	Artifacts                  []artifactDigest `json:"artifacts"`
	GateCReady                 bool             `json:"gateCReady"`
}

type readinessCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
}

type unresolvedDisposition struct {
	ID       string `json:"id"`
	Blocking bool   `json:"blocking"`
}

type readinessTargetDifference struct {
	ApplicationID string `json:"applicationId"`
	TargetAdapter string `json:"targetAdapter"`
	Host          string `json:"host"`
}

type gateCReadiness struct {
	EvidenceNature             string                    `json:"evidenceNature"`
	HandoverBaselineID         string                    `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                    `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                    `json:"targetBuildRevision"`
	TargetAdapter              string                    `json:"targetAdapter"`
	Host                       string                    `json:"host"`
	SemanticFunctionalRun      string                    `json:"semanticFunctionalRun"`
	VisualRun                  string                    `json:"visualRun"`
	Checks                     []readinessCheck          `json:"checks"`
	Unresolved                 []unresolvedDisposition   `json:"unresolved"`
	TargetDifferenceHandoff    readinessTargetDifference `json:"targetDifferenceHandoff"`
	GateCReady                 bool                      `json:"gateCReady"`
}

type behaviorSpec struct {
	BehaviorID           string
	SemanticOperation    string
	TargetRealizationRef string
	RawArtifactFile      string
	ObservationClass     string
}

type replayableBehaviorCandidate struct {
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
	EvidenceNature                   string                        `json:"evidenceNature"`
	HandoverBaselineID               string                        `json:"handoverBaselineId"`
	BaselineRepositoryRevision       string                        `json:"baselineRepositoryRevision"`
	TargetBuildRevision              string                        `json:"targetBuildRevision"`
	TargetAdapter                    string                        `json:"targetAdapter"`
	Host                             string                        `json:"host"`
	SemanticFunctionalRun            string                        `json:"semanticFunctionalRun"`
	VisualRun                        string                        `json:"visualRun"`
	GateCReadinessArtifactSHA256     string                        `json:"gateCReadinessArtifactSHA256"`
	SemanticFunctionalSummarySHA256  string                        `json:"semanticFunctionalSummarySHA256"`
	Candidates                       []replayableBehaviorCandidate `json:"candidates"`
	AuthorityLimits                  []string                      `json:"authorityLimits"`
}

func main() {
	var (
		readinessPath = flag.String("gate-c-readiness", "", "Gate C final readiness audit JSON")
		semanticPath  = flag.String("semantic-functional-summary", "", "Semantic / Functional summary JSON")
	)
	flag.Parse()
	if *readinessPath == "" || *semanticPath == "" {
		fatalf("-gate-c-readiness and -semantic-functional-summary are required")
	}

	var readiness gateCReadiness
	readinessDigest, err := readJSONWithDigest(*readinessPath, &readiness)
	if err != nil {
		fatalf("%v", err)
	}
	var semantic semanticFunctionalSummary
	semanticDigest, err := readJSONWithDigest(*semanticPath, &semantic)
	if err != nil {
		fatalf("%v", err)
	}

	handoff, err := buildHandoff(
		readiness,
		semantic,
		readinessDigest,
		semanticDigest,
		firstWebBehaviorSpecs(),
	)
	if err != nil {
		fatalf("%v", err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(handoff); err != nil {
		fatalf("encode R7A handoff: %v", err)
	}
}

func buildHandoff(
	readiness gateCReadiness,
	semantic semanticFunctionalSummary,
	readinessDigest,
	semanticDigest string,
	specs []behaviorSpec,
) (handoffSummary, error) {
	if readiness.EvidenceNature != "first-web-target-gate-c-readiness-audit" {
		return handoffSummary{}, fmt.Errorf(
			"unexpected Gate C readiness evidenceNature %q",
			readiness.EvidenceNature,
		)
	}
	if !readiness.GateCReady {
		return handoffSummary{}, fmt.Errorf("Gate C is not ready")
	}
	if readiness.HandoverBaselineID == "" ||
		readiness.BaselineRepositoryRevision == "" ||
		readiness.TargetBuildRevision == "" ||
		readiness.TargetAdapter == "" ||
		readiness.Host == "" ||
		readiness.SemanticFunctionalRun == "" ||
		readiness.VisualRun == "" {
		return handoffSummary{}, fmt.Errorf("Gate C readiness provenance is incomplete")
	}
	if readinessDigest == "" || semanticDigest == "" {
		return handoffSummary{}, fmt.Errorf("R7A input artifact digest is incomplete")
	}
	if err := validateReadinessChecks(readiness.Checks); err != nil {
		return handoffSummary{}, err
	}
	for _, unresolved := range readiness.Unresolved {
		if unresolved.ID == "" {
			return handoffSummary{}, fmt.Errorf(
				"Gate C readiness contains unnamed unresolved item",
			)
		}
		if unresolved.Blocking {
			return handoffSummary{}, fmt.Errorf(
				"blocking unresolved %q remains",
				unresolved.ID,
			)
		}
	}
	if readiness.TargetDifferenceHandoff.ApplicationID != "order_entry" ||
		readiness.TargetDifferenceHandoff.TargetAdapter != readiness.TargetAdapter ||
		readiness.TargetDifferenceHandoff.Host != readiness.Host {
		return handoffSummary{}, fmt.Errorf(
			"Gate C target-difference handoff does not describe the ready First Web Target",
		)
	}

	if semantic.EvidenceNature != "first-web-target-semantic-functional-evidence" {
		return handoffSummary{}, fmt.Errorf(
			"unexpected Semantic / Functional evidenceNature %q",
			semantic.EvidenceNature,
		)
	}
	if !semantic.SourceQualification.Passed ||
		!semantic.SemanticConformance.Passed ||
		!semantic.FunctionalEvidence.Passed {
		return handoffSummary{}, fmt.Errorf(
			"Semantic / Functional Evidence is not fully passed",
		)
	}
	if semantic.GateCReady {
		return handoffSummary{}, fmt.Errorf(
			"Semantic / Functional summary must remain pre-Visual and must not declare Gate C Ready",
		)
	}
	if semantic.RunIdentity == "" ||
		semantic.HandoverBaselineID == "" ||
		semantic.BaselineRepositoryRevision == "" ||
		semantic.TargetBuildRevision == "" ||
		semantic.TargetAdapter == "" ||
		semantic.Host == "" {
		return handoffSummary{}, fmt.Errorf(
			"Semantic / Functional provenance is incomplete",
		)
	}
	if readiness.SemanticFunctionalRun != semantic.RunIdentity {
		return handoffSummary{}, fmt.Errorf(
			"Semantic / Functional run identity mismatch",
		)
	}
	if readiness.HandoverBaselineID != semantic.HandoverBaselineID ||
		readiness.BaselineRepositoryRevision != semantic.BaselineRepositoryRevision ||
		readiness.TargetBuildRevision != semantic.TargetBuildRevision ||
		readiness.TargetAdapter != semantic.TargetAdapter ||
		readiness.Host != semantic.Host {
		return handoffSummary{}, fmt.Errorf(
			"Gate C and Semantic / Functional Evidence do not share exact B/R/T/Target provenance",
		)
	}

	digests := make(map[string]string, len(semantic.Artifacts))
	for _, artifact := range semantic.Artifacts {
		if artifact.File == "" || artifact.SHA256 == "" {
			return handoffSummary{}, fmt.Errorf(
				"Semantic / Functional raw artifact digest is incomplete: %+v",
				artifact,
			)
		}
		if _, exists := digests[artifact.File]; exists {
			return handoffSummary{}, fmt.Errorf(
				"duplicate raw artifact digest for %q",
				artifact.File,
			)
		}
		digests[artifact.File] = artifact.SHA256
	}

	seen := map[string]struct{}{}
	candidates := make([]replayableBehaviorCandidate, 0, len(specs))
	for _, spec := range specs {
		if spec.BehaviorID == "" ||
			spec.SemanticOperation == "" ||
			spec.TargetRealizationRef == "" ||
			spec.RawArtifactFile == "" ||
			spec.ObservationClass == "" {
			return handoffSummary{}, fmt.Errorf(
				"incomplete replayable behavior spec: %+v",
				spec,
			)
		}
		if _, exists := seen[spec.BehaviorID]; exists {
			return handoffSummary{}, fmt.Errorf(
				"duplicate behavior identity %q",
				spec.BehaviorID,
			)
		}
		seen[spec.BehaviorID] = struct{}{}

		digest := digests[spec.RawArtifactFile]
		if digest == "" {
			return handoffSummary{}, fmt.Errorf(
				"raw artifact %q has no exact digest",
				spec.RawArtifactFile,
			)
		}
		candidates = append(candidates, replayableBehaviorCandidate{
			BehaviorID:                      spec.BehaviorID,
			SemanticOperation:               spec.SemanticOperation,
			TargetRealizationRef:            spec.TargetRealizationRef,
			RawArtifactFile:                 spec.RawArtifactFile,
			RawArtifactSHA256:               digest,
			ObservationClass:                spec.ObservationClass,
			ReplayStatus:                    "not-executed",
			CoverageClass:                   "behavior-unit",
			HandoverBaselineID:              readiness.HandoverBaselineID,
			BaselineRepositoryRevision:      readiness.BaselineRepositoryRevision,
			TargetBuildRevision:             readiness.TargetBuildRevision,
			SemanticFunctionalRun:           readiness.SemanticFunctionalRun,
			VisualRun:                       readiness.VisualRun,
			GateCReadinessArtifactSHA256:    readinessDigest,
			SemanticFunctionalSummarySHA256: semanticDigest,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].BehaviorID < candidates[j].BehaviorID
	})

	return handoffSummary{
		EvidenceNature:                  "first-web-target-r7a-replayable-behavior-handoff",
		HandoverBaselineID:              readiness.HandoverBaselineID,
		BaselineRepositoryRevision:      readiness.BaselineRepositoryRevision,
		TargetBuildRevision:             readiness.TargetBuildRevision,
		TargetAdapter:                   readiness.TargetAdapter,
		Host:                            readiness.Host,
		SemanticFunctionalRun:           readiness.SemanticFunctionalRun,
		VisualRun:                       readiness.VisualRun,
		GateCReadinessArtifactSHA256:    readinessDigest,
		SemanticFunctionalSummarySHA256: semanticDigest,
		Candidates:                      candidates,
		AuthorityLimits: []string{
			"handoff-requires-actual-final-gate-c-readiness",
			"handoff-pins-exact-readiness-and-semantic-summary-bytes",
			"candidate-replay-status-starts-not-executed",
			"behavior-unit-does-not-imply-workflow-coverage",
			"target-realization-ref-is-not-canonical-semantic-identity",
			"raw-artifact-digest-preserves-evidence-provenance",
			"visual-run-is-entry-provenance-not-functional-replay-result",
			"selectors-coordinates-and-click-sequences-are-not-canonicalized",
		},
	}, nil
}

func validateReadinessChecks(checks []readinessCheck) error {
	expected := expectedReadinessChecks()
	if len(checks) != len(expected) {
		return fmt.Errorf(
			"Gate C readiness check count=%d, want %d",
			len(checks),
			len(expected),
		)
	}
	seen := map[string]struct{}{}
	for _, check := range checks {
		if check.Name == "" || !check.Passed {
			return fmt.Errorf(
				"Gate C readiness check is not passed: %+v",
				check,
			)
		}
		if _, ok := expected[check.Name]; !ok {
			return fmt.Errorf("unknown Gate C readiness check %q", check.Name)
		}
		if _, duplicate := seen[check.Name]; duplicate {
			return fmt.Errorf("duplicate Gate C readiness check %q", check.Name)
		}
		seen[check.Name] = struct{}{}
	}
	return nil
}

func expectedReadinessChecks() map[string]struct{} {
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

func firstWebBehaviorSpecs() []behaviorSpec {
	return []behaviorSpec{
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

func readJSONWithDigest(path string, target any) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if err := json.Unmarshal(data, target); err != nil {
		return "", fmt.Errorf("decode %s: %w", path, err)
	}
	return digest, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
