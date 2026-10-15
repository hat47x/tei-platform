package driverscenarios

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/scenariotest"
)

func revisedBrowseInputs(t *testing.T) ([]byte, []byte, scenariotest.Plan) {
	t.Helper()
	scenarioSource, err := os.ReadFile("order-entry-pattern-smoke-revised.json")
	if err != nil {
		t.Fatal(err)
	}
	fixtureSource, err := os.ReadFile("fixtures/recent_orders_revised.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := scenariotest.Parse(scenarioSource)
	if err != nil {
		t.Fatal(err)
	}
	cli, err := scenariotest.Generate(
		parsed, scenarioSource, "order-entry-pattern-smoke-revised.json", "../workspace",
	)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(fixtureSource)
	cli.FixtureSHA256 = map[string]string{"browse-synthetic": hex.EncodeToString(digest[:])}
	return scenarioSource, fixtureSource, cli
}

func persistedWebCase(t *testing.T, current WebBrowseCase) WebBrowseCase {
	t.Helper()
	raw, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	var saved WebBrowseCase
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	return saved
}

func TestScenarioRevisionRejectsOldCLIAndWebDerivedCase(t *testing.T) {
	oldSource, oldFixture, oldCLI, resolver, binding := webCaseInputs(t)
	oldWeb, err := GenerateWebBrowseCase(oldSource, oldFixture, oldCLI, resolver, binding)
	if err != nil {
		t.Fatal(err)
	}
	oldWeb = persistedWebCase(t, oldWeb)
	if err := CheckCLIWebBrowseDerivations(
		oldSource, oldFixture,
		"order-entry-pattern-smoke.json", "../workspace",
		oldCLI, oldWeb, resolver, binding,
	); err != nil {
		t.Fatalf("unchanged saved CLI/Web outputs do not match Scenario: %v", err)
	}

	newSource, newFixture, newCLI := revisedBrowseInputs(t)
	if oldCLI.ScenarioSHA256 == newCLI.ScenarioSHA256 ||
		oldCLI.FixtureSHA256["browse-synthetic"] == newCLI.FixtureSHA256["browse-synthetic"] {
		t.Fatal("test revision changed neither Scenario nor Fixture fingerprint")
	}
	if err := CheckCLIWebBrowseDerivations(
		newSource, newFixture,
		"order-entry-pattern-smoke-revised.json", "../workspace",
		oldCLI, oldWeb, resolver, binding,
	); err == nil || !strings.Contains(err.Error(), "stale or incomplete derived CLI plan") {
		t.Fatalf("old CLI/Web pair incorrectly qualified against revised Scenario: %v", err)
	}

	newWeb, err := GenerateWebBrowseCase(newSource, newFixture, newCLI, resolver, binding)
	if err != nil {
		t.Fatal(err)
	}
	newWeb = persistedWebCase(t, newWeb)
	if newWeb.ScenarioSHA256 == oldWeb.ScenarioSHA256 ||
		reflect.DeepEqual(newWeb.ExpectedRows, oldWeb.ExpectedRows) {
		t.Fatal("Web projection did not regenerate changed assertions")
	}
	if err := CheckCLIWebBrowseDerivations(
		newSource, newFixture,
		"order-entry-pattern-smoke-revised.json", "../workspace",
		newCLI, oldWeb, resolver, binding,
	); err == nil || !strings.Contains(err.Error(), "stale or modified derived Web Browse case") {
		t.Fatalf("new CLI with old Web case silently passed: %v", err)
	}
	if err := CheckCLIWebBrowseDerivations(
		newSource, newFixture,
		"order-entry-pattern-smoke-revised.json", "../workspace",
		newCLI, newWeb, resolver, binding,
	); err != nil {
		t.Fatalf("regenerated CLI/Web pair failed: %v", err)
	}

	// Execute the revised read through the existing Canonical Driver mapping
	// rather than treating a rewritten expected JSON object as an observed row.
	rawFixture, err := scenariotest.ParseBrowseFixture(newFixture)
	if err != nil {
		t.Fatal(err)
	}
	params := make(map[string]any, len(rawFixture.Parameters))
	for name, value := range rawFixture.Parameters {
		decoded, decodeErr := value.Decode()
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		params[name] = decoded
	}
	execution, err := resolver.ExecuteBrowseQuery(
		context.Background(),
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.recent_orders"),
		webFixtureExecutor{fixture: rawFixture},
		params,
	)
	if err != nil {
		t.Fatal(err)
	}
	foundStatus := false
	for i, column := range execution.Result.Mapped.Columns {
		if string(column.Attribute) == "sales.order.status" {
			foundStatus = true
			if len(execution.Result.Mapped.Rows) != 1 ||
				execution.Result.Mapped.Rows[0][i] != "CLOSED" ||
				newWeb.ExpectedRows[0][string(column.Attribute)] != (scenariotest.Value{Kind: "string", Data: "CLOSED"}) {
				t.Fatalf("revised Driver result and Web Scenario assertion diverged: %#v", execution.Result.Mapped)
			}
		}
	}
	if !foundStatus {
		t.Fatal("Canonical Query mapping omitted revised status attribute")
	}
}

func TestScenarioDerivedPairRejectsTamperOrMissingFixture(t *testing.T) {
	source, fixture, cli, resolver, binding := webCaseInputs(t)
	web, err := GenerateWebBrowseCase(source, fixture, cli, resolver, binding)
	if err != nil {
		t.Fatal(err)
	}
	runCheck := func(plan scenariotest.Plan, derived WebBrowseCase, webBinding WebBrowseBinding) error {
		return CheckCLIWebBrowseDerivations(
			source, fixture, "order-entry-pattern-smoke.json", "../workspace",
			plan, derived, resolver, webBinding,
		)
	}

	// An otherwise identical CLI plan with one changed argv cannot establish
	// lineage from the persisted Scenario's generated command plan.
	modifiedCLI := cli
	modifiedCLI.Commands = append([]scenariotest.Command(nil), cli.Commands...)
	modifiedCLI.Commands[2].Argv = append([]string(nil), cli.Commands[2].Argv...)
	modifiedCLI.Commands[2].Argv[8] = "record-pattern"
	if err := runCheck(modifiedCLI, web, binding); err == nil ||
		!strings.Contains(err.Error(), "stale or modified CLI argv") {
		t.Fatalf("edited derived CLI argv accepted: %v", err)
	}

	// Hand-modifying the Web assertion or coverage list must never become
	// the next Scenario authority.
	modifiedWeb := persistedWebCase(t, web)
	modifiedWeb.ExpectedRows[0]["sales.order.status"] = scenariotest.Value{Kind: "string", Data: "FORGED"}
	if err := runCheck(cli, modifiedWeb, binding); err == nil ||
		!strings.Contains(err.Error(), "stale or modified derived Web Browse case") {
		t.Fatalf("edited Web semantic expectation accepted: %v", err)
	}
	modifiedCoverage := persistedWebCase(t, web)
	modifiedCoverage.UncoveredSteps = nil
	if err := runCheck(cli, modifiedCoverage, binding); err == nil ||
		!strings.Contains(err.Error(), "stale or modified derived Web Browse case") {
		t.Fatalf("deleted uncovered Web Steps accepted: %v", err)
	}

	// Target Web binding changes also require Web output regeneration even
	// though the Scenario and its CLI argv remain identical.
	changedBinding := binding
	changedBinding.Route = "/test/order-entry/revised-orders"
	if err := runCheck(cli, web, changedBinding); err == nil ||
		!strings.Contains(err.Error(), "stale or modified derived Web Browse case") {
		t.Fatalf("old Web route accepted after target mapping update: %v", err)
	}
	changedCLI := cli
	changedCLI.FixtureSHA256 = map[string]string{"browse-synthetic": strings.Repeat("0", 64)}
	if err := runCheck(changedCLI, web, binding); err == nil ||
		!strings.Contains(err.Error(), "stale or missing Browse fixture pin") {
		t.Fatalf("tampered synthetic fixture digest accepted: %v", err)
	}
}
