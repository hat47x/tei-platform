package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/tools/internal/firstwebtargetevidence"
)

const (
	expectedBaselineID       = "B"
	expectedBaselineRevision = "e5231839a5dcc62cce1181a2731fcb7c70d16736"
	expectedTargetRevision   = "0ba87a4e0a96f70a198b03ff7652fb00e3826abe"
	expectedTargetAdapter    = "reference-first-web-target"
	expectedHost             = "go-net-http"
)

type replayPlan struct {
	EvidenceNature                     string       `json:"evidenceNature"`
	SourceExecutionIntakeSHA256        string       `json:"sourceExecutionIntakeSHA256"`
	SourceExecutionIntakeReceiptSHA256 string       `json:"sourceExecutionIntakeReceiptSHA256"`
	HandoverBaselineID                 string       `json:"handoverBaselineId"`
	BaselineRepositoryRevision         string       `json:"baselineRepositoryRevision"`
	ReplayTargetBuildRevision          string       `json:"replayTargetBuildRevision"`
	TargetAdapter                      string       `json:"targetAdapter"`
	Host                               string       `json:"host"`
	Units                              []replayUnit `json:"units"`
}

type replayUnit struct {
	BehaviorID                string `json:"behaviorId"`
	ExpectedHTTPStatus        int    `json:"expectedHTTPStatus"`
	ExpectedObservationClass  string `json:"expectedObservationClass"`
	OriginRawArtifactFile     string `json:"originRawArtifactFile"`
	OriginRawArtifactSHA256   string `json:"originRawArtifactSHA256"`
	ReplayTargetBuildRevision string `json:"replayTargetBuildRevision"`
	RequiredDistinctRuns      int    `json:"requiredDistinctRuns"`
	ReplayExecutionStatus     string `json:"replayExecutionStatus"`
	RepeatabilityStatus       string `json:"repeatabilityStatus"`
	CoverageClass             string `json:"coverageClass"`
}

type replayPlanReceipt struct {
	EvidenceNature                     string `json:"evidenceNature"`
	ReplayPlanToolRepositoryRevision   string `json:"replayPlanToolRepositoryRevision"`
	RepositoryPostRunRevision          string `json:"repositoryPostRunRevision"`
	RepositoryPostRunClean             bool   `json:"repositoryPostRunClean"`
	SourceQualificationPassed          bool   `json:"sourceQualificationPassed"`
	SourceExecutionIntakeSHA256        string `json:"sourceExecutionIntakeSHA256"`
	SourceExecutionIntakeReceiptSHA256 string `json:"sourceExecutionIntakeReceiptSHA256"`
	ReplayPlanArtifact                 string `json:"replayPlanArtifact"`
	ReplayPlanArtifactSHA256           string `json:"replayPlanArtifactSHA256"`
	ReplayExecuted                     bool   `json:"replayExecuted"`
	RepeatabilityObserved              bool   `json:"repeatabilityObserved"`
}

type runMetadata struct {
	EvidenceNature                string       `json:"evidenceNature"`
	RunIdentity                   string       `json:"runIdentity"`
	SourceReplayPlanSHA256        string       `json:"sourceReplayPlanSHA256"`
	SourceReplayPlanReceiptSHA256 string       `json:"sourceReplayPlanReceiptSHA256"`
	ReplayTargetBuildRevision     string       `json:"replayTargetBuildRevision"`
	TargetProcessInstanceIdentity string       `json:"targetProcessInstanceIdentity"`
	TargetListenScope             string       `json:"targetListenScope"`
	Captures                      []runCapture `json:"captures"`
}

type runCapture struct {
	BehaviorID         string `json:"behaviorId"`
	ObservedHTTPStatus int    `json:"observedHTTPStatus"`
	RawArtifactFile    string `json:"rawArtifactFile"`
}

