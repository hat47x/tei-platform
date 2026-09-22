package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type stringListFlag []string

func (f *stringListFlag) String() string { return strings.Join(*f, ",") }
func (f *stringListFlag) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("run-observation path is empty")
	}
	*f = append(*f, value)
	return nil
}

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

type runIdentityRecord struct {
	RunIdentity                   string `json:"runIdentity"`
	TargetProcessInstanceIdentity string `json:"targetProcessInstanceIdentity"`
	RunObservationSHA256          string `json:"runObservationSHA256"`
}

type behaviorRepeatability struct {
	BehaviorID   string `json:"behaviorId"`
	DistinctRuns int    `json:"distinctRuns"`
	Repeatable   bool   `json:"repeatable"`
}

type repeatabilitySummary struct {
	EvidenceNature                string                  `json:"evidenceNature"`
	SourceReplayPlanSHA256        string                  `json:"sourceReplayPlanSHA256"`
	SourceReplayPlanReceiptSHA256 string                  `json:"sourceReplayPlanReceiptSHA256"`
	HandoverBaselineID            string                  `json:"handoverBaselineId"`
	BaselineRepositoryRevision    string                  `json:"baselineRepositoryRevision"`
	ReplayTargetBuildRevision     string                  `json:"replayTargetBuildRevision"`
	TargetAdapter                 string                  `json:"targetAdapter"`
	Host                          string                  `json:"host"`
	RequiredDistinctRuns          int                     `json:"requiredDistinctRuns"`
	ObservedDistinctRuns          int                     `json:"observedDistinctRuns"`
	Runs                          []runIdentityRecord     `json:"runs"`
	Behaviors                     []behaviorRepeatability `json:"behaviors"`
	RepeatabilityObserved         bool                    `json:"repeatabilityObserved"`
	CoverageClass                 string                  `json:"coverageClass"`
	DeepWorkflowCovered           bool                    `json:"deepWorkflowCovered"`
	AuthorityLimits               []string                `json:"authorityLimits"`
}

