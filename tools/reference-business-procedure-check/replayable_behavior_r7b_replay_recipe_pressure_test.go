package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
)

type replayableBehaviorR7BReplayRecipe struct {
	BehaviorID                      string
	SemanticOperation               string
	TargetRealizationRef            string
	RequestFixtureID                string
	TargetLocalInteraction          string
	ExpectedHTTPStatus              int
	ExpectedObservationClass        string
	OriginRawArtifactFile           string
	OriginRawArtifactSHA256         string
	HandoffArtifactSHA256           string
	GateCReadinessArtifactSHA256    string
	SemanticFunctionalSummarySHA256 string
	HandoverBaselineID              string
	BaselineRepositoryRevision      string
	OriginTargetBuildRevision       string
	ReplayTargetBuildRevision       string
	SemanticFunctionalRun           string
	VisualRun                       string
	ReplayExecutionStatus           string
	RepeatabilityStatus             string
	CoverageClass                   string
	TargetInteractionCanonical      bool
}

type replayableBehaviorR7BReplayObservation struct {
	BehaviorID                string
	ReplayRunID               string
	ReplayTargetBuildRevision string
	ObservedHTTPStatus        int
	ObservedSemanticClass     string
	RawReplayArtifactSHA256   string
	ExecutionPassed           bool
}

type replayableBehaviorR7BRepeatabilityAssessment struct {
	BehaviorID              string
	RepeatabilityStatus     string
	SuccessfulReplayRuns    int
	CoverageClass           string
	DeepWorkflowCovered     bool
	OriginRawArtifactSHA256 string
}

func TestReplayableBehaviorR7BReplayRecipeDerivesAllSixBehaviors(t *testing.T) {
	intake, err := replayableBehaviorR7BDeriveExecutionIntake(replayableBehaviorR7BTestEntry())
	if err != nil {
		t.Fatal(err)
	}

	got, err := replayableBehaviorR7BDeriveReplayRecipes(intake)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("R7B replay recipe count=%d, want 6", len(got))
	}

	want := map[string]struct {
		fixture string
		local   string
		status  int
	}{
		"first-web/read-only": {
			fixture: "gate-c/read-from-0830-row-0",
			local:   "GET /scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0",
			status:  200,
		},
		"first-web/command-success": {
			fixture: "gate-c/command-success",
			local:   "POST /scenario/submit fixture=gate-c/command-success",
			status:  200,
		},
		"first-web/authentication-denied": {
			fixture: "gate-c/authentication-denied",
			local:   "POST /scenario/submit fixture=gate-c/authentication-denied",
			status:  200,
		},
		"first-web/authorization-denied": {
			fixture: "gate-c/authorization-denied",
			local:   "POST /scenario/submit fixture=gate-c/authorization-denied",
			status:  200,
		},
		"first-web/validation-not-satisfied": {
			fixture: "gate-c/validation-not-satisfied",
			local:   "POST /scenario/submit fixture=gate-c/validation-not-satisfied",
			status:  200,
		},
		"first-web/execution-error": {
			fixture: "gate-c/execution-error",
			local:   "POST /scenario/submit fixture=gate-c/execution-error",
			status:  422,
		},
	}

	for _, recipe := range got {
		expected, ok := want[recipe.BehaviorID]
		if !ok {
			t.Fatalf("unexpected R7B behavior %q", recipe.BehaviorID)
		}
		if recipe.RequestFixtureID != expected.fixture ||
			recipe.TargetLocalInteraction != expected.local ||
			recipe.ExpectedHTTPStatus != expected.status {
			t.Fatalf("R7B target-local replay recipe drifted for %q: %+v", recipe.BehaviorID, recipe)
		}
		if recipe.ReplayExecutionStatus != "not-executed" ||
			recipe.RepeatabilityStatus != "not-observed" ||
			recipe.CoverageClass != "behavior-unit" {
			t.Fatalf("R7B replay recipe fabricated execution state: %+v", recipe)
		}
		if recipe.TargetInteractionCanonical {
			t.Fatalf("target-local replay interaction was promoted to Canonical identity: %+v", recipe)
		}
		if recipe.ReplayTargetBuildRevision != recipe.OriginTargetBuildRevision {
			t.Fatalf("first replay did not pin the origin Target build: %+v", recipe)
		}
		if recipe.OriginRawArtifactSHA256 == "" ||
			recipe.HandoffArtifactSHA256 == "" ||
			recipe.GateCReadinessArtifactSHA256 == "" ||
			recipe.SemanticFunctionalSummarySHA256 == "" {
			t.Fatalf("R7B replay recipe lost exact Evidence provenance: %+v", recipe)
		}
	}
}