type runObservation struct {
	EvidenceNature                string                `json:"evidenceNature"`
	RunIdentity                   string                `json:"runIdentity"`
	TargetProcessInstanceIdentity string                `json:"targetProcessInstanceIdentity"`
	SourceReplayPlanSHA256        string                `json:"sourceReplayPlanSHA256"`
	SourceReplayPlanReceiptSHA256 string                `json:"sourceReplayPlanReceiptSHA256"`
	HandoverBaselineID            string                `json:"handoverBaselineId"`
	BaselineRepositoryRevision    string                `json:"baselineRepositoryRevision"`
	ReplayTargetBuildRevision     string                `json:"replayTargetBuildRevision"`
	TargetAdapter                 string                `json:"targetAdapter"`
	Host                          string                `json:"host"`
	TargetListenScope             string                `json:"targetListenScope"`
	Observations                  []behaviorObservation `json:"observations"`
}

type behaviorObservation struct {
	BehaviorID               string `json:"behaviorId"`
	ObservedHTTPStatus       int    `json:"observedHTTPStatus"`
	ObservationClass         string `json:"observationClass"`
	SemanticValidationPassed bool   `json:"semanticValidationPassed"`
	RawArtifactFile          string `json:"rawArtifactFile"`
	RawArtifactSHA256        string `json:"rawArtifactSHA256"`
}

type behaviorRule struct {
	ExpectedHTTPStatus int
	ObservationClass   string
	OriginRawFile      string
	Validate           func([]byte, firstwebtargetevidence.Provenance) error
}

func main() {
	var (
		planPath             = flag.String("replay-plan", "", "R7B replay-plan JSON")
		receiptPath          = flag.String("replay-plan-receipt", "", "R7B replay-plan receipt JSON")
		metadataPath         = flag.String("run-metadata", "", "R7B run metadata JSON")
		rawDir               = flag.String("raw-dir", "", "directory containing captured raw response artifacts")
		originArtifactDir    = flag.String("origin-artifact-dir", "", "optional extracted adopted Gate C Evidence directory for origin raw identity verification")
		preflightOnly        = flag.Bool("preflight-only", false, "validate exact replay-plan / receipt binding without auditing a run")
		emitLaunchProvenance = flag.Bool("emit-launch-provenance-tsv", false, "with -preflight-only, emit validated B/R/T/adapter/host as one TSV record")
	)
	flag.Parse()
	if *planPath == "" || *receiptPath == "" {
		fatalf("-replay-plan and -replay-plan-receipt are required")
	}
	if !*preflightOnly && (*metadataPath == "" || *rawDir == "") {
		fatalf("-run-metadata and -raw-dir are required unless -preflight-only is set")
	}

	planBytes, err := os.ReadFile(*planPath)
	if err != nil {
		fatalf("read replay plan: %v", err)
	}
	receiptBytes, err := os.ReadFile(*receiptPath)
	if err != nil {
		fatalf("read replay-plan receipt: %v", err)
	}
	if *preflightOnly {
		plan, _, _, _, err := decodeAndValidateReplayInputs(planBytes, receiptBytes)
		if err != nil {
			fatalf("%v", err)
		}
		if *originArtifactDir != "" {
			if err := verifyOriginArtifacts(plan, *originArtifactDir); err != nil {
				fatalf("%v", err)
			}
		}
		if *emitLaunchProvenance {
			fmt.Fprintln(os.Stdout, launchProvenanceTSV(plan))
		} else {
			fmt.Fprintln(os.Stdout, "ok")
		}
		return
	}

	metadataBytes, err := os.ReadFile(*metadataPath)
	if err != nil {
		fatalf("read run metadata: %v", err)
	}

	observation, err := auditRun(planBytes, receiptBytes, metadataBytes, *rawDir)
	if err != nil {
		fatalf("%v", err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(observation); err != nil {
		fatalf("encode run observation: %v", err)
	}
}

func decodeAndValidateReplayInputs(planBytes, receiptBytes []byte) (replayPlan, replayPlanReceipt, string, string, error) {
	var plan replayPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		return replayPlan{}, replayPlanReceipt{}, "", "", fmt.Errorf("decode replay plan: %w", err)
	}
	var receipt replayPlanReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return replayPlan{}, replayPlanReceipt{}, "", "", fmt.Errorf("decode replay-plan receipt: %w", err)
	}
	planSHA := sha256Hex(planBytes)
	receiptSHA := sha256Hex(receiptBytes)
	if err := validatePlan(plan); err != nil {
		return replayPlan{}, replayPlanReceipt{}, "", "", err
	}
	if err := validateReceipt(receipt, plan, planSHA); err != nil {
		return replayPlan{}, replayPlanReceipt{}, "", "", err
	}
	return plan, receipt, planSHA, receiptSHA, nil
}