func main() {
	var observations stringListFlag
	planPath := flag.String("replay-plan", "", "R7B replay-plan JSON")
	receiptPath := flag.String("replay-plan-receipt", "", "R7B replay-plan receipt JSON")
	flag.Var(&observations, "run-observation", "audited R7B run-observation JSON (repeat at least twice)")
	flag.Parse()
	if *planPath == "" || *receiptPath == "" || len(observations) < 2 {
		fatalf("-replay-plan, -replay-plan-receipt, and at least two -run-observation values are required")
	}

	planBytes, err := os.ReadFile(*planPath)
	if err != nil {
		fatalf("read replay plan: %v", err)
	}
	receiptBytes, err := os.ReadFile(*receiptPath)
	if err != nil {
		fatalf("read replay-plan receipt: %v", err)
	}
	runBytes := make([][]byte, 0, len(observations))
	for _, path := range observations {
		data, err := os.ReadFile(path)
		if err != nil {
			fatalf("read run observation %s: %v", path, err)
		}
		runBytes = append(runBytes, data)
	}

	summary, err := aggregateRepeatability(planBytes, receiptBytes, runBytes)
	if err != nil {
		fatalf("%v", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		fatalf("encode repeatability summary: %v", err)
	}
}

func aggregateRepeatability(planBytes, receiptBytes []byte, runBytes [][]byte) (repeatabilitySummary, error) {
	var plan replayPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		return repeatabilitySummary{}, fmt.Errorf("decode replay plan: %w", err)
	}
	var receipt replayPlanReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return repeatabilitySummary{}, fmt.Errorf("decode replay-plan receipt: %w", err)
	}

	planSHA := sha256Hex(planBytes)
	receiptSHA := sha256Hex(receiptBytes)
	units, required, err := validatePlan(plan)
	if err != nil {
		return repeatabilitySummary{}, err
	}
	if err := validateReceipt(receipt, plan, planSHA); err != nil {
		return repeatabilitySummary{}, err
	}
	if len(runBytes) < required {
		return repeatabilitySummary{}, fmt.Errorf("repeatability requires at least %d audited runs, got %d", required, len(runBytes))
	}

	runIDs := map[string]struct{}{}
	processIDs := map[string]struct{}{}
	perBehavior := make(map[string]int, len(units))
	records := make([]runIdentityRecord, 0, len(runBytes))

	for index, data := range runBytes {
		var run runObservation
		if err := json.Unmarshal(data, &run); err != nil {
			return repeatabilitySummary{}, fmt.Errorf("decode run observation %d: %w", index+1, err)
		}
		if err := validateRun(run, plan, units, planSHA, receiptSHA); err != nil {
			return repeatabilitySummary{}, fmt.Errorf("run observation %d: %w", index+1, err)
		}
		if _, duplicate := runIDs[run.RunIdentity]; duplicate {
			return repeatabilitySummary{}, fmt.Errorf("duplicate run identity %q", run.RunIdentity)
		}
		runIDs[run.RunIdentity] = struct{}{}
		if _, duplicate := processIDs[run.TargetProcessInstanceIdentity]; duplicate {
			return repeatabilitySummary{}, fmt.Errorf("duplicate Target process instance identity %q", run.TargetProcessInstanceIdentity)
		}
		processIDs[run.TargetProcessInstanceIdentity] = struct{}{}

		for _, obs := range run.Observations {
			perBehavior[obs.BehaviorID]++
		}
		records = append(records, runIdentityRecord{
			RunIdentity:                   run.RunIdentity,
			TargetProcessInstanceIdentity: run.TargetProcessInstanceIdentity,
			RunObservationSHA256:          sha256Hex(data),
		})
	}

	sort.Slice(records, func(i, j int) bool { return records[i].RunIdentity < records[j].RunIdentity })
	behaviorIDs := make([]string, 0, len(units))
	for id := range units {
		behaviorIDs = append(behaviorIDs, id)
	}
	sort.Strings(behaviorIDs)
	behaviors := make([]behaviorRepeatability, 0, len(behaviorIDs))
	allRepeatable := true
	for _, id := range behaviorIDs {
		count := perBehavior[id]
		repeatable := count >= units[id].RequiredDistinctRuns
		if !repeatable {
			allRepeatable = false
		}
		behaviors = append(behaviors, behaviorRepeatability{BehaviorID: id, DistinctRuns: count, Repeatable: repeatable})
	}

	return repeatabilitySummary{
		EvidenceNature:                "first-web-target-r7b-repeatability-summary",
		SourceReplayPlanSHA256:        planSHA,
		SourceReplayPlanReceiptSHA256: receiptSHA,
		HandoverBaselineID:            plan.HandoverBaselineID,
		BaselineRepositoryRevision:    plan.BaselineRepositoryRevision,
		ReplayTargetBuildRevision:     plan.ReplayTargetBuildRevision,
		TargetAdapter:                 plan.TargetAdapter,
		Host:                          plan.Host,
		RequiredDistinctRuns:          required,
		ObservedDistinctRuns:          len(runIDs),
		Runs:                          records,
		Behaviors:                     behaviors,
		RepeatabilityObserved:         allRepeatable,
		CoverageClass:                 "behavior-unit",
		DeepWorkflowCovered:           false,
		AuthorityLimits: []string{
			"repeatability-summary-does-not-reclassify-raw-artifacts",
			"raw-artifact-byte-equality-is-not-required-for-repeatability",
			"behavior-unit-repeatability-does-not-imply-workflow-coverage",
			"repeatability-does-not-grant-execution-authority",
		},
	}, nil
}