func TestReplayableBehaviorR7BReplayRecipeRejectsFabricatedExecutionState(t *testing.T) {
	intake, err := replayableBehaviorR7BDeriveExecutionIntake(replayableBehaviorR7BTestEntry())
	if err != nil {
		t.Fatal(err)
	}
	intake[0].ReplayExecutionStatus = "passed"
	if _, err := replayableBehaviorR7BDeriveReplayRecipes(intake); err == nil {
		t.Fatal("R7B replay recipe accepted intake that already claimed replay success")
	}
}

func TestReplayableBehaviorR7BRepeatabilityRequiresTwoDistinctSuccessfulRuns(t *testing.T) {
	recipe := replayableBehaviorR7BTestRecipe(t, "first-web/command-success")

	one := []replayableBehaviorR7BReplayObservation{
		replayableBehaviorR7BMatchingObservation(recipe, "replay-1", replayableBehaviorR7BTestSHA256("replay-1")),
	}
	if _, err := replayableBehaviorR7BEvaluateRepeatability(recipe, one); err == nil {
		t.Fatal("one replay pass was treated as repeatability")
	}

	two := []replayableBehaviorR7BReplayObservation{
		replayableBehaviorR7BMatchingObservation(recipe, "replay-1", replayableBehaviorR7BTestSHA256("replay-1")),
		replayableBehaviorR7BMatchingObservation(recipe, "replay-2", replayableBehaviorR7BTestSHA256("replay-2")),
	}
	assessment, err := replayableBehaviorR7BEvaluateRepeatability(recipe, two)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.RepeatabilityStatus != "observed-repeatable" || assessment.SuccessfulReplayRuns != 2 {
		t.Fatalf("repeatability assessment=%+v", assessment)
	}
	if assessment.CoverageClass != "behavior-unit" || assessment.DeepWorkflowCovered {
		t.Fatalf("behavior repeatability was inflated into workflow coverage: %+v", assessment)
	}
	if assessment.OriginRawArtifactSHA256 != recipe.OriginRawArtifactSHA256 {
		t.Fatalf("repeatability assessment lost origin raw Evidence identity: %+v", assessment)
	}
}

func TestReplayableBehaviorR7BRepeatabilityComparesMeaningNotExactReplayBytes(t *testing.T) {
	recipe := replayableBehaviorR7BTestRecipe(t, "first-web/read-only")
	observations := []replayableBehaviorR7BReplayObservation{
		replayableBehaviorR7BMatchingObservation(recipe, "replay-a", replayableBehaviorR7BTestSHA256("replay-a")),
		replayableBehaviorR7BMatchingObservation(recipe, "replay-b", replayableBehaviorR7BTestSHA256("replay-b")),
	}
	if _, err := replayableBehaviorR7BEvaluateRepeatability(recipe, observations); err != nil {
		t.Fatalf("different exact replay bytes with the same observed meaning should remain repeatable: %v", err)
	}
}

func TestReplayableBehaviorR7BRepeatabilityRejectsSemanticOrTargetDrift(t *testing.T) {
	recipe := replayableBehaviorR7BTestRecipe(t, "first-web/authorization-denied")

	t.Run("semantic-class", func(t *testing.T) {
		observations := []replayableBehaviorR7BReplayObservation{
			replayableBehaviorR7BMatchingObservation(recipe, "replay-1", replayableBehaviorR7BTestSHA256("sha-1")),
			replayableBehaviorR7BMatchingObservation(recipe, "replay-2", replayableBehaviorR7BTestSHA256("sha-2")),
		}
		observations[1].ObservedSemanticClass = "different-meaning"
		if _, err := replayableBehaviorR7BEvaluateRepeatability(recipe, observations); err == nil {
			t.Fatal("semantic drift was treated as repeatable")
		}
	})

	t.Run("target-revision", func(t *testing.T) {
		observations := []replayableBehaviorR7BReplayObservation{
			replayableBehaviorR7BMatchingObservation(recipe, "replay-1", replayableBehaviorR7BTestSHA256("sha-1")),
			replayableBehaviorR7BMatchingObservation(recipe, "replay-2", replayableBehaviorR7BTestSHA256("sha-2")),
		}
		observations[1].ReplayTargetBuildRevision = "different-target-revision"
		if _, err := replayableBehaviorR7BEvaluateRepeatability(recipe, observations); err == nil {
			t.Fatal("Target build drift was treated as repeatability")
		}
	})

	t.Run("raw-artifact-digest", func(t *testing.T) {
		observations := []replayableBehaviorR7BReplayObservation{
			replayableBehaviorR7BMatchingObservation(recipe, "replay-1", replayableBehaviorR7BTestSHA256("sha-1")),
			replayableBehaviorR7BMatchingObservation(recipe, "replay-2", "not-a-sha256"),
		}
		if _, err := replayableBehaviorR7BEvaluateRepeatability(recipe, observations); err == nil {
			t.Fatal("replay without exact raw artifact identity was treated as repeatable")
		}
	})
}

