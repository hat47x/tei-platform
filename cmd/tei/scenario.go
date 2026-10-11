package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"

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
		if *stepID != "" && st.ID != *stepID {
			continue
		}
		selected = true
		evidence = append(evidence, scenarioStepEvidence{StepID: st.ID, Operation: st.Operation, Pattern: st.Pattern, Verified: true, EvidenceNature: "canonical-pattern-resolution-only"})
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
		return json.NewEncoder(output).Encode(map[string]any{"scenarioId": s.ID, "scenarioSha256": plan.ScenarioSHA256, "evidence": evidence, "scope": "pattern-resolution-only; not Query/Command/Observation E2E"})
	}
	return fmt.Errorf("unreachable scenario action %q", action)
}