func validatePlan(plan replayPlan) (map[string]replayUnit, int, error) {
	if plan.EvidenceNature != "first-web-target-r7b-replay-plan" {
		return nil, 0, fmt.Errorf("unexpected replay plan evidenceNature %q", plan.EvidenceNature)
	}
	if !isSHA256(plan.SourceExecutionIntakeSHA256) || !isSHA256(plan.SourceExecutionIntakeReceiptSHA256) {
		return nil, 0, fmt.Errorf("replay plan source identities are invalid")
	}
	if strings.TrimSpace(plan.HandoverBaselineID) == "" || !isGitRevision(plan.BaselineRepositoryRevision) || !isGitRevision(plan.ReplayTargetBuildRevision) || strings.TrimSpace(plan.TargetAdapter) == "" || strings.TrimSpace(plan.Host) == "" {
		return nil, 0, fmt.Errorf("replay plan launch provenance is incomplete")
	}
	type expectedBehavior struct {
		HTTPStatus       int
		ObservationClass string
	}
	expectedBehaviors := map[string]expectedBehavior{
		"first-web/read-only": {
			HTTPStatus:       200,
			ObservationClass: "read-result",
		},
		"first-web/command-success": {
			HTTPStatus:       200,
			ObservationClass: "business-result-plus-separate-observation",
		},
		"first-web/authentication-denied": {
			HTTPStatus:       200,
			ObservationClass: "business-denial-authentication",
		},
		"first-web/authorization-denied": {
			HTTPStatus:       200,
			ObservationClass: "business-denial-authorization",
		},
		"first-web/validation-not-satisfied": {
			HTTPStatus:       200,
			ObservationClass: "validation-not-satisfied",
		},
		"first-web/execution-error": {
			HTTPStatus:       422,
			ObservationClass: "command-invocation-execution-error",
		},
	}
	if len(plan.Units) != len(expectedBehaviors) {
		return nil, 0, fmt.Errorf("replay plan behavior count=%d, want %d", len(plan.Units), len(expectedBehaviors))
	}
	units := make(map[string]replayUnit, len(plan.Units))
	required := 0
	for _, unit := range plan.Units {
		if strings.TrimSpace(unit.BehaviorID) == "" {
			return nil, 0, fmt.Errorf("empty replay-plan behavior identity")
		}
		expected, ok := expectedBehaviors[unit.BehaviorID]
		if !ok {
			return nil, 0, fmt.Errorf("unknown replay-plan behavior %q", unit.BehaviorID)
		}
		if _, dup := units[unit.BehaviorID]; dup {
			return nil, 0, fmt.Errorf("duplicate replay-plan behavior %q", unit.BehaviorID)
		}
		if unit.ReplayTargetBuildRevision != plan.ReplayTargetBuildRevision {
			return nil, 0, fmt.Errorf("behavior %q Target revision drift", unit.BehaviorID)
		}
		if unit.RequiredDistinctRuns != 2 {
			return nil, 0, fmt.Errorf("behavior %q distinct-run requirement=%d, want 2", unit.BehaviorID, unit.RequiredDistinctRuns)
		}
		if required == 0 {
			required = unit.RequiredDistinctRuns
		} else if required != unit.RequiredDistinctRuns {
			return nil, 0, fmt.Errorf("behavior %q distinct-run requirement differs", unit.BehaviorID)
		}
		if unit.ExpectedHTTPStatus != expected.HTTPStatus {
			return nil, 0, fmt.Errorf("behavior %q expected HTTP status=%d, want %d", unit.BehaviorID, unit.ExpectedHTTPStatus, expected.HTTPStatus)
		}
		if unit.ExpectedObservationClass != expected.ObservationClass {
			return nil, 0, fmt.Errorf("behavior %q expected Observation class=%q, want %q", unit.BehaviorID, unit.ExpectedObservationClass, expected.ObservationClass)
		}
		if unit.ReplayExecutionStatus != "not-executed" || unit.RepeatabilityStatus != "not-observed" || unit.CoverageClass != "behavior-unit" {
			return nil, 0, fmt.Errorf("behavior %q replay-plan state is not the unexecuted behavior-unit contract", unit.BehaviorID)
		}
		units[unit.BehaviorID] = unit
	}
	return units, required, nil
}

