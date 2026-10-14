package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"tei/internal/modelref"
	"tei/internal/scenariotest"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// ScenarioPlanBindingReport attaches an existing persisted Driver Scenario and
// generated CLI Plan to a two-Workspace Source revision review. This is a
// read-only, static candidate; it never produces a qualification receipt.
type ScenarioPlanBindingReport struct {
	SchemaVersion               string              `json:"schemaVersion"`
	Authority                   string              `json:"authority"`
	Diff                        WorkspaceDiffReport `json:"diff"`
	ScenarioID                  string              `json:"scenarioId"`
	ScenarioSHA256              string              `json:"scenarioSha256"`
	FixtureSHA256               map[string]string   `json:"fixtureSha256"`
	ObservedAfterSourceRevision string              `json:"observedAfterSourceRevision"`
	PlanPinsSourceRevision      bool                `json:"planPinsSourceRevision"`
	PlanMatches                 bool                `json:"planMatches"`
	ScenarioStepCount           int                 `json:"scenarioStepCount"`
	SyntheticFixtureStepCount   int                 `json:"syntheticFixtureStepCount"`
	Qualification               string              `json:"qualification"`
	ReasonCodes                 []string            `json:"reasonCodes"`
	ExecutionAllowed            bool                `json:"executionAllowed"`
	AuthorityLimits             []string            `json:"authorityLimits"`
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// readScenarioLocal only reads regular files under a pinned Scenario
// directory, preventing fixture symlinks from escaping that directory.
func readScenarioLocal(root *os.Root, filename string) ([]byte, error) {
	if !filepath.IsLocal(filename) || filepath.Clean(filename) != filename {
		return nil, fmt.Errorf("Scenario-local path is not normalized or escapes root: %q", filename)
	}
	file, err := root.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Scenario-local path is not a regular file: %q", filename)
	}
	return io.ReadAll(file)
}

// buildExpectedScenarioPlan uses the existing scenariotest generator contract
// and reads existing scenario/fixture source bytes. It is not a replacement
// for the tei scenario check/run command or for an independent oracle.
func buildExpectedScenarioPlan(scenarioPath, workspacePath string, loaded workspace.Result) (scenariotest.Plan, scenariotest.Scenario, error) {
	var empty scenariotest.Plan
	root, err := os.OpenRoot(filepath.Dir(scenarioPath))
	if err != nil {
		return empty, scenariotest.Scenario{}, err
	}
	defer root.Close()
	source, err := readScenarioLocal(root, filepath.Base(scenarioPath))
	if err != nil {
		return empty, scenariotest.Scenario{}, fmt.Errorf("read Driver Scenario: %w", err)
	}
	scenario, err := scenariotest.Parse(source)
	if err != nil {
		return empty, scenariotest.Scenario{}, err
	}
	plan, err := scenariotest.Generate(scenario, source, scenarioPath, workspacePath)
	if err != nil {
		return empty, scenariotest.Scenario{}, err
	}
	digests := make(map[string]string)
	for _, step := range scenario.Steps {
		if step.Fixture == "" {
			continue
		}
		data, err := readScenarioLocal(root, step.Fixture)
		if err != nil {
			return empty, scenariotest.Scenario{}, fmt.Errorf("step %s fixture: %w", step.ID, err)
		}
		switch step.Operation {
		case "execute-browse-query":
			fixture, err := scenariotest.ParseBrowseFixture(data)
			if err != nil {
				return empty, scenariotest.Scenario{}, fmt.Errorf("step %s: %w", step.ID, err)
			}
			if fixture.Query != step.ExpectQuery || !reflect.DeepEqual(fixture.Parameters, step.Parameters) {
				return empty, scenariotest.Scenario{}, fmt.Errorf("step %s: Scenario/Fixture query or parameters differ", step.ID)
			}
		case "evaluate-validations":
			if _, err := scenariotest.ParseValidationFixture(data); err != nil {
				return empty, scenariotest.Scenario{}, fmt.Errorf("step %s: %w", step.ID, err)
			}
		case "observe-synthetic-state":
			fixture, err := scenariotest.ParseObservationFixture(data)
			if err != nil {
				return empty, scenariotest.Scenario{}, fmt.Errorf("step %s: %w", step.ID, err)
			}
			if fixture.ReadIdentity != step.ReadIdentity {
				return empty, scenariotest.Scenario{}, fmt.Errorf("step %s: observation identity differs", step.ID)
			}
		default:
			return empty, scenariotest.Scenario{}, fmt.Errorf("step %s: unsupported fixture-bearing operation %q", step.ID, step.Operation)
		}
		digests[step.ID] = sha256Hex(data)
	}
	if len(digests) > 0 {
		plan.FixtureSHA256 = digests
	}
	worldBytes, err := json.Marshal(loaded.World)
	if err != nil {
		return empty, scenariotest.Scenario{}, err
	}
	plan.WorldSHA256 = sha256Hex(worldBytes)
	lock, err := os.ReadFile(filepath.Join(workspacePath, ".tei", "plugin-lock.json"))
	if err != nil {
		return empty, scenariotest.Scenario{}, fmt.Errorf("read current Plugin Lock: %w", err)
	}
	plan.PluginLockSHA256 = sha256Hex(lock)
	for i := range plan.Commands {
		command := &plan.Commands[i]
		command.Argv = append(command.Argv,
			"-world-sha256", plan.WorldSHA256,
			"-plugin-lock-sha256", plan.PluginLockSHA256)
		if digest, exists := plan.FixtureSHA256[command.StepID]; exists {
			command.Argv = append(command.Argv, "-fixture-sha256", digest)
		}
	}
	return plan, scenario, nil
}

