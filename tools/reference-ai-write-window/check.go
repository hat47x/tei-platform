// Package aiwritewindow is a read-only reference for #2092.
//
// It checks an AI edit against a plugin-declared write window: paths,
// expressions, value types, change kinds, and effects. Edits outside the window
// are rejected before acceptance. Unknown dependencies and effects stay
// unresolved; a change inside the window that alters meaning or authority goes
// back to a human decision. Accepting a candidate is never business execution
// permission, and a self-reported pass is not evidence.
package aiwritewindow

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

const (
	InputSchema  = "tei.reference.ai-write-window-input/v0"
	ReportSchema = "tei.reference.ai-write-window-report/v0"

	ModeWindow          = "window"
	ModeWholeGeneration = "whole-generation"

	StatusCandidateAccepted   = "candidate-accepted"
	StatusHumanDecisionNeeded = "human-decision-required"
	StatusUnresolved          = "unresolved"
	StatusBlocked             = "blocked"

	TargetVerifiedCandidate = "verified-candidate"
	TargetNotApplicable     = "not-applicable"
	TargetUnresolved        = "unresolved"
)

// Window is the plugin-declared region an AI may edit. DeclaredBy must be
// "human"; an AI cannot create or widen its own window.
type Window struct {
	ID                 string   `json:"id"`
	DeclaredBy         string   `json:"declaredBy"`
	SourceRevision     string   `json:"sourceRevision"`
	ProviderRevision   string   `json:"providerRevision"`
	AllowedPaths       []string `json:"allowedPaths"`
	AllowedExpressions []string `json:"allowedExpressions"`
	AllowedTypes       []string `json:"allowedTypes"`
	AllowedEffects     []string `json:"allowedEffects"`
	AllowedKinds       []string `json:"allowedKinds"`
}

// Change is one edit operation. Kind is "expression", "datatype", "authority",
// or "destructive".
type Change struct {
	Path        string   `json:"path"`
	Kind        string   `json:"kind"`
	Expressions []string `json:"expressions,omitempty"`
	Types       []string `json:"types,omitempty"`
	Effects     []string `json:"effects,omitempty"`
}

// Helper is a function the edited expression calls. An unresolved helper keeps
// its effects unknown.
type Helper struct {
	Name     string   `json:"name"`
	Effects  []string `json:"effects,omitempty"`
	Resolved bool     `json:"resolved"`
}

// TestEvidence is one executed check. A self-reported pass does not replace it.
type TestEvidence struct {
	Command    string   `json:"command"`
	Head       string   `json:"head"`
	ExitCode   int      `json:"exitCode"`
	Unverified []string `json:"unverified,omitempty"`
}

// Edit is the proposed change.
type Edit struct {
	ID                   string         `json:"id"`
	Author               string         `json:"author"`
	BaseSourceRevision   string         `json:"baseSourceRevision"`
	BaseProviderRevision string         `json:"baseProviderRevision"`
	Changes              []Change       `json:"changes"`
	SemanticChange       bool           `json:"semanticChange"`
	AuthorityChange      bool           `json:"authorityChange"`
	Helpers              []Helper       `json:"helpers,omitempty"`
	Tests                []TestEvidence `json:"tests,omitempty"`
	SelfReportedPass     bool           `json:"selfReportedPass,omitempty"`
	IsolatedBranch       bool           `json:"isolatedBranch,omitempty"`
	RoundTripEqual       bool           `json:"roundTripEqual,omitempty"`
	HumanApproval        bool           `json:"humanApproval,omitempty"`
	AiWritesToProduction bool           `json:"aiWritesToProduction,omitempty"`
	DestructiveApproval  bool           `json:"destructiveApproval,omitempty"`
	PreChangeSnapshot    bool           `json:"preChangeSnapshot,omitempty"`
}

