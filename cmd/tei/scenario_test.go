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

func scenarioFixtureCopy(t *testing.T, scenarioPath string) (string, string) {
	t.Helper()
	source, err := os.ReadFile(scenarioPath)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "fixtures"), 0700); err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(dir, "scenario.json")
	if err := os.WriteFile(newPath, source, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"recent_orders.json", "order_valid.json", "order_invalid.json", "order_state_pending.json"} {
		fixtureSource, err := os.ReadFile(filepath.Join(filepath.Dir(scenarioPath), "fixtures", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "fixtures", name), fixtureSource, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return newPath, filepath.Join(dir, "fixtures", "recent_orders.json")
}

func runScenarioCheckForTest(path, workspace, plan string) (string, error) {
	var out bytes.Buffer
	err := runScenarioCommand([]string{"check", "-scenario", path, "-workspace", workspace, "-plan", plan}, &out)
	return out.String(), err
}

// TestRegenerateScenarioArgvAndStdinPlan exercises every generated command from
// the persisted six-step Scenario. It never substitutes a hand-authored
// "scenario run" shortcut for argv generation or claims real provider input.
func TestRegenerateScenarioArgvAndStdinPlan(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	scenarioPath, _ := scenarioFixtureCopy(t, reference)
	fixtureDir := filepath.Dir(reference)
	readStdin := func(name string) scenariotest.StdinPlan {
		t.Helper()
		source, err := os.ReadFile(filepath.Join(fixtureDir, "fixtures", name))
		if err != nil {
			t.Fatal(err)
		}
		var stdin scenariotest.StdinPlan
		if err := json.Unmarshal(source, &stdin); err != nil {
			t.Fatal(err)
		}
		return stdin
	}
	positiveStdin := readStdin("cli_stdin_none.json")
	negativeStdin := readStdin("cli_stdin_unsupported.json")
	if err := scenariotest.ValidateStdinPlan(positiveStdin); err != nil {
		t.Fatalf("positive transport fixture invalid: %v", err)
	}
	if err := scenariotest.ValidateStdinPlan(negativeStdin); err == nil {
		t.Fatal("unsupported stdin fixture was accepted")
	}

	run := func(action string, extra ...string) (string, error) {
		var output bytes.Buffer
		args := append([]string{action, "-scenario", scenarioPath, "-workspace", workspace}, extra...)
		err := runScenarioCommand(args, &output)
		return output.String(), err
	}
	firstJSON, err := run("generate")
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := run("generate")
	if err != nil || firstJSON != secondJSON {
		t.Fatalf("same Scenario did not regenerate deterministic CLI plan: %v", err)
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal([]byte(firstJSON), &plan); err != nil {
		t.Fatal(err)
	}
	persistedSource, err := os.ReadFile(scenarioPath)
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := scenariotest.Parse(persistedSource)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != len(scenario.Steps) || len(plan.Commands) != 6 {
		t.Fatalf("Scenario steps missing in generated argv/stdin plan: %#v", plan.Commands)
	}
	for i, command := range plan.Commands {
		if command.StepID != scenario.Steps[i].ID || command.Stdin != positiveStdin {
			t.Fatalf("generated Step %d has wrong binding or stdin: %#v", i, command)
		}
		if len(command.Argv) < 3 || command.Argv[0] != "tei" || command.Argv[1] != "scenario" {
			t.Fatalf("CLI argv was not generated for Step %s: %q", command.StepID, command.Argv)
		}
		if err := scenariotest.ValidateStdinPlan(command.Stdin); err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if err := runScenarioCommand(command.Argv[2:], &output); err != nil {
			t.Fatalf("generated argv for Step %s not executable: %v", command.StepID, err)
		}
		var result struct {
			ExecutionMode string `json:"executionMode"`
			Evidence      []struct {
				StepID   string `json:"stepId"`
				Verified bool   `json:"verified"`
			} `json:"evidence"`
		}
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.ExecutionMode != "generated-pinned" || len(result.Evidence) != 1 ||
			result.Evidence[0].StepID != command.StepID || !result.Evidence[0].Verified {
			t.Fatalf("generated command Step %s did not verify: %s", command.StepID, output.String())
		}
	}

	planPath := filepath.Join(t.TempDir(), "generated-cli-plan.json")
	savePlan := func(p scenariotest.Plan) {
		t.Helper()
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(planPath, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	savePlan(plan)
	if _, err := run("check", "-plan", planPath); err != nil {
		t.Fatalf("generated CLI plan did not pass provenance check: %v", err)
	}

	// One negative fixture changes only the declared input transport.
	// A CLI run cannot silently claim that arbitrary stdin was consumed.
	invalid := plan
	invalid.Commands = append([]scenariotest.Command(nil), plan.Commands...)
	invalid.Commands[2].Stdin = negativeStdin
	savePlan(invalid)
	if _, err := run("check", "-plan", planPath); err == nil ||
		!strings.Contains(err.Error(), "unsupported CLI stdin plan") {
		t.Fatalf("unimplemented inline stdin was accepted: %v", err)
	}
	savePlan(plan)

	// Revision changes must invalidate the previous generated argv and plan.
	changed := strings.Replace(string(persistedSource),
		"order-entry-browse-pattern-smoke", "order-entry-browse-pattern-smoke-revised", 1)
	if changed == string(persistedSource) {
		t.Fatal("Scenario identity mutation missed fixture")
	}
	if err := os.WriteFile(scenarioPath, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	var staleOutput bytes.Buffer
	if err := runScenarioCommand(plan.Commands[2].Argv[2:], &staleOutput); err == nil ||
		!strings.Contains(err.Error(), "Scenario fingerprint differs") {
		t.Fatalf("old generated argv accepted against updated Scenario: %v", err)
	}
	if _, err := run("check", "-plan", planPath); err == nil {
		t.Fatal("old generated argv/stdin plan accepted after Scenario changed")
	}
	regenerated, err := run("generate")
	if err != nil {
		t.Fatal(err)
	}
	var newPlan scenariotest.Plan
	if err := json.Unmarshal([]byte(regenerated), &newPlan); err != nil {
		t.Fatal(err)
	}
	if newPlan.ScenarioSHA256 == plan.ScenarioSHA256 ||
		len(newPlan.Commands) != len(plan.Commands) ||
		newPlan.Commands[2].Stdin != positiveStdin {
		t.Fatal("Scenario change was not reflected in regenerated argv/stdin")
	}
	savePlan(newPlan)
	if _, err := run("check", "-plan", planPath); err != nil {
		t.Fatalf("new generated argv/stdin plan failed: %v", err)
	}
}

// TestScenarioCLIRejectsShadowedUnsupportedOperation enforces the saved
// Scenario as an unambiguous source before any CLI plan or Driver execution.
// encoding/json otherwise silently lets the final duplicate key win.
func TestScenarioCLIRejectsShadowedUnsupportedOperation(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "driver-scenarios")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	accepted := filepath.Join(root, "order-entry-pattern-smoke.json")
	ambiguous := filepath.Join(root, "fixtures", "scenario_duplicate_step_operation.json")

	var positive bytes.Buffer
	if err := runScenarioCommand([]string{
		"generate", "-scenario", accepted, "-workspace", workspace,
	}, &positive); err != nil {
		t.Fatalf("existing unambiguous Scenario failed to generate: %v", err)
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal(positive.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != 6 {
		t.Fatalf("positive Scenario lost generated Steps: %d", len(plan.Commands))
	}

	source, err := os.ReadFile(ambiguous)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scenariotest.Parse(source); err == nil ||
		!strings.Contains(err.Error(), "duplicate JSON member") {
		t.Fatalf("ambiguous persisted Scenario was accepted by parser: %v", err)
	}
	for _, action := range []string{"verify", "generate", "run"} {
		t.Run(action, func(t *testing.T) {
			args := []string{action, "-scenario", ambiguous, "-workspace", workspace}
			if action == "run" {
				args = append(args, "-step", "browse-pattern")
			}
			var output bytes.Buffer
			if err := runScenarioCommand(args, &output); err == nil ||
				!strings.Contains(err.Error(), "duplicate JSON member") {
				t.Fatalf("%s accepted shadowed unsupported operation: %v", action, err)
			}
			if output.Len() != 0 {
				t.Fatalf("%s emitted a success artifact from ambiguous Scenario: %s", action, output.String())
			}
		})
	}
}

// TestScenarioCheckRejectsShadowedGeneratedPlanMembers checks the boundary
// between a freshly generated CLI plan and a separately saved derived JSON.
// The last of duplicate JSON keys would otherwise win in encoding/json,
// potentially making an edited artifact pass the semantic DeepEqual check.
func TestScenarioCheckRejectsShadowedGeneratedPlanMembers(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	var generated bytes.Buffer
	if err := runScenarioCommand([]string{
		"generate", "-scenario", reference, "-workspace", workspace,
	}, &generated); err != nil {
		t.Fatal(err)
	}
	var parsed scenariotest.Plan
	if err := json.Unmarshal(generated.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Commands) != 6 || parsed.ScenarioID == "" {
		t.Fatalf("generated CLI plan is incomplete: %#v", parsed)
	}

	planPath := filepath.Join(t.TempDir(), "cli-plan.json")
	check := func(content []byte) (string, error) {
		t.Helper()
		if err := os.WriteFile(planPath, content, 0600); err != nil {
			t.Fatal(err)
		}
		return runScenarioCheckForTest(reference, workspace, planPath)
	}

	if output, err := check(generated.Bytes()); err != nil ||
		!strings.Contains(output, `"planMatches":true`) {
		t.Fatalf("unchanged derived CLI plan was rejected: %v output=%s", err, output)
	}

	// An earlier spoofed scenarioId must not be ignored just because the
	// original generator-issued scenarioId occurs later in the object.
	spoofed := []byte(`{"scenarioId":"spoofed",` +
		strings.TrimPrefix(strings.TrimSpace(generated.String()), "{"))
	if output, err := check(spoofed); err == nil ||
		!strings.Contains(err.Error(), `duplicate JSON member "scenarioId"`) ||
		output != "" {
		t.Fatalf("duplicate top-level Scenario identity accepted: %v output=%s", err, output)
	}

	// The same rule also applies inside generated command objects, e.g. an
	// earlier forged stdin mode shadowed by the normal explicit none mode.
	original := `"stdin":{"mode":"none"}`
	replacement := `"stdin":{"mode":"inline","mode":"none"}`
	if !strings.Contains(generated.String(), original) {
		t.Fatalf("missing expected generated stdin plan: %s", generated.String())
	}
	nested := []byte(strings.Replace(generated.String(), original, replacement, 1))
	if output, err := check(nested); err == nil ||
		!strings.Contains(err.Error(), `duplicate JSON member "mode"`) ||
		output != "" {
		t.Fatalf("duplicate command stdin transport accepted: %v output=%s", err, output)
	}
}

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
	if len(plan.Commands) != 6 || plan.Commands[0].StepID != "browse-pattern" || plan.Commands[1].StepID != "record-pattern" || plan.Commands[2].StepID != "browse-synthetic" || plan.Commands[3].StepID != "validate-positive" || plan.Commands[4].StepID != "validate-denied" || plan.Commands[5].StepID != "observe-synthetic" {
		t.Fatalf("missing command bindings: %#v", plan.Commands)
	}
	if len(plan.WorldSHA256) != 64 || len(plan.PluginLockSHA256) != 64 || len(plan.FixtureSHA256["browse-synthetic"]) != 64 || len(plan.FixtureSHA256["validate-positive"]) != 64 || len(plan.FixtureSHA256["validate-denied"]) != 64 || len(plan.FixtureSHA256["observe-synthetic"]) != 64 {
		t.Fatalf("missing independently bound hashes: %#v", plan)
	}
	var pinnedOutput bytes.Buffer
	if err := runScenarioCommand(plan.Commands[2].Argv[2:], &pinnedOutput); err != nil {
		t.Fatalf("generated pinned argv could not be executed: %v", err)
	}
	if !strings.Contains(pinnedOutput.String(), "synthetic-browse-query-executed") {
		t.Fatalf("generated pinned command did not execute synthetic Browse: %s", pinnedOutput.String())
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
	output, err := run("run", "-step", "browse-synthetic")
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Evidence []scenarioStepEvidence `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) != 1 || !result.Evidence[0].Verified || result.Evidence[0].RowCount != 1 || result.Evidence[0].EvidenceNature != "synthetic-browse-query-executed" {
		t.Fatalf("incorrect synthetic result evidence: %s", output)
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
}

func TestScenarioCLIRejectsStaleFixtureAndWrongBusinessResult(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	scenarioPath, fixturePath := scenarioFixtureCopy(t, reference)
	run := func(action string, extra ...string) (string, error) {
		var out bytes.Buffer
		args := append([]string{action, "-scenario", scenarioPath, "-workspace", workspace}, extra...)
		err := runScenarioCommand(args, &out)
		return out.String(), err
	}
	var generated bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", scenarioPath, "-workspace", workspace}, &generated); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(planPath, generated.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("check", "-plan", planPath); err != nil {
		t.Fatal(err)
	}
	fixtureSource, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(fixtureSource), "\"OPEN\"", "\"REJECTED\"", 1)
	if tampered == string(fixtureSource) {
		t.Fatal("fixture mutation has no effect")
	}
	if err := os.WriteFile(fixturePath, []byte(tampered), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("check", "-plan", planPath); err == nil {
		t.Fatal("changed fixture incorrectly accepted")
	}
	var persisted scenariotest.Plan
	if err := json.Unmarshal(generated.Bytes(), &persisted); err != nil {
		t.Fatal(err)
	}
	var pinnedOutput bytes.Buffer
	if err := runScenarioCommand(persisted.Commands[2].Argv[2:], &pinnedOutput); err == nil {
		t.Fatal("stale generated argv accepted after fixture mutation")
	}
	if _, err := run("run", "-step", "browse-synthetic"); err == nil {
		t.Fatal("wrong mapped status incorrectly passed Scenario expectation")
	}
	if err := os.WriteFile(fixturePath, fixtureSource, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("check", "-plan", planPath); err != nil {
		t.Fatal(err)
	}
	scenarioSource, err := os.ReadFile(scenarioPath)
	if err != nil {
		t.Fatal(err)
	}
	modified := strings.Replace(string(scenarioSource), "order-entry-browse-pattern-smoke", "order-entry-revised", 1)
	if err := os.WriteFile(scenarioPath, []byte(modified), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("check", "-plan", planPath); err == nil {
		t.Fatal("changed Scenario incorrectly accepted")
	}
	var pinnedOutput2 bytes.Buffer
	if err := runScenarioCommand(persisted.Commands[2].Argv[2:], &pinnedOutput2); err == nil {
		t.Fatal("stale generated argv accepted after Scenario mutation")
	}
}

func TestScenarioCLIRejectsFixtureTraversal(t *testing.T) {
	for _, name := range []string{"../outside.json", "/etc/passwd"} {
		if _, err := scenarioFixturePath("scenario.json", name); err == nil {
			t.Fatalf("accepted escaping fixture %q", name)
		}
	}
}

func TestScenarioCLIRejectsUnmappedRawColumnAndWrongParameters(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	scenarioPath, fixturePath := scenarioFixtureCopy(t, reference)
	run := func() error {
		var out bytes.Buffer
		return runScenarioCommand([]string{"run", "-scenario", scenarioPath, "-workspace", workspace, "-step", "browse-synthetic"}, &out)
	}
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	unmapped := strings.Replace(string(fixture), `"customer_id"`, `"unexpected_customer_column"`, 1)
	if unmapped == string(fixture) {
		t.Fatal("fixture column mutation has no effect")
	}
	if err := os.WriteFile(fixturePath, []byte(unmapped), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil {
		t.Fatal("unmapped provider column was accepted by existing Driver")
	}
	if err := os.WriteFile(fixturePath, fixture, 0600); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(scenarioPath)
	if err != nil {
		t.Fatal(err)
	}
	wrongParameter := strings.Replace(string(source), "2026-09-01T00:00:00Z", "2026-09-03T00:00:00Z", 1)
	if wrongParameter == string(source) {
		t.Fatal("parameter mutation has no effect")
	}
	if err := os.WriteFile(scenarioPath, []byte(wrongParameter), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil {
		t.Fatal("Query fixture parameter mismatch was accepted")
	}
}

func TestScenarioCLIRejectsWrongGeneratedEnvironmentFingerprint(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	var generated bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", reference, "-workspace", workspace}, &generated); err != nil {
		t.Fatal(err)
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal(generated.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != 6 {
		t.Fatalf("generated commands=%d", len(plan.Commands))
	}
	for _, target := range []string{"-world-sha256", "-plugin-lock-sha256", "-fixture-sha256"} {
		args := append([]string(nil), plan.Commands[2].Argv[2:]...)
		matched := false
		for i := range args {
			if args[i] == target {
				args[i+1] = strings.Repeat("f", 64)
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("generated argv missing required pin %s", target)
		}
		var out bytes.Buffer
		if err := runScenarioCommand(args, &out); err == nil {
			t.Fatalf("generated run ignored modified %s", target)
		}
	}
}

func TestScenarioCanonicalValidationGeneratedCommands(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	var output bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", reference, "-workspace", workspace}, &output); err != nil {
		t.Fatal(err)
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal(output.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ id, result string }{
		{"validate-positive", "satisfied"},
		{"validate-denied", "not-satisfied"},
	} {
		var command scenariotest.Command
		for _, item := range plan.Commands {
			if item.StepID == tc.id {
				command = item
				break
			}
		}
		if len(command.Argv) == 0 {
			t.Fatalf("missing generated validation command for %s", tc.id)
		}
		var actual bytes.Buffer
		if err := runScenarioCommand(command.Argv[2:], &actual); err != nil {
			t.Fatal(err)
		}
		var receipt struct {
			Evidence []scenarioStepEvidence `json:"evidence"`
		}
		if err := json.Unmarshal(actual.Bytes(), &receipt); err != nil {
			t.Fatal(err)
		}
		if len(receipt.Evidence) != 1 || !receipt.Evidence[0].Verified ||
			receipt.Evidence[0].ValidationResults["sales.order_line.quantity_positive"] != tc.result {
			t.Fatalf("%s validation outcome mismatch: %s", tc.id, actual.String())
		}
	}
}

func TestScenarioValidationRejectsWrongResultAndMissingInput(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	path, _ := scenarioFixtureCopy(t, reference)
	run := func() error {
		var out bytes.Buffer
		return runScenarioCommand([]string{"run", "-scenario", path, "-workspace", workspace, "-step", "validate-positive"}, &out)
	}
	if err := run(); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(original), `"sales.order_line.line_number_positive": "satisfied"`, `"sales.order_line.line_number_positive": "not-satisfied"`, 1)
	if bad == string(original) {
		t.Fatal("expected-result mutation had no effect")
	}
	if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil {
		t.Fatal("mismatched Validation result passed")
	}
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	inputPath := filepath.Join(filepath.Dir(path), "fixtures", "order_valid.json")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	incomplete := strings.Replace(string(input), `"sales.order_line.quantity": "2.5",`, "", 1)
	if incomplete == string(input) {
		t.Fatal("input mutation had no effect")
	}
	if err := os.WriteFile(inputPath, []byte(incomplete), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil {
		t.Fatal("missing Validation input passed")
	}
}

func TestScenarioIndependentSyntheticObservationFromGeneratedCLI(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	var generated bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", reference, "-workspace", workspace}, &generated); err != nil {
		t.Fatal(err)
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal(generated.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	var command scenariotest.Command
	for _, item := range plan.Commands {
		if item.StepID == "observe-synthetic" {
			command = item
			break
		}
	}
	if len(command.Argv) == 0 {
		t.Fatal("Observation command missing")
	}
	var output bytes.Buffer
	if err := runScenarioCommand(command.Argv[2:], &output); err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Evidence []scenarioStepEvidence `json:"evidence"`
	}
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if len(receipt.Evidence) != 1 || !receipt.Evidence[0].Verified ||
		receipt.Evidence[0].EvidenceNature != "synthetic-independent-observation-read" ||
		receipt.Evidence[0].ObservationResult != "pending" {
		t.Fatalf("unexpected Observation receipt: %s", output.String())
	}
}
func TestScenarioObservationRejectsWrongResultAndOldFixture(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	path, _ := scenarioFixtureCopy(t, reference)
	var generated bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", path, "-workspace", workspace}, &generated); err != nil {
		t.Fatal(err)
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal(generated.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	var args []string
	for _, item := range plan.Commands {
		if item.StepID == "observe-synthetic" {
			args = item.Argv[2:]
			break
		}
	}
	if len(args) == 0 {
		t.Fatal("missing Observation argv")
	}
	read := func(argv []string) error {
		var out bytes.Buffer
		return runScenarioCommand(argv, &out)
	}
	if err := read(args); err != nil {
		t.Fatal(err)
	}
	fixturePath := filepath.Join(filepath.Dir(path), "fixtures", "order_state_pending.json")
	original, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(original), `"pending"`, `"submitted"`, 1)
	if tampered == string(original) {
		t.Fatal("Observation mutation had no effect")
	}
	if err := os.WriteFile(fixturePath, []byte(tampered), 0600); err != nil {
		t.Fatal(err)
	}
	if err := read(args); err == nil {
		t.Fatal("old pinned command accepted after Observation change")
	}
	if err := read([]string{"run", "-scenario", path, "-workspace", workspace, "-step", "observe-synthetic"}); err == nil {
		t.Fatal("independent Observation mismatch unexpectedly passed")
	}
}

func TestScenarioCLIRejectsPartialPinsAndDistinguishesUnpinnedExecution(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "order-entry-pattern-smoke.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	var generated bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", reference, "-workspace", workspace}, &generated); err != nil {
		t.Fatal(err)
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal(generated.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	args := append([]string(nil), plan.Commands[2].Argv[2:]...)
	for i := range args {
		if args[i] == "-fixture-sha256" {
			args = args[:i]
			break
		}
	}
	var discarded bytes.Buffer
	if err := runScenarioCommand(args, &discarded); err == nil {
		t.Fatal("generated command accepted without its Fixture pin")
	}
	args = append([]string(nil), plan.Commands[2].Argv[2:]...)
	for i := range args {
		if args[i] == "-world-sha256" {
			args = append(args[:i], args[i+2:]...)
			break
		}
	}
	discarded.Reset()
	if err := runScenarioCommand(args, &discarded); err == nil {
		t.Fatal("generated command accepted with incomplete World pin")
	}
	var ordinary bytes.Buffer
	if err := runScenarioCommand([]string{"run", "-scenario", reference, "-workspace", workspace, "-step", "browse-synthetic"}, &ordinary); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ordinary.String(), `"executionMode":"exploratory-unpinned"`) {
		t.Fatalf("unpinned direct invocation lacked provenance: %s", ordinary.String())
	}
	var pinned bytes.Buffer
	if err := runScenarioCommand(plan.Commands[2].Argv[2:], &pinned); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pinned.String(), `"executionMode":"generated-pinned"`) {
		t.Fatalf("generated invocation lacked pinned provenance: %s", pinned.String())
	}
}
