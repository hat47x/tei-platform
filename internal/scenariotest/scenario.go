// Package scenariotest provides a bounded test-case projection of accepted Driver semantics.
// It never defines domain operations or grants runtime authority.
package scenariotest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const SchemaVersion = "tei.driver-scenario/v0"
const PlanVersion = "tei.cli-scenario-plan/v0"

var stepIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

type Scenario struct {
	SchemaVersion string `json:"schemaVersion"`
	ID            string `json:"id"`
	Application   string `json:"application"`
	Steps         []Step `json:"steps"`
}

type Step struct {
	ID          string `json:"id"`
	Operation   string `json:"operation"`
	Pattern     string `json:"pattern"`
	ExpectKind  string `json:"expectKind"`
	ExpectData  string `json:"expectData"`
	ExpectQuery string `json:"expectQuery,omitempty"`
}

type Command struct {
	StepID string   `json:"stepId"`
	Argv   []string `json:"argv"`
}

type Plan struct {
	SchemaVersion  string    `json:"schemaVersion"`
	ScenarioID     string    `json:"scenarioId"`
	ScenarioSHA256 string    `json:"scenarioSha256"`
	Commands       []Command `json:"commands"`
}

func Parse(source []byte) (Scenario, error) {
	var s Scenario
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&s); err != nil {
		return Scenario{}, fmt.Errorf("decode Driver Scenario: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Scenario{}, fmt.Errorf("trailing Driver Scenario content")
	}
	if s.SchemaVersion != SchemaVersion {
		return Scenario{}, fmt.Errorf("unsupported Scenario schemaVersion %q", s.SchemaVersion)
	}
	if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.Application) == "" {
		return Scenario{}, fmt.Errorf("Scenario id and application are required")
	}
	if len(s.Steps) == 0 {
		return Scenario{}, fmt.Errorf("Scenario must have steps")
	}
	seen := make(map[string]bool, len(s.Steps))
	for _, step := range s.Steps {
		if !stepIDPattern.MatchString(step.ID) || seen[step.ID] {
			return Scenario{}, fmt.Errorf("invalid or duplicate step id %q", step.ID)
		}
		seen[step.ID] = true
		if step.Operation != "resolve-pattern" {
			return Scenario{}, fmt.Errorf("step %q: unsupported operation %q", step.ID, step.Operation)
		}
		if strings.TrimSpace(step.Pattern) == "" || strings.TrimSpace(step.ExpectKind) == "" || strings.TrimSpace(step.ExpectData) == "" {
			return Scenario{}, fmt.Errorf("step %q: missing explicit expectation", step.ID)
		}
		if step.ExpectKind != "browse" && step.ExpectKind != "record" {
			return Scenario{}, fmt.Errorf("step %q: unsupported Pattern expectation %q", step.ID, step.ExpectKind)
		}
		if step.ExpectKind == "browse" && step.ExpectQuery == "" {
			return Scenario{}, fmt.Errorf("step %q: browse requires expectQuery", step.ID)
		}
		if step.ExpectKind == "record" && step.ExpectQuery != "" {
			return Scenario{}, fmt.Errorf("step %q: record cannot expect Query", step.ID)
		}
	}
	return s, nil
}

// Generate derives every CLI command from the Scenario. No step is skipped.
// Paths are explicit inputs, not implicit working-directory state.
func Generate(s Scenario, source []byte, scenarioPath, workspacePath string) (Plan, error) {
	if strings.TrimSpace(scenarioPath) == "" || strings.TrimSpace(workspacePath) == "" {
		return Plan{}, fmt.Errorf("scenario and workspace paths are required")
	}
	parsed, err := Parse(source)
	if err != nil {
		return Plan{}, err
	}
	if parsed.ID != s.ID {
		return Plan{}, fmt.Errorf("Scenario identity mismatch")
	}
	digest := sha256.Sum256(source)
	plan := Plan{SchemaVersion: PlanVersion, ScenarioID: parsed.ID, ScenarioSHA256: hex.EncodeToString(digest[:]), Commands: make([]Command, 0, len(parsed.Steps))}
	for _, step := range parsed.Steps {
		plan.Commands = append(plan.Commands, Command{StepID: step.ID, Argv: []string{"tei", "scenario", "run", "-scenario", scenarioPath, "-workspace", workspacePath, "-step", step.ID}})
	}
	return plan, nil
}