func replayableBehaviorR7BDeriveReplayRecipes(
	intake []replayableBehaviorR7BExecutionIntake,
) ([]replayableBehaviorR7BReplayRecipe, error) {
	expected := replayableBehaviorR7BExpectedFirstWebBehaviors()
	if len(intake) != len(expected) {
		return nil, fmt.Errorf("R7B execution intake count=%d, want %d", len(intake), len(expected))
	}

	seen := map[string]struct{}{}
	out := make([]replayableBehaviorR7BReplayRecipe, 0, len(intake))
	for _, item := range intake {
		if _, ok := expected[item.BehaviorID]; !ok {
			return nil, fmt.Errorf("unknown R7B behavior %q", item.BehaviorID)
		}
		if _, duplicate := seen[item.BehaviorID]; duplicate {
			return nil, fmt.Errorf("duplicate R7B behavior %q", item.BehaviorID)
		}
		seen[item.BehaviorID] = struct{}{}
		if item.ReplayExecutionStatus != "not-executed" || item.RepeatabilityStatus != "not-observed" {
			return nil, fmt.Errorf("behavior %q entered replay recipe with fabricated execution state", item.BehaviorID)
		}
		if item.RawArtifactSHA256 == "" ||
			item.HandoffArtifactSHA256 == "" ||
			item.GateCReadinessArtifactSHA256 == "" ||
			item.SemanticFunctionalSummarySHA256 == "" {
			return nil, fmt.Errorf("behavior %q lacks exact Evidence provenance", item.BehaviorID)
		}

		fixture, interaction, status, err := replayableBehaviorR7BTargetLocalRecipe(item.BehaviorID)
		if err != nil {
			return nil, err
		}
		out = append(out, replayableBehaviorR7BReplayRecipe{
			BehaviorID:                      item.BehaviorID,
			SemanticOperation:               item.SemanticOperation,
			TargetRealizationRef:            item.TargetRealizationRef,
			RequestFixtureID:                fixture,
			TargetLocalInteraction:          interaction,
			ExpectedHTTPStatus:              status,
			ExpectedObservationClass:        item.ObservationClass,
			OriginRawArtifactFile:           item.RawArtifactFile,
			OriginRawArtifactSHA256:         item.RawArtifactSHA256,
			HandoffArtifactSHA256:           item.HandoffArtifactSHA256,
			GateCReadinessArtifactSHA256:    item.GateCReadinessArtifactSHA256,
			SemanticFunctionalSummarySHA256: item.SemanticFunctionalSummarySHA256,
			HandoverBaselineID:              item.HandoverBaselineID,
			BaselineRepositoryRevision:      item.BaselineRepositoryRevision,
			OriginTargetBuildRevision:       item.TargetBuildRevision,
			ReplayTargetBuildRevision:       item.TargetBuildRevision,
			SemanticFunctionalRun:           item.SemanticFunctionalRun,
			VisualRun:                       item.VisualRun,
			ReplayExecutionStatus:           "not-executed",
			RepeatabilityStatus:             "not-observed",
			CoverageClass:                   "behavior-unit",
			TargetInteractionCanonical:      false,
		})
	}
	if len(seen) != len(expected) {
		return nil, fmt.Errorf("R7B replay recipe behavior set is incomplete")
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].BehaviorID < out[j].BehaviorID
	})
	return out, nil
}

