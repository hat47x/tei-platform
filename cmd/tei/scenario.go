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
	"reflect"
	"path/filepath"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/scenariotest"
	"tei/internal/workspace"
)

type scenarioStepEvidence struct {
	StepID         string `json:"stepId"`
	Operation      string `json:"operation"`
	Pattern        string `json:"pattern"`
	Verified       bool   `json:"verified"`
	EvidenceNature string `json:"evidenceNature"`
	RowCount int `json:"rowCount,omitempty"`
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
	for _, st := range s.Steps {
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
		rowCount := 0
		nature := "canonical-pattern-resolution-only"
		verified := true
		if st.Operation == "execute-browse-query" {
			nature = "synthetic-fixture-bound; query-not-executed"
			verified = false
			if action == "run" {
				rowCount, err = executeScenarioBrowse(resolver, applicationID, st, fixture)
				if err != nil {
					return fmt.Errorf("step %s: %w", st.ID, err)
				}
				verified = true
				nature = "synthetic-browse-query-executed"
			}
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
	plan.FixtureSHA256 = fixtureDigests
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
		return json.NewEncoder(output).Encode(map[string]any{"scenarioId": s.ID, "scenarioSha256": plan.ScenarioSHA256, "evidence": evidence, "scope": "reference Pattern / synthetic Browse only; not Auth/Command/Observation/Web E2E"})
	}
	return fmt.Errorf("unreachable scenario action %q", action)
}