// Input is the model the check reads.
type Input struct {
	Schema             string              `json:"schema"`
	Mode               string              `json:"mode"`
	Window             Window              `json:"window"`
	Edit               Edit                `json:"edit"`
	Targets            []string            `json:"targets"`
	TargetCapabilities map[string][]string `json:"targetCapabilities"`
}

// TargetResult is the candidate status for one realization target. It is not a
// business execution permission.
type TargetResult struct {
	Target  string   `json:"target"`
	Status  string   `json:"status"`
	Reasons []string `json:"reasons,omitempty"`
}

// Report is the check result.
type Report struct {
	Schema                   string         `json:"schema"`
	Status                   string         `json:"status"`
	Mode                     string         `json:"mode"`
	Violations               []string       `json:"violations,omitempty"`
	HumanDecisions           []string       `json:"humanDecisions,omitempty"`
	Unresolved               []string       `json:"unresolved,omitempty"`
	Stale                    []string       `json:"stale,omitempty"`
	Targets                  []TargetResult `json:"targets"`
	SelfReportUsed           bool           `json:"selfReportUsed"`
	BusinessExecutionAllowed bool           `json:"businessExecutionAllowed"`
	Note                     string         `json:"note"`
}

// LoadInput reads an input file.
func LoadInput(path string) (Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Input{}, err
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return Input{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return input, nil
}

// Check compares the edit with the declared window and the selected targets.
func Check(input Input) Report {
	report := Report{
		Schema:                   ReportSchema,
		Mode:                     input.Mode,
		BusinessExecutionAllowed: false,
		Note:                     "受理候補は業務実行の許可ではありません。AIの自己申告は検証の証拠にしません。",
	}

	var violations []string
	var stale []string
	var unresolved []string
	var humanDecisions []string

	if input.Schema != InputSchema {
		unresolved = append(unresolved, fmt.Sprintf("schema is %q, want %q", input.Schema, InputSchema))
	}

	switch input.Mode {
	case ModeWindow:
		checkWindow(input, &violations, &stale, &unresolved, &humanDecisions)
	case ModeWholeGeneration:
		checkWholeGeneration(input.Edit, &violations, &unresolved)
	default:
		unresolved = append(unresolved, fmt.Sprintf("unknown mode %q", input.Mode))
	}

	if input.Edit.Author != "ai" && input.Edit.Author != "human" {
		unresolved = append(unresolved, fmt.Sprintf("unknown edit author %q", input.Edit.Author))
	}
	if input.Edit.SelfReportedPass && len(input.Edit.Tests) == 0 {
		unresolved = append(unresolved, "self-report-is-not-evidence")
	}
	for _, test := range input.Edit.Tests {
		if test.Command == "" || test.Head == "" {
			unresolved = append(unresolved, "test-evidence-incomplete")
		}
	}

	report.Targets = checkTargets(input)
	for _, target := range report.Targets {
		if target.Status == TargetUnresolved {
			unresolved = append(unresolved, "target-unknown:"+target.Target)
		}
	}
	if len(input.Targets) == 0 {
		unresolved = append(unresolved, "target-unknown")
	}

	report.Violations = uniqueSorted(violations)
	report.Stale = uniqueSorted(stale)
	report.Unresolved = uniqueSorted(unresolved)
	report.HumanDecisions = uniqueSorted(humanDecisions)
	report.SelfReportUsed = false

	switch {
	case len(report.Violations) > 0 || len(report.Stale) > 0:
		report.Status = StatusBlocked
	case len(report.Unresolved) > 0:
		report.Status = StatusUnresolved
	case len(report.HumanDecisions) > 0:
		report.Status = StatusHumanDecisionNeeded
	default:
		report.Status = StatusCandidateAccepted
	}
	return report
}

func checkWindow(input Input, violations, stale, unresolved, humanDecisions *[]string) {
	window := input.Window
	edit := input.Edit
	if window.DeclaredBy != "human" {
		*violations = append(*violations, "window-must-be-declared-by-human")
	}
	if edit.BaseSourceRevision != window.SourceRevision {
		*stale = append(*stale, "source-revision-changed")
	}
	if edit.BaseProviderRevision != window.ProviderRevision {
		*stale = append(*stale, "provider-revision-changed")
	}

	for _, change := range edit.Changes {
		if !contains(window.AllowedPaths, change.Path) {
			*violations = append(*violations, "path-outside-window:"+change.Path)
			continue
		}
		if !contains(window.AllowedKinds, change.Kind) {
			*violations = append(*violations, "change-kind-outside-window:"+change.Kind)
		}
		for _, expression := range change.Expressions {
			if !contains(window.AllowedExpressions, expression) {
				*violations = append(*violations, "expression-outside-window:"+expression)
			}
		}
		for _, valueType := range change.Types {
			if !contains(window.AllowedTypes, valueType) {
				*violations = append(*violations, "type-outside-window:"+valueType)
			}
		}
		for _, effect := range change.Effects {
			if !contains(window.AllowedEffects, effect) {
				*violations = append(*violations, "effect-outside-window:"+effect)
			}
		}
		if change.Kind == "destructive" {
			if !edit.DestructiveApproval {
				*violations = append(*violations, "destructive-requires-human-approval")
			}
			if !edit.PreChangeSnapshot {
				*violations = append(*violations, "destructive-requires-pre-change-snapshot")
			}
		}
	}

	for _, helper := range edit.Helpers {
		if !helper.Resolved {
			*unresolved = append(*unresolved, "helper-effect-unknown:"+helper.Name)
			continue
		}
		for _, effect := range helper.Effects {
			if !contains(window.AllowedEffects, effect) {
				*violations = append(*violations, "helper-effect-outside-window:"+helper.Name)
				break
			}
		}
	}

	if edit.SemanticChange {
		*humanDecisions = append(*humanDecisions, "semantic-change-requires-human-decision")
	}
	if edit.AuthorityChange {
		*humanDecisions = append(*humanDecisions, "authority-change-requires-human-decision")
	}
}

func checkWholeGeneration(edit Edit, violations, unresolved *[]string) {
	if edit.AiWritesToProduction {
		*violations = append(*violations, "whole-generation-must-not-write-to-production")
	}
	if !edit.IsolatedBranch {
		*violations = append(*violations, "whole-generation-requires-isolated-branch")
	}
	if !edit.RoundTripEqual {
		*unresolved = append(*unresolved, "whole-generation-requires-round-trip-equality")
	}
	if !edit.HumanApproval {
		*unresolved = append(*unresolved, "whole-generation-requires-human-approval")
	}
}

func checkTargets(input Input) []TargetResult {
	required := requiredEffects(input.Edit)
	var results []TargetResult
	for _, target := range input.Targets {
		capabilities, exists := input.TargetCapabilities[target]
		if !exists {
			results = append(results, TargetResult{Target: target, Status: TargetUnresolved, Reasons: []string{"target-capabilities-unknown"}})
			continue
		}
		result := TargetResult{Target: target, Status: TargetVerifiedCandidate}
		for _, effect := range required {
			if !contains(capabilities, effect) {
				result.Status = TargetNotApplicable
				result.Reasons = append(result.Reasons, "effect-not-available:"+effect)
			}
		}
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Target < results[j].Target })
	return results
}

func requiredEffects(edit Edit) []string {
	set := map[string]bool{}
	for _, change := range edit.Changes {
		for _, effect := range change.Effects {
			set[effect] = true
		}
	}
	for _, helper := range edit.Helpers {
		for _, effect := range helper.Effects {
			set[effect] = true
		}
	}
	var effects []string
	for effect := range set {
		effects = append(effects, effect)
	}
	sort.Strings(effects)
	return effects
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	set := map[string]bool{}
	var out []string
	for _, value := range values {
		if set[value] {
			continue
		}
		set[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
