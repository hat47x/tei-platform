package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/scenariotest"
)

func TestScenarioCLIReferenceRoundTrip(t *testing.T) {
	scenarioPath := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspacePath := filepath.Join("..", "..", "reference", "workspace")
	run := func(action string, extra ...string) (string, error) {
		var out bytes.Buffer
		args := append([]string{action, "-scenario", scenarioPath, "-workspace", workspacePath}, extra...)
		err := runScenarioCommand(args, &out)
		return out.String(), err
	}
	if _, err := run("verify"); err != nil {
		t.Fatal(err)
	}
	generated, err := run("generate")
	if err != nil {
		t.Fatal(err)
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal([]byte(generated), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != 2 || plan.Commands[0].StepID != "browse-pattern" || plan.Commands[1].StepID != "record-pattern" {
		t.Fatalf("missing command bindings: %#v", plan.Commands)
	}
	planPath := filepath.Join(t.TempDir(), "commands.json")
	if err := os.WriteFile(planPath, []byte(generated), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("check", "-plan", planPath); err != nil {
		t.Fatal(err)
	}
	if _, err := run("run", "-step", "browse-pattern"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("run", "-step", "record-pattern"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("run", "-step", "missing"); err == nil {
		t.Fatal("unknown step accepted")
	}
	var tampered scenariotest.Plan
	if err := json.Unmarshal([]byte(generated), &tampered); err != nil {
		t.Fatal(err)
	}
	tampered.Commands[0].Argv[0] = "unauthorized"
	b, err := json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("check", "-plan", planPath); err == nil {
		t.Fatal("modified CLI plan accepted")
	}
	source, err := os.ReadFile(scenarioPath)
	if err != nil {
		t.Fatal(err)
	}
	newScenarioPath := filepath.Join(t.TempDir(), "changed.json")
	if err := os.WriteFile(newScenarioPath, source, 0600); err != nil {
		t.Fatal(err)
	}
	var originalPlan bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", newScenarioPath, "-workspace", workspacePath}, &originalPlan); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, originalPlan.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runScenarioCheckForTest(newScenarioPath, workspacePath, planPath); err != nil {
		t.Fatal(err)
	}
	modified := strings.Replace(string(source), "order-entry-browse-pattern-smoke", "order-entry-browse-pattern-revised", 1)
	if err := os.WriteFile(newScenarioPath, []byte(modified), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runScenarioCheckForTest(newScenarioPath, workspacePath, planPath); err == nil {
		t.Fatal("stale source Scenario plan accepted")
	}
}

func runScenarioCheckForTest(path, workspace, plan string) (string, error) {
	var out bytes.Buffer
	err := runScenarioCommand([]string{"check", "-scenario", path, "-workspace", workspace, "-plan", plan}, &out)
	return out.String(), err
}
