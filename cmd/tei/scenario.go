package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/scenariotest"
	"tei/internal/workspace"
)

type scenarioStepEvidence struct {
	StepID            string            `json:"stepId"`
	Operation         string            `json:"operation"`
	Pattern           string            `json:"pattern"`
	Verified          bool              `json:"verified"`
	EvidenceNature    string            `json:"evidenceNature"`
	RowCount          int               `json:"rowCount,omitempty"`
	RowCount          int               `json:"rowCount,omitempty"`
	ValidationResults map[string]string `json:"validationResults,omitempty"`
	ObservationResult any               `json:"observationResult,omitempty"`
}

func runScenarioCommand(args []string, output io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("scenario requires verify, generate, run, or check")
	}
	action := args[0]
	switch action {
	case "verify", "generate", "run", "check":
	default:
		return fmt.Errorf("unsupported scenario action %q", action)
	}
	flags := flag.NewFlagSet("scenario "+action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	scenarioPath := flags.String("scenario", "", "Driver Scenario JSON")
	workspacePath := flags.String("workspace", "", "TEI Workspace root")
	stepID := flags.String("step", "", "specific Scenario step for run")
	planPath := flags.String("plan", "", "generated CLI plan for check")
	expectedScenarioSHA := flags.String("scenario-sha256", "", "required Scenario revision of generated command")
	expectedWorldSHA := flags.String("world-sha256", "", "required resolved World fingerprint of generated command")
	expectedLockSHA := flags.String("plugin-lock-sha256", "", "required plugin lock fingerprint of generated command")
	expectedFixtureSHA := flags.String("fixture-sha256", "", "required synthetic Fixture fingerprint of generated command")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *scenarioPath == "" || *workspacePath == "" {
		return fmt.Errorf("scenario requires -scenario and -workspace, with no positional arguments")
	}
	if *stepID != "" && action != "run" {
		return fmt.Errorf("-step is only valid for run")
	}
	if (*planPath != "") != (action == "check") {
		return fmt.Errorf("-plan is required only for check")
	}
	if action != "run" && (*expectedScenarioSHA != "" || *expectedWorldSHA != "" || *expectedLockSHA != "" || *expectedFixtureSHA != "") {
		return fmt.Errorf("generated-command fingerprint flags are valid only for run")
	}
	pinsSupplied := *expectedScenarioSHA != "" || *expectedWorldSHA != "" || *expectedLockSHA != "" || *expectedFixtureSHA != ""
	if action == "run" && pinsSupplied && (*stepID == "" || *expectedScenarioSHA == "" || *expectedWorldSHA == "" || *expectedLockSHA == "") {
		return fmt.Errorf("generated Scenario run requires complete Scenario, World, Plugin Lock pins and one Step")
	}
	source, err := os.ReadFile(*scenarioPath)
	if err != nil {
		return err
	}
	s, err := scenariotest.Parse(source)
	if err != nil {
		return err
	}
	loaded, err := workspace.Load(*workspacePath)
	if err != nil {
		return err
	}
	applicationID, err := core.ParseCanonicalID(s.Application)
	if err != nil {
		return err
	}
	resolver := driver.NewResolver(loaded.World)
	evidence := make([]scenarioStepEvidence, 0, len(s.Steps))
	selected := false
	fixtureDigests := make(map[string]string)
	resolvedFixtures := make(map[string]scenariotest.BrowseFixture)
	validationFixtures := make(map[string]scenariotest.ValidationFixture)
	observationFixtures := make(map[string]scenariotest.ObservationFixture)
	stepsByID := make(map[string]scenariotest.Step)
	for _, st := range s.Steps {
		if st.Operation == "observe-synthetic-state" {
			path, err := scenarioFixturePath(*scenarioPath, st.Fixture)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			fixture, err := scenariotest.ParseObservationFixture(source)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			if fixture.ReadIdentity != st.ReadIdentity {
				return fmt.Errorf("step %s: Observation identity mismatch", st.ID)
			}
			digest := sha256.Sum256(source)
			fixtureDigests[st.ID] = hex.EncodeToString(digest[:])
			observationFixtures[st.ID] = fixture
			if *stepID != "" && *stepID != st.ID {
				continue
			}
			selected = true
			stepsByID[st.ID] = st
			evidence = append(evidence, scenarioStepEvidence{StepID: st.ID, Operation: st.Operation, Verified: false, EvidenceNature: "synthetic-observation-bound; not-read"})
			continue
		}
		if st.Operation == "evaluate-validations" {
			path, err := scenarioFixturePath(*scenarioPath, st.Fixture)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			fixture, err := scenariotest.ParseValidationFixture(source)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			digest := sha256.Sum256(source)
			fixtureDigests[st.ID] = hex.EncodeToString(digest[:])
			validationFixtures[st.ID] = fixture
			if *stepID != "" && *stepID != st.ID {
				continue
			}
			// Validate selection without re-declaring it in Scenario.
			if _, err := resolver.ResolveApplicationValidations(applicationID); err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			selected = true
			stepsByID[st.ID] = st
			evidence = append(evidence, scenarioStepEvidence{StepID: st.ID, Operation: st.Operation, Verified: false, EvidenceNature: "canonical-validations-bound; not-evaluated"})
			continue
		}
		patternID, err := core.ParseCanonicalID(st.Pattern)
		if err != nil {
			return fmt.Errorf("step %s: %w", st.ID, err)
		}
		resolved, err := resolver.ResolvePattern(applicationID, patternID)
		if err != nil {
			return fmt.Errorf("step %s: %w", st.ID, err)
		}
		if string(resolved.Pattern.Kind) != st.ExpectKind {
			return fmt.Errorf("step %s: expected Pattern kind %s, resolved %s", st.ID, st.ExpectKind, resolved.Pattern.Kind)
		}
		if resolved.Data == nil || string(resolved.Data.ID) != st.ExpectData {
			return fmt.Errorf("step %s: expected Data %s not resolved", st.ID, st.ExpectData)
		}
		if st.ExpectQuery != "" && (resolved.Query == nil || string(resolved.Query.ID) != st.ExpectQuery) {
			return fmt.Errorf("step %s: expected Query %s not resolved", st.ID, st.ExpectQuery)
		}
		var fixture scenariotest.BrowseFixture
		if st.Operation == "execute-browse-query" {
			fixturePath, err := scenarioFixturePath(*scenarioPath, st.Fixture)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			fixtureSource, err := os.ReadFile(fixturePath)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			fixture, err = scenariotest.ParseBrowseFixture(fixtureSource)
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			digest := sha256.Sum256(fixtureSource)
			fixtureDigests[st.ID] = hex.EncodeToString(digest[:])
			resolvedFixtures[st.ID] = fixture
			if resolved.Query == nil || fixture.Query != string(resolved.Query.ID) || resolved.Datasource == nil ||
				fixture.DatasourceAlias != string(resolved.Datasource.Alias) ||
				fixture.TargetDatabase != string(resolved.Datasource.TargetDatabase) {
				return fmt.Errorf("step %s: Browse Fixture query/datasource differs from Canonical", st.ID)
			}
		}
		if *stepID != "" && st.ID != *stepID {
			continue
		}
		selected = true
		stepsByID[st.ID] = st
		rowCount := 0
		nature := "canonical-pattern-resolution-only"
		verified := true
		if st.Operation == "execute-browse-query" {
			nature = "synthetic-fixture-bound; query-not-executed"
			verified = false
		}
		evidence = append(evidence, scenarioStepEvidence{StepID: st.ID, Operation: st.Operation, Pattern: st.Pattern, Verified: verified, EvidenceNature: nature, RowCount: rowCount})
	}
	if action == "run" && !selected {
		return fmt.Errorf("unknown or unselected scenario step %q", *stepID)
	}
	plan, err := scenariotest.Generate(s, source, *scenarioPath, *workspacePath)
	if err != nil {
		return err
	}
	if len(plan.Commands) != len(s.Steps) {
		return fmt.Errorf("generator omitted a Scenario step")
	}
	if len(fixtureDigests) != 0 {
		plan.FixtureSHA256 = fixtureDigests
	}
	worldBytes, err := json.Marshal(loaded.World)
	if err != nil {
		return fmt.Errorf("fingerprint resolved Canonical World: %w", err)
	}
	worldDigest := sha256.Sum256(worldBytes)
	plan.WorldSHA256 = hex.EncodeToString(worldDigest[:])
	lockSource, err := os.ReadFile(filepath.Join(*workspacePath, ".tei", "plugin-lock.json"))
	if err != nil {
		return fmt.Errorf("pin Workspace plugin lock: %w", err)
	}
	lockDigest := sha256.Sum256(lockSource)
	plan.PluginLockSHA256 = hex.EncodeToString(lockDigest[:])
	for i := range plan.Commands {
		command := &plan.Commands[i]
		command.Argv = append(command.Argv,
			"-world-sha256", plan.WorldSHA256,
			"-plugin-lock-sha256", plan.PluginLockSHA256)
		if expected, ok := plan.FixtureSHA256[command.StepID]; ok {
			command.Argv = append(command.Argv, "-fixture-sha256", expected)
		}
	}
	if action == "run" {
		_, stepHasFixture := plan.FixtureSHA256[*stepID]
		if pinsSupplied && (stepHasFixture != (*expectedFixtureSHA != "")) {
			return fmt.Errorf("generated Scenario run requires the exact Fixture pin for the selected Step")
		}
		if *expectedScenarioSHA != "" && *expectedScenarioSHA != plan.ScenarioSHA256 {
			return fmt.Errorf("generated CLI command is stale: Scenario fingerprint differs")
		}
		if *expectedWorldSHA != "" && *expectedWorldSHA != plan.WorldSHA256 {
			return fmt.Errorf("generated CLI command is stale: Canonical World fingerprint differs")
		}
		if *expectedLockSHA != "" && *expectedLockSHA != plan.PluginLockSHA256 {
			return fmt.Errorf("generated CLI command is stale: Plugin Lock fingerprint differs")
		}
		if *expectedFixtureSHA != "" {
			if *stepID == "" || *expectedFixtureSHA != plan.FixtureSHA256[*stepID] {
				return fmt.Errorf("generated CLI command is stale: Fixture fingerprint differs")
			}
		}
		// No Driver Query invocation occurs until all source/binding fingerprints pass.
		for i := range evidence {
			if evidence[i].Operation == "observe-synthetic-state" {
				st := stepsByID[evidence[i].StepID]
				result, err := executeScenarioObservation(st, observationFixtures[st.ID])
				if err != nil {
					return fmt.Errorf("step %s: %w", st.ID, err)
				}
				evidence[i].ObservationResult = result
				evidence[i].Verified = true
				evidence[i].EvidenceNature = "synthetic-independent-observation-read"
				continue
			}
			if evidence[i].Operation == "evaluate-validations" {
				st := stepsByID[evidence[i].StepID]
				results, err := executeScenarioValidations(resolver, applicationID, st, validationFixtures[st.ID])
				if err != nil {
					return fmt.Errorf("step %s: %w", st.ID, err)
				}
				evidence[i].ValidationResults = results
				evidence[i].Verified = true
				evidence[i].EvidenceNature = "canonical-validations-synthetic-input-evaluated"
				continue
			}
			if evidence[i].Operation != "execute-browse-query" {
				continue
			}
			st := stepsByID[evidence[i].StepID]
			rows, err := executeScenarioBrowse(resolver, applicationID, st, resolvedFixtures[st.ID])
			if err != nil {
				return fmt.Errorf("step %s: %w", st.ID, err)
			}
			evidence[i].RowCount = rows
			evidence[i].Verified = true
			evidence[i].EvidenceNature = "synthetic-browse-query-executed"
		}
	}
	switch action {
	case "generate":
		return json.NewEncoder(output).Encode(plan)
	case "check":
		saved, err := os.ReadFile(*planPath)
		if err != nil {
			return err
		}
		var old scenariotest.Plan
		d := json.NewDecoder(bytes.NewReader(saved))
		d.DisallowUnknownFields()
		if err := d.Decode(&old); err != nil {
			return err
		}
		if d.Decode(new(any)) != io.EOF {
			return fmt.Errorf("invalid trailing plan content")
		}
		if !reflect.DeepEqual(old, plan) {
			return fmt.Errorf("stale or modified CLI plan: regenerate from Scenario")
		}
		return json.NewEncoder(output).Encode(map[string]any{"scenarioId": s.ID, "planMatches": true})
	case "verify", "run":
		mode := "static-verification"
		if action == "run" {
			mode = "exploratory-unpinned"
			if pinsSupplied {
				mode = "generated-pinned"
			}
		}
		return json.NewEncoder(output).Encode(map[string]any{
			"scenarioId": s.ID, "scenarioSha256": plan.ScenarioSHA256, "evidence": evidence,
			"executionMode": mode, "selectedStepCount": len(evidence),
			"scope": "reference Pattern, synthetic Browse, Canonical Validation and independent synthetic Observation only; no Auth/Command/Web E2E",
		})
	}
	return fmt.Errorf("unreachable scenario action %q", action)
}