func auditRun(planBytes, receiptBytes, metadataBytes []byte, rawDir string) (runObservation, error) {
	plan, _, planSHA, receiptSHA, err := decodeAndValidateReplayInputs(planBytes, receiptBytes)
	if err != nil {
		return runObservation{}, err
	}
	var metadata runMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return runObservation{}, fmt.Errorf("decode run metadata: %w", err)
	}
	if err := validateMetadata(metadata, plan, planSHA, receiptSHA); err != nil {
		return runObservation{}, err
	}

	units := make(map[string]replayUnit, len(plan.Units))
	for _, unit := range plan.Units {
		units[unit.BehaviorID] = unit
	}

	expectedProvenance := firstwebtargetevidence.Provenance{
		HandoverBaselineID:         plan.HandoverBaselineID,
		BaselineRepositoryRevision: plan.BaselineRepositoryRevision,
		TargetBuildRevision:        plan.ReplayTargetBuildRevision,
		TargetAdapter:              plan.TargetAdapter,
		Host:                       plan.Host,
	}

	captures := append([]runCapture(nil), metadata.Captures...)
	sort.Slice(captures, func(i, j int) bool { return captures[i].BehaviorID < captures[j].BehaviorID })
	observations := make([]behaviorObservation, 0, len(captures))
	for _, capture := range captures {
		unit := units[capture.BehaviorID]
		rule := behaviorRules()[capture.BehaviorID]
		if capture.ObservedHTTPStatus != unit.ExpectedHTTPStatus {
			return runObservation{}, fmt.Errorf("behavior %q HTTP status=%d, want %d", capture.BehaviorID, capture.ObservedHTTPStatus, unit.ExpectedHTTPStatus)
		}
		path := filepath.Join(rawDir, capture.RawArtifactFile)
		info, err := os.Lstat(path)
		if err != nil {
			return runObservation{}, fmt.Errorf("read raw artifact %q: %w", capture.RawArtifactFile, err)
		}
		if !info.Mode().IsRegular() {
			return runObservation{}, fmt.Errorf("raw artifact %q is not a regular file", capture.RawArtifactFile)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return runObservation{}, fmt.Errorf("read raw artifact %q: %w", capture.RawArtifactFile, err)
		}
		if err := firstwebtargetevidence.RejectReferenceCredentialMaterial(capture.RawArtifactFile, data); err != nil {
			return runObservation{}, err
		}
		if err := rule.Validate(data, expectedProvenance); err != nil {
			return runObservation{}, fmt.Errorf("behavior %q semantic validation: %w", capture.BehaviorID, err)
		}
		if unit.ExpectedObservationClass != rule.ObservationClass {
			return runObservation{}, fmt.Errorf("behavior %q Observation class=%q, shared validator class=%q", capture.BehaviorID, unit.ExpectedObservationClass, rule.ObservationClass)
		}
		observations = append(observations, behaviorObservation{
			BehaviorID:               capture.BehaviorID,
			ObservedHTTPStatus:       capture.ObservedHTTPStatus,
			ObservationClass:         rule.ObservationClass,
			SemanticValidationPassed: true,
			RawArtifactFile:          capture.RawArtifactFile,
			RawArtifactSHA256:        sha256Hex(data),
		})
	}

	return runObservation{
		EvidenceNature:                "first-web-target-r7b-run-observation",
		RunIdentity:                   metadata.RunIdentity,
		TargetProcessInstanceIdentity: metadata.TargetProcessInstanceIdentity,
		SourceReplayPlanSHA256:        planSHA,
		SourceReplayPlanReceiptSHA256: receiptSHA,
		HandoverBaselineID:            plan.HandoverBaselineID,
		BaselineRepositoryRevision:    plan.BaselineRepositoryRevision,
		ReplayTargetBuildRevision:     plan.ReplayTargetBuildRevision,
		TargetAdapter:                 plan.TargetAdapter,
		Host:                          plan.Host,
		TargetListenScope:             metadata.TargetListenScope,
		Observations:                  observations,
	}, nil
}