// reviewScenarioPlanDiff verifies that a previously generated, persisted CLI
// plan still matches the current Scenario, Fixture, World and Plugin Lock,
// while retaining the prior two-snapshot Source Index review. Matching pins
// do NOT prove the Scenario's independent expectations, actual execution,
// Evidence Packet trust, original Source authenticity or runtime authority.
func reviewScenarioPlanDiff(beforeRoot, afterRoot, scope string, consumer modelref.Key, scenarioPath, planPath string) (ScenarioPlanBindingReport, error) {
	if scenarioPath == "" || planPath == "" || consumer.Subject.Kind != "application" {
		return ScenarioPlanBindingReport{}, fmt.Errorf("Scenario, Plan and application consumer are required")
	}
	diff, err := reviewWorkspaceDiff(beforeRoot, afterRoot, scope, consumer)
	if err != nil {
		return ScenarioPlanBindingReport{}, err
	}
	loaded, err := workspace.Load(afterRoot)
	if err != nil {
		return ScenarioPlanBindingReport{}, fmt.Errorf("reload after Workspace: %w", err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(afterRoot, loaded)
	if err != nil {
		return ScenarioPlanBindingReport{}, fmt.Errorf("reindex after Workspace: %w", err)
	}
	currentRevision := sourceindex.ModelRevision(sourceIdx)
	if currentRevision != diff.AfterSourceRevision {
		return ScenarioPlanBindingReport{}, fmt.Errorf("after Workspace source revision changed between reviews")
	}
	expected, scenario, err := buildExpectedScenarioPlan(scenarioPath, afterRoot, loaded)
	if err != nil {
		return ScenarioPlanBindingReport{}, err
	}
	if scenario.Application != consumer.Subject.ID {
		return ScenarioPlanBindingReport{}, fmt.Errorf("Scenario application %q differs from typed consumer %q", scenario.Application, consumer.Subject.ID)
	}
	persisted, err := os.ReadFile(planPath)
	if err != nil {
		return ScenarioPlanBindingReport{}, fmt.Errorf("read generated CLI Plan: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(persisted))
	decoder.DisallowUnknownFields()
	var previous scenariotest.Plan
	if err := decoder.Decode(&previous); err != nil {
		return ScenarioPlanBindingReport{}, fmt.Errorf("decode generated CLI Plan: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return ScenarioPlanBindingReport{}, fmt.Errorf("generated CLI Plan has trailing content")
	}
	if !reflect.DeepEqual(expected, previous) {
		return ScenarioPlanBindingReport{}, fmt.Errorf("generated CLI Plan is stale or modified: use tei scenario generate/check")
	}
	fixtures := make(map[string]string, len(expected.FixtureSHA256))
	for step, digest := range expected.FixtureSHA256 {
		fixtures[step] = digest
	}
	return ScenarioPlanBindingReport{
		SchemaVersion:               "tei.reference.scenario-source-binding/v0",
		Authority:                   "read-only-scenario-plan-pins-no-evidence-qualification",
		Diff:                        diff,
		ScenarioID:                  scenario.ID,
		ScenarioSHA256:              expected.ScenarioSHA256,
		FixtureSHA256:               fixtures,
		ObservedAfterSourceRevision: currentRevision,
		PlanPinsSourceRevision:      false,
		PlanMatches:                 true,
		ScenarioStepCount:           len(scenario.Steps),
		SyntheticFixtureStepCount:   len(fixtures),
		Qualification:               "unresolved",
		ReasonCodes:                 []string{"cli-plan-has-no-source-revision-pin", "no-independent-oracle-provenance", "no-execution-or-evidence-packet", "no-runtime-authority"},
		ExecutionAllowed:            false,
		AuthorityLimits: []string{
			"separate-loads-are-not-atomic-workspace-snapshots",
			"cli-plan-matching-is-not-executed-scenario-evidence",
			"scenario-expectations-and-fixtures-do-not-establish-oracle-independence",
			"no-accepted-source-revision-target-provider-or-runtime-authority",
		},
	}, nil
}
