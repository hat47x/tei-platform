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
	"math"
	"reflect"
	"regexp"
	"strconv"
	"time"
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
	Parameters map[string]Value `json:"parameters,omitempty"`
	Fixture string `json:"fixture,omitempty"`
	ExpectRows []map[string]Value `json:"expectRows"`
	ExpectValidations map[string]string `json:"expectValidations,omitempty"`
	ReadIdentity string `json:"readIdentity,omitempty"`
	ExpectObservation *Value `json:"expectObservation,omitempty"`
}

// Value is a typed test fixture value, never an authority or semantic type definition.
type Value struct {
	Kind string `json:"kind"`
	Data string `json:"data"`
}

func (v Value) Decode() (any, error) {
	switch v.Kind {
	case "string":
		return v.Data, nil
	case "int64":
		return strconv.ParseInt(v.Data, 10, 64)
	case "float64":
		f, err := strconv.ParseFloat(v.Data, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, fmt.Errorf("non-finite or invalid float value")
		}
		return f, nil
	case "timestamp":
		return time.Parse(time.RFC3339Nano, v.Data)
	default:
		return nil, fmt.Errorf("unsupported fixture value kind %q", v.Kind)
	}
}

func ValidateValues(values map[string]Value) error {
	for key, v := range values {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("empty fixture value name")
		}
		if _, err := v.Decode(); err != nil {
			return fmt.Errorf("fixture value %s: %w", key, err)
		}
	}
	return nil
}

// BrowseFixture is a test-only substitute for a resolved Query Executor.
// It stores raw rows independently of the Scenario's expected semantic rows.
type BrowseFixture struct {
	Query string `json:"query"`
	DatasourceAlias string `json:"datasourceAlias"`
	TargetDatabase string `json:"targetDatabase"`
	Parameters map[string]Value `json:"parameters"`
	Columns []string `json:"columns"`
	Rows [][]Value `json:"rows"`
}

func ParseBrowseFixture(source []byte) (BrowseFixture, error) {
	var fixture BrowseFixture
	d := json.NewDecoder(bytes.NewReader(source))
	d.DisallowUnknownFields()
	if err := d.Decode(&fixture); err != nil {
		return BrowseFixture{}, fmt.Errorf("decode Browse Fixture: %w", err)
	}
	if d.Decode(new(any)) != io.EOF {
		return BrowseFixture{}, fmt.Errorf("trailing Browse Fixture content")
	}
	if fixture.Query == "" || fixture.DatasourceAlias == "" || fixture.TargetDatabase == "" || len(fixture.Columns) == 0 {
		return BrowseFixture{}, fmt.Errorf("Browse Fixture missing Query, datasource or columns")
	}
	if err := ValidateValues(fixture.Parameters); err != nil {
		return BrowseFixture{}, err
	}
	for i, row := range fixture.Rows {
		if len(row) != len(fixture.Columns) {
			return BrowseFixture{}, fmt.Errorf("Browse Fixture row %d has %d values for %d columns", i, len(row), len(fixture.Columns))
		}
		for _, value := range row {
			if _, err := value.Decode(); err != nil {
				return BrowseFixture{}, fmt.Errorf("Browse Fixture row %d: %w", i, err)
			}
		}
	}
	return fixture, nil
}

type Command struct {
	StepID string   `json:"stepId"`
	Argv   []string `json:"argv"`
}

type Plan struct {
	SchemaVersion  string    `json:"schemaVersion"`
	ScenarioID     string    `json:"scenarioId"`
	ScenarioSHA256 string    `json:"scenarioSha256"`
	WorldSHA256 string `json:"worldSha256,omitempty"`
	PluginLockSHA256 string `json:"pluginLockSha256,omitempty"`
	FixtureSHA256 map[string]string `json:"fixtureSha256,omitempty"`
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
		if step.Operation != "resolve-pattern" && step.Operation != "execute-browse-query" && step.Operation != "evaluate-validations" && step.Operation != "observe-synthetic-state" {
			return Scenario{}, fmt.Errorf("step %q: unsupported operation %q", step.ID, step.Operation)
		}
		if step.Operation == "observe-synthetic-state" {
			if step.Pattern != "" || step.ExpectKind != "" || step.ExpectData != "" || step.ExpectQuery != "" ||
				step.Parameters != nil || step.ExpectRows != nil || len(step.ExpectValidations) != 0 ||
				step.ReadIdentity == "" || step.Fixture == "" || step.ExpectObservation == nil {
				return Scenario{}, fmt.Errorf("step %q: synthetic Observation requires read identity, fixture and expectation", step.ID)
			}
			if _, err := step.ExpectObservation.Decode(); err != nil {
				return Scenario{}, fmt.Errorf("step %q: %w", step.ID, err)
			}
			continue
		}
		if step.Operation == "evaluate-validations" {
			if step.ReadIdentity != "" || step.ExpectObservation != nil {
				return Scenario{}, fmt.Errorf("step %q: unexpected Observation fields", step.ID)
			}
			if step.Pattern != "" || step.ExpectKind != "" || step.ExpectData != "" || step.ExpectQuery != "" || step.Parameters != nil || step.ExpectRows != nil ||
				strings.TrimSpace(step.Fixture) == "" || len(step.ExpectValidations) == 0 {
				return Scenario{}, fmt.Errorf("step %q: Validation requires fixture and expected results, without Pattern fields", step.ID)
			}
			for id, outcome := range step.ExpectValidations {
				if strings.TrimSpace(id) == "" || (outcome != "satisfied" && outcome != "not-satisfied") {
					return Scenario{}, fmt.Errorf("step %q: unsupported Validation expectation %s=%s", step.ID, id, outcome)
				}
			}
			continue
		}
		if len(step.ExpectValidations) != 0 || step.ReadIdentity != "" || step.ExpectObservation != nil {
			return Scenario{}, fmt.Errorf("step %q: Pattern operation cannot own Validation/Observation results", step.ID)
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
		if step.Operation == "resolve-pattern" {
			if step.Fixture != "" || step.Parameters != nil || step.ExpectRows != nil {
				return Scenario{}, fmt.Errorf("step %q: resolve-pattern cannot have Query execution fields", step.ID)
			}
		} else {
			if step.ExpectKind != "browse" || step.Fixture == "" || step.ExpectRows == nil {
				return Scenario{}, fmt.Errorf("step %q: browse execution requires browse Pattern, fixture and expected rows", step.ID)
			}
			if err := ValidateValues(step.Parameters); err != nil {
				return Scenario{}, fmt.Errorf("step %q: %w", step.ID, err)
			}
			for row, columns := range step.ExpectRows {
				if len(columns) == 0 {
					return Scenario{}, fmt.Errorf("step %q expected row %d is empty", step.ID, row)
				}
				if err := ValidateValues(columns); err != nil {
					return Scenario{}, fmt.Errorf("step %q expected row %d: %w", step.ID, row, err)
				}
			}
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
	if !reflect.DeepEqual(parsed, s) {
		return Plan{}, fmt.Errorf("Scenario input differs from source bytes")
	}
	digest := sha256.Sum256(source)
	plan := Plan{SchemaVersion: PlanVersion, ScenarioID: parsed.ID, ScenarioSHA256: hex.EncodeToString(digest[:]), Commands: make([]Command, 0, len(parsed.Steps))}
	for _, step := range parsed.Steps {
		plan.Commands = append(plan.Commands, Command{StepID: step.ID, Argv: []string{"tei", "scenario", "run", "-scenario", scenarioPath, "-workspace", workspacePath, "-step", step.ID, "-scenario-sha256", plan.ScenarioSHA256}})
	}
	return plan, nil
}
