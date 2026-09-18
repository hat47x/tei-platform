package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root with go.mod not found")
		}
		dir = parent
	}
}

func referenceFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(
		repoRoot(t),
		"reference",
		"sei-adoption",
		"decision-continuity-counterfactual-001.json",
	)
}

func loadReferenceFixture(t *testing.T) fixture {
	t.Helper()
	f, err := readFixture(referenceFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestReferenceFixtureProjectsDecisionContinuityWithoutTemporalLeak(t *testing.T) {
	result, err := run(referenceFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}

	if result.FixtureID != "e73-decision-continuity-001-provider-pilot" {
		t.Fatalf("fixture id = %q", result.FixtureID)
	}
	if result.InitialResume.CurrentQuestion != "通知Provider Aのtimeoutが増えたためProvider Bへ移行すべきか。" {
		t.Fatalf("current question must come from scenario, got %q", result.InitialResume.CurrentQuestion)
	}
	if result.InitialResume.Decision.ID != "dec-pilot" {
		t.Fatalf("decision id = %q", result.InitialResume.Decision.ID)
	}
	if got := ids(result.InitialResume.Recommendations); !equalStringSlices(got, []string{"rec-full-switch"}) {
		t.Fatalf("recommendation ids = %#v", got)
	}
	if got := ids(result.InitialResume.HistoricalBasis); !equalStringSlices(got, []string{"obs-a-timeout", "obs-b-synthetic"}) {
		t.Fatalf("historical basis ids = %#v", got)
	}
	if got := ids(result.InitialResume.Unresolved); !equalStringSlices(got, []string{"unk-b-rate-limit", "unk-rollback"}) {
		t.Fatalf("unresolved ids = %#v", got)
	}
	if len(result.InitialResume.ChangesSinceDecision) != 0 {
		t.Fatalf("initial Resume contains post-decision change: %#v", result.InitialResume.ChangesSinceDecision)
	}
	if got := ids(result.PostDecision.ChangesSinceDecision); !equalStringSlices(got, []string{"obs-b-429", "obs-a-maintenance"}) {
		t.Fatalf("post-decision ids = %#v", got)
	}
	if !result.SemanticAudit.RecommendationSeparated ||
		!result.SemanticAudit.DecisionMatchesOracle ||
		!result.SemanticAudit.RecommendationMatchesOracle ||
		!result.SemanticAudit.HistoricalBasisMatches ||
		!result.SemanticAudit.UnresolvedMatches ||
		!result.SemanticAudit.PostDecisionMatches ||
		!result.SemanticAudit.TemporalSeparation {
		t.Fatalf("semantic audit not fully satisfied: %#v", result.SemanticAudit)
	}
}

func TestReferenceProjectionDoesNotUseOracleQuestionAsInput(t *testing.T) {
	f := loadReferenceFixture(t)
	raw := f.Conditions["C_sei_resume"]
	var condition resumeCondition
	if err := json.Unmarshal(raw, &condition); err != nil {
		t.Fatal(err)
	}
	condition.SurfaceText.Question = "oracle-only question that must not drive projection"
	changed, err := json.Marshal(condition)
	if err != nil {
		t.Fatal(err)
	}
	f.Conditions["C_sei_resume"] = changed

	result, err := projectFixture(f)
	if err != nil {
		t.Fatal(err)
	}
	if result.InitialResume.CurrentQuestion != f.Scenario.InitialQuestion {
		t.Fatalf("projection question = %q; scenario question = %q", result.InitialResume.CurrentQuestion, f.Scenario.InitialQuestion)
	}
}

func TestReferenceRejectsMissingHistoricalBasisReference(t *testing.T) {
	f := loadReferenceFixture(t)
	for i := range f.Scenario.Phase1 {
		if f.Scenario.Phase1[i].Role == "decision" {
			f.Scenario.Phase1[i].BasisRefs = append(f.Scenario.Phase1[i].BasisRefs, "obs-missing")
		}
	}
	if _, err := projectFixture(f); err == nil || !strings.Contains(err.Error(), "does not resolve within phase_1") {
		t.Fatalf("expected missing ref rejection, got %v", err)
	}
}

func TestReferenceRejectsRecommendationRoleConfusion(t *testing.T) {
	f := loadReferenceFixture(t)
	for i := range f.Scenario.Phase1 {
		if f.Scenario.Phase1[i].Role == "decision" {
			f.Scenario.Phase1[i].RecommendationRefs = []string{"unk-b-rate-limit"}
		}
	}
	if _, err := projectFixture(f); err == nil || !strings.Contains(err.Error(), "resolves role") {
		t.Fatalf("expected recommendation role rejection, got %v", err)
	}
}

func TestReferenceRejectsUnknownRole(t *testing.T) {
	f := loadReferenceFixture(t)
	f.Scenario.Phase1[0].Role = "summary"
	if _, err := projectFixture(f); err == nil || !strings.Contains(err.Error(), "unsupported role") {
		t.Fatalf("expected unknown role rejection, got %v", err)
	}
}

func TestReferenceRejectsTemporalIDCollision(t *testing.T) {
	f := loadReferenceFixture(t)
	f.Scenario.PostDecisionEvidence[0].ID = f.Scenario.Phase1[0].ID
	if _, err := projectFixture(f); err == nil || !strings.Contains(err.Error(), "exists in both") {
		t.Fatalf("expected temporal id collision rejection, got %v", err)
	}
}

func TestDecodeFixtureRejectsUnknownTopLevelField(t *testing.T) {
	payload := `{"schema":"sei.decision-continuity-comparison-fixture/v0","status":"evaluation-only","fixture_id":"x","unknown":true}`
	if _, err := decodeFixture(strings.NewReader(payload)); err == nil {
		t.Fatal("expected unknown top-level field rejection")
	}
}
