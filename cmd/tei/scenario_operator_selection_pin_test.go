package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/scenariotest"
	"tei/internal/workspace"
)

// A generated Scenario command must not silently call implementation B after
// the manifest selected A, even when both happen to return the same result.
// This is a product CLI-plan test, not a Provider/Host or Owner-acceptance test.
func TestGeneratedScenarioCommandInvalidatesOperatorSelectionChange(t *testing.T) {
	const operator = "sales.non_negative"
	const a = "tei.number.non-negative"
	const b = "tei.number.non-zero"
	const selectionA = "implementations:\n  - operator: " + operator + "\n    use: " + a + "\n"
	root := scenarioOperatorWorkspace(t, selectionA)

	// The scenario entry already pins a Plugin Lock. Use the same fixed lock
	// for A and B so a changed result cannot be attributed to lock turnover.
	repoLock := filepath.Join("..", "..", "reference", "workspace", ".tei", "plugin-lock.json")
	lock, err := os.ReadFile(repoLock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".tei"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".tei", "plugin-lock.json"), lock, 0600); err != nil {
		t.Fatal(err)
	}
	scenarioPath := filepath.Join(root, "operator-scenario.json")
	fixtureDir := filepath.Join(root, "fixtures")
	if err := os.MkdirAll(fixtureDir, 0700); err != nil {
		t.Fatal(err)
	}
	fixture, err := json.Marshal(scenariotest.ValidationFixture{
		Numbers: map[string]string{"sales.order_line.discount": "2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "discount.json"), fixture, 0600); err != nil {
		t.Fatal(err)
	}
	scenario, err := json.Marshal(scenariotest.Scenario{
		SchemaVersion: scenariotest.SchemaVersion,
		ID: "discount-operator-pinned",
		Application: "discount_check",
		Steps: []scenariotest.Step{{
			ID: "discount", Operation: "evaluate-validations",
			Fixture: "fixtures/discount.json",
			ExpectValidations: map[string]string{scenarioOperatorRule: "satisfied"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scenarioPath, scenario, 0600); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) (string, error) {
		t.Helper()
		var output bytes.Buffer
		err := runScenarioCommand(args, &output)
		return output.String(), err
	}
	generate := func() scenariotest.Plan {
		t.Helper()
		result, err := run("generate", "-scenario", scenarioPath, "-workspace", root)
		if err != nil {
			t.Fatalf("real Scenario generate: %v", err)
		}
		var plan scenariotest.Plan
		if err := json.Unmarshal([]byte(result), &plan); err != nil {
			t.Fatalf("decode generated Plan: %v", err)
		}
		if len(plan.Commands) != 1 || len(plan.OperatorSelectionSHA256) != 64 {
			t.Fatalf("generated Plan lost Operator selection pin: %+v", plan)
		}
		return plan
	}
	assertCalled := func(plan scenariotest.Plan, expected string) {
		t.Helper()
		result, err := run(plan.Commands[0].Argv[2:]...)
		if err != nil {
			t.Fatalf("generated CLI run: %v", err)
		}
		var receipt struct {
			ExecutionMode string                 `json:"executionMode"`
			Evidence      []scenarioStepEvidence `json:"evidence"`
		}
		if err := json.Unmarshal([]byte(result), &receipt); err != nil {
			t.Fatal(err)
		}
		if receipt.ExecutionMode != "generated-pinned" ||
			len(receipt.Evidence) != 1 || !receipt.Evidence[0].Verified ||
			receipt.Evidence[0].ValidationResults[scenarioOperatorRule] != "satisfied" ||
			receipt.Evidence[0].ValidationImplementations[scenarioOperatorRule] != expected {
			t.Fatalf("generated run used wrong implementation: %s", result)
		}
	}
	before, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	oldPlan := generate()
	assertCalled(oldPlan, a)

	// A changed selection leaves Semantic World, Scenario, Plugin Lock and
	// input Fixture intact; the result of both operators on 2 is also true.
	manifestPath := filepath.Join(root, "tei.workspace.yaml")
	original, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(original), "use: "+a) != 1 {
		t.Fatal("manifest selection A is not unique")
	}
	edited := []byte(strings.Replace(string(original), "use: "+a, "use: "+b, 1))
	if err := os.WriteFile(manifestPath, edited, 0600); err != nil {
		t.Fatal(err)
	}
	after, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	worldA, err := json.Marshal(before.World)
	if err != nil {
		t.Fatal(err)
	}
	worldB, err := json.Marshal(after.World)
	if err != nil || !bytes.Equal(worldA, worldB) {
		t.Fatal("operator selection changed Canonical World")
	}
	newPlan := generate()
	if oldPlan.ScenarioSHA256 != newPlan.ScenarioSHA256 ||
		oldPlan.WorldSHA256 != newPlan.WorldSHA256 ||
		oldPlan.PluginLockSHA256 != newPlan.PluginLockSHA256 ||
		!reflect.DeepEqual(oldPlan.FixtureSHA256, newPlan.FixtureSHA256) ||
		oldPlan.OperatorSelectionSHA256 == newPlan.OperatorSelectionSHA256 {
		t.Fatal("selection change not independently observable from existing pins")
	}

	if output, err := run(oldPlan.Commands[0].Argv[2:]...); err == nil ||
		!strings.Contains(err.Error(), "Operator Selection fingerprint differs") ||
		output != "" {
		t.Fatalf("old generated command executed B after selection A was pinned: output=%q err=%v", output, err)
	}
	// A newly generated plan does use B, although it produces the *same*
	// independent expected result on 2. Equal results are not provenance.
	assertCalled(newPlan, b)

	planPath := filepath.Join(root, "saved-plan.json")
	raw, err := json.Marshal(oldPlan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("check", "-scenario", scenarioPath, "-workspace", root, "-plan", planPath); err == nil ||
		!strings.Contains(err.Error(), "stale or modified CLI plan") {
		t.Fatalf("old generated Plan remained accepted after selection change: %v", err)
	}
	// Removing the new pin cannot turn a generated command back into an
	// apparently fully qualified, but stale, execution.
	args := append([]string(nil), newPlan.Commands[0].Argv[2:]...)
	for i := range args {
		if args[i] == "-operator-selection-sha256" {
			args = append(args[:i], args[i+2:]...)
			break
		}
	}
	if _, err := run(args...); err == nil ||
		!strings.Contains(err.Error(), "requires complete Scenario, World, Plugin Lock and Operator Selection pins") {
		t.Fatalf("generated command accepted missing Operator Selection pin: %v", err)
	}
}

// Order is not part of the selected Operator contract. A genuine identity
// change is, and nil and an explicitly empty section mean the same selection.
func TestScenarioOperatorSelectionFingerprintIgnoresManifestOrder(t *testing.T) {
	a := []workspace.OperatorImplementationSelection{
		{Operator: "sales.first", Implementation: "example.alpha"},
		{Operator: "sales.second", Implementation: "example.beta"},
	}
	b := []workspace.OperatorImplementationSelection{a[1], a[0]}
	first, err := fingerprintScenarioOperatorSelection(a)
	if err != nil {
		t.Fatal(err)
	}
	reordered, err := fingerprintScenarioOperatorSelection(b)
	if err != nil || first != reordered {
		t.Fatalf("reordered but equivalent selections changed fingerprint: %q vs %q err=%v", first, reordered, err)
	}
	b[0].Implementation = "example.gamma"
	changed, err := fingerprintScenarioOperatorSelection(b)
	if err != nil || first == changed {
		t.Fatalf("changed selected implementation kept fingerprint: %q vs %q err=%v", first, changed, err)
	}
	none, err := fingerprintScenarioOperatorSelection(nil)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := fingerprintScenarioOperatorSelection([]workspace.OperatorImplementationSelection{})
	if err != nil || none != empty {
		t.Fatalf("nil and empty selections differ: %q vs %q err=%v", none, empty, err)
	}
}