func validatePlan(plan replayPlan) error {
	if plan.EvidenceNature != "first-web-target-r7b-replay-plan" {
		return fmt.Errorf("unexpected replay plan evidenceNature %q", plan.EvidenceNature)
	}
	if !isSHA256(plan.SourceExecutionIntakeSHA256) || !isSHA256(plan.SourceExecutionIntakeReceiptSHA256) {
		return fmt.Errorf("replay plan source identities are invalid")
	}
	if plan.HandoverBaselineID != expectedBaselineID ||
		plan.BaselineRepositoryRevision != expectedBaselineRevision ||
		plan.ReplayTargetBuildRevision != expectedTargetRevision ||
		plan.TargetAdapter != expectedTargetAdapter ||
		plan.Host != expectedHost {
		return fmt.Errorf("replay plan launch provenance does not match the adopted Gate C Target")
	}

	rules := behaviorRules()
	if len(plan.Units) != len(rules) {
		return fmt.Errorf("replay plan behavior count=%d, want %d", len(plan.Units), len(rules))
	}
	seen := map[string]struct{}{}
	for _, unit := range plan.Units {
		rule, ok := rules[unit.BehaviorID]
		if !ok {
			return fmt.Errorf("unknown replay-plan behavior %q", unit.BehaviorID)
		}
		if _, duplicate := seen[unit.BehaviorID]; duplicate {
			return fmt.Errorf("duplicate replay-plan behavior %q", unit.BehaviorID)
		}
		seen[unit.BehaviorID] = struct{}{}
		if unit.ReplayTargetBuildRevision != plan.ReplayTargetBuildRevision {
			return fmt.Errorf("behavior %q Target revision drift", unit.BehaviorID)
		}
		if unit.ExpectedHTTPStatus != rule.ExpectedHTTPStatus {
			return fmt.Errorf("behavior %q expected HTTP status=%d, want %d", unit.BehaviorID, unit.ExpectedHTTPStatus, rule.ExpectedHTTPStatus)
		}
		if unit.ExpectedObservationClass != rule.ObservationClass {
			return fmt.Errorf("behavior %q expected Observation class=%q, want %q", unit.BehaviorID, unit.ExpectedObservationClass, rule.ObservationClass)
		}
		if unit.OriginRawArtifactFile != rule.OriginRawFile || !safeArtifactName(unit.OriginRawArtifactFile) || !isSHA256(unit.OriginRawArtifactSHA256) {
			return fmt.Errorf("behavior %q origin raw artifact identity is invalid", unit.BehaviorID)
		}
		if unit.RequiredDistinctRuns != 2 || unit.ReplayExecutionStatus != "not-executed" || unit.RepeatabilityStatus != "not-observed" || unit.CoverageClass != "behavior-unit" {
			return fmt.Errorf("behavior %q replay-plan state is not the unexecuted behavior-unit contract", unit.BehaviorID)
		}
	}
	return nil
}

