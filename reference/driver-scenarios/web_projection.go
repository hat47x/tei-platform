// Package driverscenarios contains a bounded, executable Reference projection
// from a persisted Driver Scenario. It does not define another operation SSOT.
package driverscenarios

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/scenariotest"
)

const WebBrowsePlanVersion = "tei.web-browse-e2e-reference/v0"

// WebBrowseBinding is target-specific routing and response field information,
// not authority to change the underlying Canonical Query or Scenario assertions.
type WebBrowseBinding struct {
	StepID         string            `json:"stepId"`
	Method         string            `json:"method"`
	Route          string            `json:"route"`
	ResponseFields map[string]string `json:"responseFields"`
}

// WebBrowseCase is a derived Web HTTP test case. It explicitly enumerates
// Scenario steps it cannot qualify rather than claiming complete E2E coverage.
type WebBrowseCase struct {
	SchemaVersion  string                         `json:"schemaVersion"`
	ScenarioID     string                         `json:"scenarioId"`
	ScenarioSHA256 string                         `json:"scenarioSha256"`
	FixtureSHA256  string                         `json:"fixtureSha256"`
	StepID         string                         `json:"stepId"`
	Operation      string                         `json:"operation"`
	PatternID      string                         `json:"patternId"`
	QueryID        string                         `json:"queryId"`
	Method         string                         `json:"method"`
	Route          string                         `json:"route"`
	ResponseFields map[string]string              `json:"responseFields"`
	ExpectedRows   []map[string]scenariotest.Value `json:"expectedRows"`
	UncoveredSteps []string                       `json:"uncoveredSteps"`
	EvidenceNature string                         `json:"evidenceNature"`
}

