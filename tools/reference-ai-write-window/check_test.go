package aiwritewindow

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func loadFixture(t *testing.T) Input {
	t.Helper()
	input, err := LoadInput(filepath.Join("testdata", "input-window.json"))
	if err != nil {
		t.Fatalf("LoadInput: %v", err)
	}
	return input
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func targetByID(report Report, id string) TargetResult {
	for _, target := range report.Targets {
		if target.Target == id {
			return target
		}
	}
	return TargetResult{}
}

func TestInsideWindowEditIsCandidateAccepted(t *testing.T) {
	report := Check(loadFixture(t))
	if report.Status != StatusCandidateAccepted {
		t.Fatalf("status = %q, want %q; violations=%v unresolved=%v", report.Status, StatusCandidateAccepted, report.Violations, report.Unresolved)
	}
	if report.BusinessExecutionAllowed {
		t.Error("candidate acceptance must not grant business execution")
	}
}

func TestPathOutsideWindowIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Changes[0].Path = "src/order/permissions.go"
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasPrefix(report.Violations, "path-outside-window:") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestAuthorityChangeOutsideWindowIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Changes[0].Kind = "authority"
	input.Edit.Changes[0].Effects = []string{"grant"}
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Violations, "change-kind-outside-window:authority") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestDataTypeOutsideWindowIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Changes[0].Types = []string{"uuid"}
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Violations, "type-outside-window:uuid") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestExpressionOutsideWindowIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Changes[0].Expressions = []string{"customer.creditLimit"}
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Violations, "expression-outside-window:customer.creditLimit") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestEffectOutsideWindowIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Changes[0].Effects = []string{"write:audit"}
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Violations, "effect-outside-window:write:audit") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestSemanticChangeInsideWindowRequiresHumanDecision(t *testing.T) {
	input := loadFixture(t)
	input.Edit.SemanticChange = true
	report := Check(input)
	if report.Status != StatusHumanDecisionNeeded {
		t.Fatalf("status = %q, want human-decision-required", report.Status)
	}
	if !hasString(report.HumanDecisions, "semantic-change-requires-human-decision") {
		t.Errorf("humanDecisions = %v", report.HumanDecisions)
	}
}

func TestAuthorityChangeInsideWindowRequiresHumanDecision(t *testing.T) {
	input := loadFixture(t)
	input.Edit.AuthorityChange = true
	report := Check(input)
	if report.Status != StatusHumanDecisionNeeded {
		t.Fatalf("status = %q, want human-decision-required", report.Status)
	}
	if !hasString(report.HumanDecisions, "authority-change-requires-human-decision") {
		t.Errorf("humanDecisions = %v", report.HumanDecisions)
	}
}

func TestUnknownHelperEffectIsUnresolved(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Helpers = append(input.Edit.Helpers, Helper{Name: "mystery", Resolved: false})
	report := Check(input)
	if report.Status != StatusUnresolved {
		t.Fatalf("status = %q, want unresolved", report.Status)
	}
	if !hasString(report.Unresolved, "helper-effect-unknown:mystery") {
		t.Errorf("unresolved = %v", report.Unresolved)
	}
}

func TestHelperEffectOutsideWindowIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Helpers = append(input.Edit.Helpers, Helper{Name: "audit", Resolved: true, Effects: []string{"write:audit"}})
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Violations, "helper-effect-outside-window:audit") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestStaleProviderRevisionIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Edit.BaseProviderRevision = "order-plugin-0.9.0"
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Stale, "provider-revision-changed") {
		t.Errorf("stale = %v", report.Stale)
	}
}

func TestStaleSourceRevisionIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Edit.BaseSourceRevision = "rev-9"
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Stale, "source-revision-changed") {
		t.Errorf("stale = %v", report.Stale)
	}
}

func TestTargetWithoutRequiredEffectIsNotApplicable(t *testing.T) {
	input := loadFixture(t)
	input.Window.AllowedEffects = append(input.Window.AllowedEffects, "write:audit")
	input.Edit.Changes[0].Effects = []string{"read", "write:audit"}
	input.TargetCapabilities["go"] = []string{"read", "write:audit"}
	report := Check(input)
	if got := targetByID(report, "go"); got.Status != TargetVerifiedCandidate {
		t.Errorf("go = %q, want verified-candidate", got.Status)
	}
	wasm := targetByID(report, "wasm")
	if wasm.Status != TargetNotApplicable {
		t.Errorf("wasm = %q, want not-applicable", wasm.Status)
	}
	if !hasString(wasm.Reasons, "effect-not-available:write:audit") {
		t.Errorf("wasm reasons = %v", wasm.Reasons)
	}
	if report.BusinessExecutionAllowed {
		t.Error("a verified candidate is not a business execution permission")
	}
}

