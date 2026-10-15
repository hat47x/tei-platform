package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/scenariotest"
	"tei/internal/workspace"
)

func prepareScenarioPlanBinding(t *testing.T, changeSQL bool) (beforeRoot, afterRoot, scenarioPath, planPath string) {
	t.Helper()
	referenceWorkspace := filepath.Join("..", "..", "reference", "workspace")
	beforeRoot, afterRoot = t.TempDir(), t.TempDir()
	for _, root := range []string{beforeRoot, afterRoot} {
		if err := os.CopyFS(root, os.DirFS(referenceWorkspace)); err != nil {
			t.Fatalf("copy real reference Workspace: %v", err)
		}
	}
	if changeSQL {
		relative := filepath.Join("applications", "order-entry", "queries", "recent_orders.sql")
		path := filepath.Join(afterRoot, relative)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		changed := strings.Replace(string(raw), "ORDER BY o.ordered_at DESC", "ORDER BY o.ordered_at ASC", 1)
		if changed == string(raw) {
			t.Fatal("reference SQL did not contain expected ORDER BY clause")
		}
		if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
			t.Fatal(err)
		}
	}
	scenarioRoot := t.TempDir()
	if err := os.CopyFS(scenarioRoot, os.DirFS(filepath.Join("..", "..", "reference", "driver-scenarios"))); err != nil {
		t.Fatal(err)
	}
	scenarioPath = filepath.Join(scenarioRoot, "order-entry-pattern-smoke.json")
	loaded, err := workspace.Load(afterRoot)
	if err != nil {
		t.Fatal(err)
	}
	generated, s, err := buildExpectedScenarioPlan(scenarioPath, afterRoot, loaded)
	if err != nil {
		t.Fatalf("derive existing Scenario/CLI Plan: %v", err)
	}
	if s.ID != "order-entry-browse-pattern-smoke" || len(generated.Commands) != 6 {
		t.Fatalf("actual persisted Driver Scenario or generated command coverage changed: %+v", generated)
	}
	planPath = filepath.Join(t.TempDir(), "generated-plan.json")
	data, err := json.MarshalIndent(generated, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	return
}

func TestScenarioPlanJoinsActualSourceDiffWithoutInventingReceipt(t *testing.T) {
	beforeRoot, afterRoot, scenarioPath, planPath := prepareScenarioPlanBinding(t, true)
	consumer := workspaceKey("application", "order_entry")
	result, err := reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference", consumer, scenarioPath, planPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.ScenarioID != "order-entry-browse-pattern-smoke" ||
		!result.PlanMatches || result.ScenarioSHA256 == "" ||
		result.ScenarioStepCount != 6 || result.SyntheticFixtureStepCount != 4 ||
		len(result.FixtureSHA256) != 4 {
		t.Fatalf("Scenario Plan association lost steps, fixtures or pins: %+v", result)
	}
	if !result.Diff.Impact.GraphValid || len(result.Diff.Impact.Changes) != 0 ||
		len(result.Diff.SourceChanges) != 1 ||
		result.Diff.Decision.Action != "unresolved" ||
		result.Diff.Decision.ReasonCodes[0] != "source-changed-without-typed-graph-diff" {
		t.Fatalf("source-only SQL change hidden by Scenario Plan: %+v", result.Diff)
	}
	if result.ObservedAfterSourceRevision != result.Diff.AfterSourceRevision ||
		result.Diff.AfterSourceRevision == result.Diff.BeforeSourceRevision ||
		result.PlanPinsSourceRevision || result.Qualification != "unresolved" ||
		result.ExecutionAllowed || len(result.ReasonCodes) == 0 {
		t.Fatalf("synthetic Scenario or Plan promoted into independent evidence: %+v", result)
	}
}

func TestScenarioPlanMatchingCannotQualifyUnchangedWorkspace(t *testing.T) {
	beforeRoot, afterRoot, scenarioPath, planPath := prepareScenarioPlanBinding(t, false)
	result, err := reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference",
		workspaceKey("application", "order_entry"), scenarioPath, planPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diff.SourceChanges) != 0 || !result.PlanMatches ||
		result.Qualification != "unresolved" || result.PlanPinsSourceRevision || result.ExecutionAllowed {
		t.Fatalf("unchanged Source plus matching Plan was incorrectly qualified: %+v", result)
	}
}