func validateReceipt(receipt replayPlanReceipt, plan replayPlan, planSHA string) error {
	if receipt.EvidenceNature != "first-web-target-r7b-replay-plan-acquisition-receipt" {
		return fmt.Errorf("unexpected replay-plan receipt evidenceNature %q", receipt.EvidenceNature)
	}
	if !isGitRevision(receipt.ReplayPlanToolRepositoryRevision) ||
		receipt.RepositoryPostRunRevision != receipt.ReplayPlanToolRepositoryRevision ||
		!receipt.RepositoryPostRunClean || !receipt.SourceQualificationPassed {
		return fmt.Errorf("replay-plan receipt does not prove a clean qualified acquisition")
	}
	if receipt.ReplayPlanArtifact != "r7b-replay-plan.json" || receipt.ReplayPlanArtifactSHA256 != planSHA {
		return fmt.Errorf("replay-plan receipt is not bound to the exact replay-plan bytes")
	}
	if receipt.SourceExecutionIntakeSHA256 != plan.SourceExecutionIntakeSHA256 || receipt.SourceExecutionIntakeReceiptSHA256 != plan.SourceExecutionIntakeReceiptSHA256 {
		return fmt.Errorf("replay-plan receipt source identities do not match the replay plan")
	}
	if receipt.ReplayExecuted || receipt.RepeatabilityObserved {
		return fmt.Errorf("replay-plan receipt already claims replay execution or repeatability")
	}
	return nil
}

func validateMetadata(metadata runMetadata, plan replayPlan, planSHA, receiptSHA string) error {
	if metadata.EvidenceNature != "first-web-target-r7b-run-metadata" {
		return fmt.Errorf("unexpected run metadata evidenceNature %q", metadata.EvidenceNature)
	}
	if strings.TrimSpace(metadata.RunIdentity) == "" || strings.TrimSpace(metadata.TargetProcessInstanceIdentity) == "" {
		return fmt.Errorf("run identity and Target process instance identity are required")
	}
	if metadata.SourceReplayPlanSHA256 != planSHA || metadata.SourceReplayPlanReceiptSHA256 != receiptSHA {
		return fmt.Errorf("run metadata is not bound to the exact replay plan / receipt bytes")
	}
	if metadata.ReplayTargetBuildRevision != plan.ReplayTargetBuildRevision {
		return fmt.Errorf("run metadata Target revision=%q, want %q", metadata.ReplayTargetBuildRevision, plan.ReplayTargetBuildRevision)
	}
	if metadata.TargetListenScope != "loopback" {
		return fmt.Errorf("Target listen scope=%q, want loopback", metadata.TargetListenScope)
	}
	if len(metadata.Captures) != len(plan.Units) {
		return fmt.Errorf("run capture count=%d, want %d", len(metadata.Captures), len(plan.Units))
	}

	units := make(map[string]struct{}, len(plan.Units))
	for _, unit := range plan.Units {
		units[unit.BehaviorID] = struct{}{}
	}
	seenBehavior := map[string]struct{}{}
	seenFile := map[string]struct{}{}
	for _, capture := range metadata.Captures {
		if _, ok := units[capture.BehaviorID]; !ok {
			return fmt.Errorf("unknown run capture behavior %q", capture.BehaviorID)
		}
		if _, duplicate := seenBehavior[capture.BehaviorID]; duplicate {
			return fmt.Errorf("duplicate run capture behavior %q", capture.BehaviorID)
		}
		seenBehavior[capture.BehaviorID] = struct{}{}
		if !safeArtifactName(capture.RawArtifactFile) {
			return fmt.Errorf("unsafe raw artifact filename %q", capture.RawArtifactFile)
		}
		if _, duplicate := seenFile[capture.RawArtifactFile]; duplicate {
			return fmt.Errorf("duplicate raw artifact mapping %q", capture.RawArtifactFile)
		}
		seenFile[capture.RawArtifactFile] = struct{}{}
	}
	return nil
}