// GenerateWebBrowseCase connects an existing Scenario and its CLI command plan
// to one explicit Web read binding and the existing Driver Resolver. Other
// Scenario operations are not silently skipped: they remain visible as
// uncoveredSteps. The returned case is only a synthetic HTTP Reference;
// real Web session/auth/CSRF and browser E2E remain unverified.
func GenerateWebBrowseCase(
	scenarioSource, fixtureSource []byte,
	cliPlan scenariotest.Plan,
	resolver driver.Resolver,
	binding WebBrowseBinding,
) (WebBrowseCase, error) {
	scenario, err := scenariotest.Parse(scenarioSource)
	if err != nil {
		return WebBrowseCase{}, err
	}
	sum := sha256.Sum256(scenarioSource)
	scenarioDigest := hex.EncodeToString(sum[:])
	if cliPlan.SchemaVersion != scenariotest.PlanVersion ||
		cliPlan.ScenarioID != scenario.ID || cliPlan.ScenarioSHA256 != scenarioDigest ||
		len(cliPlan.Commands) != len(scenario.Steps) {
		return WebBrowseCase{}, fmt.Errorf("stale or incomplete CLI Scenario plan")
	}
	for i, st := range scenario.Steps {
		if cliPlan.Commands[i].StepID != st.ID || len(cliPlan.Commands[i].Argv) == 0 {
			return WebBrowseCase{}, fmt.Errorf("CLI Scenario step coverage mismatch at %s", st.ID)
		}
	}
	fixtureDigest := sha256.Sum256(fixtureSource)
	fixtureSHA := hex.EncodeToString(fixtureDigest[:])
	if binding.StepID == "" || cliPlan.FixtureSHA256[binding.StepID] != fixtureSHA {
		return WebBrowseCase{}, fmt.Errorf("stale or missing CLI Fixture fingerprint for Web step %q", binding.StepID)
	}
	if binding.Method != "GET" || !strings.HasPrefix(binding.Route, "/") ||
		strings.HasPrefix(binding.Route, "//") || strings.ContainsAny(binding.Route, "?#") ||
		strings.Contains(binding.Route, "..") {
		return WebBrowseCase{}, fmt.Errorf("Web Browse binding requires safe explicit GET route")
	}

	var selected *scenariotest.Step
	uncovered := make([]string, 0, len(scenario.Steps)-1)
	for i := range scenario.Steps {
		st := &scenario.Steps[i]
		if st.ID == binding.StepID {
			selected = st
		} else {
			uncovered = append(uncovered, st.ID)
		}
	}
	if selected == nil || selected.Operation != "execute-browse-query" || selected.ExpectKind != "browse" {
		return WebBrowseCase{}, fmt.Errorf("unsupported or unknown Web Scenario step %q", binding.StepID)
	}
	fixture, err := scenariotest.ParseBrowseFixture(fixtureSource)
	if err != nil {
		return WebBrowseCase{}, fmt.Errorf("invalid synthetic Web Fixture: %w", err)
	}
	applicationID, err := core.ParseCanonicalID(scenario.Application)
	if err != nil {
		return WebBrowseCase{}, err
	}
	patternID, err := core.ParseCanonicalID(selected.Pattern)
	if err != nil {
		return WebBrowseCase{}, err
	}
	resolved, err := resolver.ResolvePattern(applicationID, patternID)
	if err != nil {
		return WebBrowseCase{}, err
	}
	if resolved.Pattern.Kind != application.PatternBrowse ||
		resolved.Query == nil || resolved.Data == nil || resolved.Datasource == nil ||
		string(resolved.Query.ID) != selected.ExpectQuery ||
		string(resolved.Data.ID) != selected.ExpectData ||
		fixture.Query != string(resolved.Query.ID) ||
		fixture.DatasourceAlias != string(resolved.Datasource.Alias) ||
		fixture.TargetDatabase != string(resolved.Datasource.TargetDatabase) ||
		!reflect.DeepEqual(fixture.Parameters, selected.Parameters) {
		return WebBrowseCase{}, fmt.Errorf("Web Browse Scenario and Canonical Query/Fixture identity mismatch")
	}
	if len(resolved.Query.ResultColumns) == 0 ||
		len(binding.ResponseFields) != len(resolved.Query.ResultColumns) {
		return WebBrowseCase{}, fmt.Errorf("Web response mapping missing or extra Semantic attribute binding")
	}
	seenJSONFields := make(map[string]bool, len(binding.ResponseFields))
	seenAttrs := make(map[string]bool, len(binding.ResponseFields))
	for _, resultColumn := range resolved.Query.ResultColumns {
		attribute := string(resultColumn.Attribute)
		name, found := binding.ResponseFields[attribute]
		if !found || strings.TrimSpace(name) == "" || seenJSONFields[name] || seenAttrs[attribute] {
			return WebBrowseCase{}, fmt.Errorf("missing or ambiguous Web binding for Semantic attribute %q", attribute)
		}
		seenJSONFields[name] = true
		seenAttrs[attribute] = true
	}
	if selected.ExpectRows == nil {
		return WebBrowseCase{}, fmt.Errorf("Web Browse requires explicit row assertions")
	}
	for i, row := range selected.ExpectRows {
		if len(row) != len(seenAttrs) {
			return WebBrowseCase{}, fmt.Errorf("Web Browse expected row %d has incomplete Semantic assertions", i)
		}
		for attr := range row {
			if !seenAttrs[attr] {
				return WebBrowseCase{}, fmt.Errorf("Web Browse expected row %d references undeclared attribute %q", i, attr)
			}
		}
	}
	return WebBrowseCase{
		SchemaVersion: WebBrowsePlanVersion, ScenarioID: scenario.ID,
		ScenarioSHA256: scenarioDigest, FixtureSHA256: fixtureSHA,
		StepID: selected.ID, Operation: selected.Operation,
		PatternID: selected.Pattern, QueryID: selected.ExpectQuery,
		Method: binding.Method, Route: binding.Route,
		ResponseFields: binding.ResponseFields, ExpectedRows: selected.ExpectRows,
		UncoveredSteps: uncovered,
		EvidenceNature: "synthetic-http-reference; no-browser-auth-or-real-provider",
	}, nil
}