func TestUnknownTargetIsUnresolved(t *testing.T) {
	input := loadFixture(t)
	input.Targets = []string{"unknown-target"}
	report := Check(input)
	if report.Status != StatusUnresolved {
		t.Fatalf("status = %q, want unresolved", report.Status)
	}
	if !hasString(report.Unresolved, "target-unknown:unknown-target") {
		t.Errorf("unresolved = %v", report.Unresolved)
	}
}

func TestSelfReportedPassWithoutTestEvidenceIsUnresolved(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Tests = nil
	input.Edit.SelfReportedPass = true
	report := Check(input)
	if report.Status != StatusUnresolved {
		t.Fatalf("status = %q, want unresolved", report.Status)
	}
	if !hasString(report.Unresolved, "self-report-is-not-evidence") {
		t.Errorf("unresolved = %v", report.Unresolved)
	}
}

func TestIncompleteTestEvidenceIsUnresolved(t *testing.T) {
	input := loadFixture(t)
	input.Edit.Tests = []TestEvidence{{Command: "go test ./...", ExitCode: 0}}
	report := Check(input)
	if report.Status != StatusUnresolved {
		t.Fatalf("status = %q, want unresolved", report.Status)
	}
	if !hasString(report.Unresolved, "test-evidence-incomplete") {
		t.Errorf("unresolved = %v", report.Unresolved)
	}
}

func TestWholeGenerationCannotWriteToProduction(t *testing.T) {
	input := Input{
		Schema:             InputSchema,
		Mode:               ModeWholeGeneration,
		Targets:            []string{"go"},
		TargetCapabilities: map[string][]string{"go": {"read"}},
		Edit: Edit{
			Author:               "ai",
			IsolatedBranch:       true,
			RoundTripEqual:       true,
			HumanApproval:        true,
			AiWritesToProduction: true,
			Tests:                []TestEvidence{{Command: "go test ./...", Head: "abc", ExitCode: 0}},
		},
	}
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Violations, "whole-generation-must-not-write-to-production") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestWholeGenerationRequiresRoundTripAndApproval(t *testing.T) {
	input := Input{
		Schema:             InputSchema,
		Mode:               ModeWholeGeneration,
		Targets:            []string{"go"},
		TargetCapabilities: map[string][]string{"go": {"read"}},
		Edit: Edit{
			Author:         "ai",
			IsolatedBranch: true,
			RoundTripEqual: false,
			HumanApproval:  false,
			Tests:          []TestEvidence{{Command: "go test ./...", Head: "abc", ExitCode: 0}},
		},
	}
	report := Check(input)
	if report.Status != StatusUnresolved {
		t.Fatalf("status = %q, want unresolved", report.Status)
	}
	if !hasString(report.Unresolved, "whole-generation-requires-round-trip-equality") {
		t.Errorf("unresolved = %v", report.Unresolved)
	}
	if !hasString(report.Unresolved, "whole-generation-requires-human-approval") {
		t.Errorf("unresolved = %v", report.Unresolved)
	}
}

func TestWindowDeclaredByAIIsBlocked(t *testing.T) {
	input := loadFixture(t)
	input.Window.DeclaredBy = "ai"
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Violations, "window-must-be-declared-by-human") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestDestructiveChangeRequiresApprovalAndSnapshot(t *testing.T) {
	input := loadFixture(t)
	input.Window.AllowedKinds = append(input.Window.AllowedKinds, "destructive")
	input.Edit.Changes[0].Kind = "destructive"
	input.Edit.DestructiveApproval = false
	input.Edit.PreChangeSnapshot = false
	report := Check(input)
	if report.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !hasString(report.Violations, "destructive-requires-human-approval") {
		t.Errorf("violations = %v", report.Violations)
	}
	if !hasString(report.Violations, "destructive-requires-pre-change-snapshot") {
		t.Errorf("violations = %v", report.Violations)
	}
}

func TestCheckDoesNotMutateInput(t *testing.T) {
	input := loadFixture(t)
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var original Input
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	_ = Check(input)
	if !reflect.DeepEqual(input, original) {
		t.Fatal("Check changed the input")
	}
}