func validateReceipt(receipt replayPlanReceipt, plan replayPlan, planSHA string) error {
	if receipt.EvidenceNature != "first-web-target-r7b-replay-plan-acquisition-receipt" {
		return fmt.Errorf("unexpected replay-plan receipt evidenceNature %q", receipt.EvidenceNature)
	}
	if !isGitRevision(receipt.ReplayPlanToolRepositoryRevision) || receipt.RepositoryPostRunRevision != receipt.ReplayPlanToolRepositoryRevision || !receipt.RepositoryPostRunClean || !receipt.SourceQualificationPassed {
		return fmt.Errorf("replay-plan receipt does not prove a clean qualified acquisition")
	}
	if receipt.ReplayPlanArtifact != "r7b-replay-plan.json" || !isSHA256(receipt.ReplayPlanArtifactSHA256) || receipt.ReplayPlanArtifactSHA256 != planSHA {
		return fmt.Errorf("replay-plan receipt is not bound to the exact replay-plan bytes")
	}
	if receipt.SourceExecutionIntakeSHA256 != plan.SourceExecutionIntakeSHA256 || receipt.SourceExecutionIntakeReceiptSHA256 != plan.SourceExecutionIntakeReceiptSHA256 {
		return fmt.Errorf("replay-plan receipt source identities do not match replay plan")
	}
	if receipt.ReplayExecuted || receipt.RepeatabilityObserved {
		return fmt.Errorf("replay-plan receipt already claims execution or repeatability")
	}
	return nil
}

func validateRun(run runObservation, plan replayPlan, units map[string]replayUnit, planSHA, receiptSHA string) error {
	if run.EvidenceNature != "first-web-target-r7b-run-observation" {
		return fmt.Errorf("unexpected run observation evidenceNature %q", run.EvidenceNature)
	}
	if strings.TrimSpace(run.RunIdentity) == "" || strings.TrimSpace(run.TargetProcessInstanceIdentity) == "" {
		return fmt.Errorf("run and Target process identities are required")
	}
	if run.SourceReplayPlanSHA256 != planSHA || run.SourceReplayPlanReceiptSHA256 != receiptSHA {
		return fmt.Errorf("run observation is not bound to exact replay plan / receipt")
	}
	if run.HandoverBaselineID != plan.HandoverBaselineID || run.BaselineRepositoryRevision != plan.BaselineRepositoryRevision || run.ReplayTargetBuildRevision != plan.ReplayTargetBuildRevision || run.TargetAdapter != plan.TargetAdapter || run.Host != plan.Host {
		return fmt.Errorf("run observation launch provenance drift")
	}
	if run.TargetListenScope != "loopback" {
		return fmt.Errorf("run Target listen scope=%q, want loopback", run.TargetListenScope)
	}
	if len(run.Observations) != len(units) {
		return fmt.Errorf("run observation behavior count=%d, want %d", len(run.Observations), len(units))
	}
	seen := map[string]struct{}{}
	for _, obs := range run.Observations {
		unit, ok := units[obs.BehaviorID]
		if !ok {
			return fmt.Errorf("unknown run behavior %q", obs.BehaviorID)
		}
		if _, dup := seen[obs.BehaviorID]; dup {
			return fmt.Errorf("duplicate run behavior %q", obs.BehaviorID)
		}
		seen[obs.BehaviorID] = struct{}{}
		if obs.ObservedHTTPStatus != unit.ExpectedHTTPStatus {
			return fmt.Errorf("behavior %q HTTP status=%d, want %d", obs.BehaviorID, obs.ObservedHTTPStatus, unit.ExpectedHTTPStatus)
		}
		if !obs.SemanticValidationPassed || obs.ObservationClass != unit.ExpectedObservationClass {
			return fmt.Errorf("behavior %q semantic observation does not match plan", obs.BehaviorID)
		}
		if strings.TrimSpace(obs.RawArtifactFile) == "" || !isSHA256(obs.RawArtifactSHA256) {
			return fmt.Errorf("behavior %q raw artifact identity is incomplete", obs.BehaviorID)
		}
	}
	return nil
}

func sha256Hex(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
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
func fatalf(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...); os.Exit(1) }
