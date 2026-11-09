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

// These fixed expectations are characterization anchors, not an independent
// business Oracle for other AI-authored candidate files.
var aiScenarioAnchors = map[string]struct {
	fixture string
	rule    string
	want    string
}{
	"ai-validation-positive": {"fixtures/order_valid.json", "sales.order_line.quantity_positive", "satisfied"},
	"ai-validation-denied":   {"fixtures/order_invalid.json", "sales.order_line.quantity_positive", "not-satisfied"},
}

// TestAIAddedScenarioCandidateFiles reuses the existing Scenario parser and CLI
// projection. Adding a new ai-candidate-*.json needs no new Go test code.
func TestAIAddedScenarioCandidateFiles(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "driver-scenarios")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	paths, err := filepath.Glob(filepath.Join(root, "ai-candidate-*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no AI Scenario candidate files: %v", err)
	}
	seenScenarios := map[string]bool{}
	seenSteps := map[string]bool{}
	foundAnchors := map[string]bool{}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		scenario, err := scenariotest.Parse(source)
		if err != nil {
			t.Fatalf("%s: invalid candidate: %v", path, err)
		}
		if seenScenarios[scenario.ID] {
			t.Fatalf("duplicate candidate Scenario ID %q", scenario.ID)
		}
		seenScenarios[scenario.ID] = true
		for _, step := range scenario.Steps {
			if seenSteps[step.ID] {
				t.Fatalf("duplicate candidate Step ID %q", step.ID)
			}
			seenSteps[step.ID] = true
			if anchor, ok := aiScenarioAnchors[step.ID]; ok {
				if step.Operation != "evaluate-validations" || step.Fixture != anchor.fixture ||
					step.ExpectValidations[anchor.rule] != anchor.want {
					t.Fatalf("candidate %q changed fixed anchor %q", scenario.ID, step.ID)
				}
				foundAnchors[step.ID] = true
			}
		}
		t.Run(scenario.ID, func(t *testing.T) {
			run := func(action string, extras ...string) ([]byte, error) {
				t.Helper()
				args := append([]string{action, "-scenario", path, "-workspace", workspace}, extras...)
				var output bytes.Buffer
				err := runScenarioCommand(args, &output)
				return output.Bytes(), err
			}
			if _, err := run("verify"); err != nil {
				t.Fatalf("candidate verify: %v", err)
			}
			generated, err := run("generate")
			if err != nil {
				t.Fatalf("candidate generate: %v", err)
			}
			var plan scenariotest.Plan
			if err := json.Unmarshal(generated, &plan); err != nil {
				t.Fatal(err)
			}
			if len(plan.Commands) != len(scenario.Steps) {
				t.Fatalf("candidate Scenario lost steps: %d of %d", len(plan.Commands), len(scenario.Steps))
			}
			planPath := filepath.Join(t.TempDir(), "plan.json")
			if err := os.WriteFile(planPath, generated, 0600); err != nil {
				t.Fatal(err)
			}
			check, err := run("check", "-plan", planPath)
			if err != nil || !bytes.Contains(check, []byte(`"planMatches":true`)) {
				t.Fatalf("candidate plan check failed: %v: %s", err, check)
			}
			for i, command := range plan.Commands {
				if command.StepID != scenario.Steps[i].ID ||
					len(command.Argv) < 3 || command.Argv[0] != "tei" ||
					command.Argv[1] != "scenario" {
					t.Fatalf("generated command does not match candidate Step %d: %#v", i, command)
				}
				var actual bytes.Buffer
				if err := runScenarioCommand(command.Argv[2:], &actual); err != nil {
					t.Fatalf("pinned candidate Step %q: %v", command.StepID, err)
				}
				var receipt struct {
					ExecutionMode string `json:"executionMode"`
					Evidence      []struct {
						StepID   string `json:"stepId"`
						Verified bool   `json:"verified"`
						Nature   string `json:"evidenceNature"`
					} `json:"evidence"`
				}
				if err := json.Unmarshal(actual.Bytes(), &receipt); err != nil {
					t.Fatal(err)
				}
				if receipt.ExecutionMode != "generated-pinned" || len(receipt.Evidence) != 1 ||
					receipt.Evidence[0].StepID != command.StepID || !receipt.Evidence[0].Verified {
					t.Fatalf("candidate Step %q lacks pinned limited evidence: %s", command.StepID, actual.String())
				}
				if scenario.Steps[i].Operation == "evaluate-validations" &&
					receipt.Evidence[0].Nature != "canonical-validations-synthetic-input-evaluated" {
					t.Fatalf("candidate Step %q overstated its evidence: %s", command.StepID, actual.String())
				}
			}
		})
	}
	for id := range aiScenarioAnchors {
		if !foundAnchors[id] {
			t.Fatalf("missing fixed AI Scenario anchor %q", id)
		}
	}
}

// If both the candidate's input and proposed Expected are malleable, a local
// PASS alone says nothing about Oracle independence. At least the pinned
// baseline cannot be silently rewritten to accept a wrong Validation outcome.
func TestAIScenarioCandidateRejectsWrongExpectedAndOldPlan(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "driver-scenarios")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	reference := filepath.Join(root, "ai-candidate-order-validation.json")
	scenarioPath, _ := scenarioFixtureCopy(t, reference)
	var generated bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", scenarioPath, "-workspace", workspace}, &generated); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(t.TempDir(), "candidate-plan.json")
	if err := os.WriteFile(planPath, generated.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(scenarioPath)
	if err != nil {
		t.Fatal(err)
	}
	s, err := scenariotest.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for i := range s.Steps {
		if s.Steps[i].ID == "ai-validation-positive" {
			s.Steps[i].ExpectValidations["sales.order_line.quantity_positive"] = "not-satisfied"
		}
	}
	mutated, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scenarioPath, mutated, 0600); err != nil {
		t.Fatal(err)
	}
	var check bytes.Buffer
	if err := runScenarioCommand([]string{"check", "-scenario", scenarioPath, "-workspace", workspace, "-plan", planPath}, &check); err == nil {
		t.Fatal("stale generated plan accepted changed Candidate Expected")
	}
	var result bytes.Buffer
	err = runScenarioCommand([]string{"run", "-scenario", scenarioPath, "-workspace", workspace, "-step", "ai-validation-positive"}, &result)
	if err == nil {
		t.Fatalf("wrong Candidate Expected passed despite fixed actual Validation: %s", result.String())
	}
	if strings.Contains(err.Error(), "unknown or unselected") {
		t.Fatalf("wrong Expected negative never reached the Validation: %v", err)
	}
}