func launchProvenanceTSV(plan replayPlan) string {
	return strings.Join([]string{
		plan.HandoverBaselineID,
		plan.BaselineRepositoryRevision,
		plan.ReplayTargetBuildRevision,
		plan.TargetAdapter,
		plan.Host,
	}, "\t")
}

func verifyOriginArtifacts(plan replayPlan, dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat origin artifact directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("origin artifact path is not a directory")
	}
	seen := make(map[string]struct{}, len(plan.Units))
	for _, unit := range plan.Units {
		if _, duplicate := seen[unit.OriginRawArtifactFile]; duplicate {
			return fmt.Errorf("duplicate origin raw artifact file %q", unit.OriginRawArtifactFile)
		}
		seen[unit.OriginRawArtifactFile] = struct{}{}
		path := filepath.Join(dir, unit.OriginRawArtifactFile)
		fileInfo, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("read origin raw artifact %q: %w", unit.OriginRawArtifactFile, err)
		}
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("origin raw artifact %q is not a regular file", unit.OriginRawArtifactFile)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read origin raw artifact %q: %w", unit.OriginRawArtifactFile, err)
		}
		if err := firstwebtargetevidence.RejectReferenceCredentialMaterial(unit.OriginRawArtifactFile, data); err != nil {
			return err
		}
		if got := sha256Hex(data); got != unit.OriginRawArtifactSHA256 {
			return fmt.Errorf("origin raw artifact %q SHA-256=%s, want %s", unit.OriginRawArtifactFile, got, unit.OriginRawArtifactSHA256)
		}
	}
	return nil
}

func behaviorRules() map[string]behaviorRule {
	return map[string]behaviorRule{
		"first-web/read-only": {
			ExpectedHTTPStatus: 200,
			ObservationClass:   firstwebtargetevidence.ObservationClassReadResult,
			OriginRawFile:      "read.json",
			Validate:           firstwebtargetevidence.ValidateReadArtifact,
		},
		"first-web/command-success": {
			ExpectedHTTPStatus: 200,
			ObservationClass:   firstwebtargetevidence.ObservationClassBusinessResultPlusObservation,
			OriginRawFile:      "submit-success.json",
			Validate:           firstwebtargetevidence.ValidateCommandSuccessArtifact,
		},
		"first-web/authentication-denied": {
			ExpectedHTTPStatus: 200,
			ObservationClass:   firstwebtargetevidence.ObservationClassAuthenticationDenied,
			OriginRawFile:      "submit-authentication-denied.json",
			Validate:           firstwebtargetevidence.ValidateAuthenticationDeniedArtifact,
		},
		"first-web/authorization-denied": {
			ExpectedHTTPStatus: 200,
			ObservationClass:   firstwebtargetevidence.ObservationClassAuthorizationDenied,
			OriginRawFile:      "submit-authorization-denied.json",
			Validate:           firstwebtargetevidence.ValidateAuthorizationDeniedArtifact,
		},
		"first-web/validation-not-satisfied": {
			ExpectedHTTPStatus: 200,
			ObservationClass:   firstwebtargetevidence.ObservationClassValidationNotSatisfied,
			OriginRawFile:      "submit-validation-not-satisfied.json",
			Validate:           firstwebtargetevidence.ValidateValidationNotSatisfiedArtifact,
		},
		"first-web/execution-error": {
			ExpectedHTTPStatus: 422,
			ObservationClass:   firstwebtargetevidence.ObservationClassExecutionError,
			OriginRawFile:      "submit-execution-error.json",
			Validate:           firstwebtargetevidence.ValidateExecutionErrorArtifact,
		},
	}
}

func safeArtifactName(name string) bool {
	if name == "" || filepath.IsAbs(name) || filepath.Clean(name) != name || filepath.Base(name) != name || strings.Contains(name, `\`) {
		return false
	}
	return name != "." && name != ".."
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
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