func TestScenarioPlanRejectsStaleFixtureScenarioAndTamperedCommands(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, scenarioPath, planPath string)
	}{
		{"changed-scenario", func(t *testing.T, scenarioPath, planPath string) {
			t.Helper()
			source, err := os.ReadFile(scenarioPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(scenarioPath, append(source, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{"changed-fixture", func(t *testing.T, scenarioPath, planPath string) {
			t.Helper()
			path := filepath.Join(filepath.Dir(scenarioPath), "fixtures", "order_valid.json")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(source, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{"modified-generated-command", func(t *testing.T, scenarioPath, planPath string) {
			t.Helper()
			raw, err := os.ReadFile(planPath)
			if err != nil {
				t.Fatal(err)
			}
			var plan scenariotest.Plan
			if err := json.Unmarshal(raw, &plan); err != nil {
				t.Fatal(err)
			}
			plan.Commands[0].Argv = []string{"tei", "scenario", "run", "-step", "not-the-original-command"}
			changed, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(planPath, changed, 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{"untrusted-scenario-application", func(t *testing.T, scenarioPath, planPath string) {
			t.Helper()
			raw, err := os.ReadFile(scenarioPath)
			if err != nil {
				t.Fatal(err)
			}
			changed := strings.Replace(string(raw), `"application": "order_entry"`, `"application": "repair_intake"`, 1)
			if changed == string(raw) {
				t.Fatal("Scenario application not found")
			}
			if err := os.WriteFile(scenarioPath, []byte(changed), 0600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			beforeRoot, afterRoot, scenarioPath, planPath := prepareScenarioPlanBinding(t, false)
			test.mutate(t, scenarioPath, planPath)
			_, err := reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference",
				workspaceKey("application", "order_entry"), scenarioPath, planPath)
			if err == nil {
				t.Fatal("stale, tampered, or foreign Scenario/Plan was accepted")
			}
		})
	}
}

func TestScenarioPlanRejectsInvalidPathAndMissingPin(t *testing.T) {
	beforeRoot, afterRoot, scenarioPath, planPath := prepareScenarioPlanBinding(t, false)
	if _, err := reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference",
		workspaceKey("semantic-data", "sales.order"), scenarioPath, planPath); err == nil {
		t.Fatal("non-Application consumer gained Scenario qualification")
	}
	raw, err := os.ReadFile(scenarioPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(raw), "fixtures/order_valid.json", "../outside.json", 1)
	if changed == string(raw) {
		t.Fatal("expected fixture path not found")
	}
	if err := os.WriteFile(scenarioPath, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference",
		workspaceKey("application", "order_entry"), scenarioPath, planPath); err == nil {
		t.Fatal("Scenario fixture escaped directory")
	}
}

func TestScenarioPlanRejectsStalePluginLock(t *testing.T) {
	beforeRoot, afterRoot, scenarioPath, planPath := prepareScenarioPlanBinding(t, false)
	path := filepath.Join(afterRoot, ".tei", "plugin-lock.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference",
		workspaceKey("application", "order_entry"), scenarioPath, planPath)
	if err == nil {
		t.Fatal("stale Plugin Lock accepted by generated Plan pins")
	}
}

// The existing CLI Plan World fingerprint protects native SQL parsed into
// Canonical World. It must not be described as an unpinned SQL edit.
func TestScenarioPlanWorldPinRejectsChangedSQL(t *testing.T) {
	beforeRoot, afterRoot, scenarioPath, planPath := prepareScenarioPlanBinding(t, false)
	consumer := workspaceKey("application", "order_entry")
	path := filepath.Join(afterRoot, "applications", "order-entry", "queries", "recent_orders.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	altered := strings.Replace(string(raw), "ORDER BY o.ordered_at DESC", "ORDER BY o.ordered_at ASC", 1)
	if altered == string(raw) {
		t.Fatal("SQL reference fixture drift")
	}
	if err := os.WriteFile(path, []byte(altered), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference", consumer, scenarioPath, planPath)
	if err == nil || !strings.Contains(err.Error(), "stale or modified") {
		t.Fatalf("existing World pin failed to reject changed SQL: %v", err)
	}
}

// The generated CLI Plan does not pin sourceindex.ModelRevision itself.
// A pure whitespace edit in a managed semantic source should change the
// Source Index revision without changing the parsed World or typed Graph.
func TestScenarioPlanDoesNotPinWhitespaceOnlySourceRevision(t *testing.T) {
	beforeRoot, afterRoot, scenarioPath, planPath := prepareScenarioPlanBinding(t, false)
	consumer := modelref.Key{Scope: "order-entry-reference", Subject: workspaceKey("application", "order_entry").Subject}
	before, err := reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference", consumer, scenarioPath, planPath)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(afterRoot, "semantic", "sales", "order.tei")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference", consumer, scenarioPath, planPath)
	if err != nil {
		t.Fatal(err)
	}
	if !after.PlanMatches || before.ScenarioSHA256 != after.ScenarioSHA256 ||
		before.ObservedAfterSourceRevision == after.ObservedAfterSourceRevision ||
		len(after.Diff.SourceChanges) != 1 || len(after.Diff.Impact.Changes) != 0 ||
		after.Diff.Decision.Action != "unresolved" || after.PlanPinsSourceRevision ||
		after.Qualification != "unresolved" || after.ExecutionAllowed {
		t.Fatalf("Plan match incorrectly proved exact Source revision: before=%+v after=%+v", before, after)
	}
}