func replayableBehaviorR7BTargetLocalRecipe(behaviorID string) (string, string, int, error) {
	switch behaviorID {
	case "first-web/read-only":
		return "gate-c/read-from-0830-row-0", "GET /scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0", 200, nil
	case "first-web/command-success":
		return "gate-c/command-success", "POST /scenario/submit fixture=gate-c/command-success", 200, nil
	case "first-web/authentication-denied":
		return "gate-c/authentication-denied", "POST /scenario/submit fixture=gate-c/authentication-denied", 200, nil
	case "first-web/authorization-denied":
		return "gate-c/authorization-denied", "POST /scenario/submit fixture=gate-c/authorization-denied", 200, nil
	case "first-web/validation-not-satisfied":
		return "gate-c/validation-not-satisfied", "POST /scenario/submit fixture=gate-c/validation-not-satisfied", 200, nil
	case "first-web/execution-error":
		return "gate-c/execution-error", "POST /scenario/submit fixture=gate-c/execution-error", 422, nil
	default:
		return "", "", 0, fmt.Errorf("no target-local R7B replay recipe for %q", behaviorID)
	}
}

func replayableBehaviorR7BEvaluateRepeatability(
	recipe replayableBehaviorR7BReplayRecipe,
	observations []replayableBehaviorR7BReplayObservation,
) (replayableBehaviorR7BRepeatabilityAssessment, error) {
	if recipe.ReplayExecutionStatus != "not-executed" || recipe.RepeatabilityStatus != "not-observed" {
		return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("replay recipe already carries an execution result")
	}
	if len(observations) < 2 {
		return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("repeatability requires at least two replay observations")
	}

	runs := map[string]struct{}{}
	for _, observation := range observations {
		if observation.BehaviorID != recipe.BehaviorID {
			return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("replay observation behavior mismatch")
		}
		if observation.ReplayRunID == "" {
			return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("replay observation lacks run identity")
		}
		if _, duplicate := runs[observation.ReplayRunID]; duplicate {
			return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("repeatability requires distinct replay runs")
		}
		runs[observation.ReplayRunID] = struct{}{}
		if observation.ReplayTargetBuildRevision != recipe.ReplayTargetBuildRevision {
			return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("replay Target build drifted")
		}
		if !observation.ExecutionPassed {
			return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("replay execution did not pass")
		}
		if observation.ObservedHTTPStatus != recipe.ExpectedHTTPStatus {
			return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("replay transport result drifted")
		}
		if observation.ObservedSemanticClass != recipe.ExpectedObservationClass {
			return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("replay semantic result drifted")
		}
		if !replayableBehaviorR7BIsSHA256(observation.RawReplayArtifactSHA256) {
			return replayableBehaviorR7BRepeatabilityAssessment{}, fmt.Errorf("replay raw artifact does not have an exact SHA-256 identity")
		}
	}

	return replayableBehaviorR7BRepeatabilityAssessment{
		BehaviorID:              recipe.BehaviorID,
		RepeatabilityStatus:     "observed-repeatable",
		SuccessfulReplayRuns:    len(observations),
		CoverageClass:           "behavior-unit",
		DeepWorkflowCovered:     false,
		OriginRawArtifactSHA256: recipe.OriginRawArtifactSHA256,
	}, nil
}

func replayableBehaviorR7BTestRecipe(t *testing.T, behaviorID string) replayableBehaviorR7BReplayRecipe {
	t.Helper()
	intake, err := replayableBehaviorR7BDeriveExecutionIntake(replayableBehaviorR7BTestEntry())
	if err != nil {
		t.Fatal(err)
	}
	recipes, err := replayableBehaviorR7BDeriveReplayRecipes(intake)
	if err != nil {
		t.Fatal(err)
	}
	for _, recipe := range recipes {
		if recipe.BehaviorID == behaviorID {
			return recipe
		}
	}
	t.Fatalf("test recipe %q not found", behaviorID)
	return replayableBehaviorR7BReplayRecipe{}
}

func replayableBehaviorR7BIsSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func replayableBehaviorR7BTestSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func replayableBehaviorR7BMatchingObservation(
	recipe replayableBehaviorR7BReplayRecipe,
	runID string,
	rawSHA256 string,
) replayableBehaviorR7BReplayObservation {
	return replayableBehaviorR7BReplayObservation{
		BehaviorID:                recipe.BehaviorID,
		ReplayRunID:               runID,
		ReplayTargetBuildRevision: recipe.ReplayTargetBuildRevision,
		ObservedHTTPStatus:        recipe.ExpectedHTTPStatus,
		ObservedSemanticClass:     recipe.ExpectedObservationClass,
		RawReplayArtifactSHA256:   rawSHA256,
		ExecutionPassed:           true,
	}
}
