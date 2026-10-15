package driverscenarios

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/scenariotest"
	"tei/internal/workspace"
)

// This executor only supplies synthetic raw rows. Semantic column mapping
// comes from the existing Driver, not from the Web binding or the fixture.
type webFixtureExecutor struct{ fixture scenariotest.BrowseFixture }

func (e webFixtureExecutor) ExecuteBrowseQuery(
	_ context.Context,
	q query.EmbeddedSQL,
	source application.DatabaseRequirement,
	params map[string]any,
) (driver.BrowseQueryResult, error) {
	if string(q.ID) != e.fixture.Query ||
		string(source.Alias) != e.fixture.DatasourceAlias ||
		string(source.TargetDatabase) != e.fixture.TargetDatabase ||
		len(params) != len(e.fixture.Parameters) {
		return driver.BrowseQueryResult{}, fmt.Errorf("synthetic Query identity or parameter mismatch")
	}
	for name, v := range e.fixture.Parameters {
		want, err := v.Decode()
		if err != nil || !reflect.DeepEqual(want, params[name]) {
			return driver.BrowseQueryResult{}, fmt.Errorf("synthetic parameter %q mismatch", name)
		}
	}
	raw := queryresult.RawResult{
		Columns: append([]string(nil), e.fixture.Columns...),
		Rows:    make([][]any, len(e.fixture.Rows)),
	}
	for i, values := range e.fixture.Rows {
		raw.Rows[i] = make([]any, len(values))
		for j, value := range values {
			decoded, err := value.Decode()
			if err != nil {
				return driver.BrowseQueryResult{}, err
			}
			raw.Rows[i][j] = decoded
		}
	}
	return driver.BrowseQueryResult{Raw: raw}, nil
}

func webCaseInputs(t *testing.T) ([]byte, []byte, scenariotest.Plan, driver.Resolver, WebBrowseBinding) {
	t.Helper()
	source, err := os.ReadFile("order-entry-pattern-smoke.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("fixtures/recent_orders.json")
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := scenariotest.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	cli, err := scenariotest.Generate(scenario, source, "order-entry-pattern-smoke.json", "../workspace")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(fixture)
	cli.FixtureSHA256 = map[string]string{"browse-synthetic": hex.EncodeToString(sum[:])}
	loaded, err := workspace.Load("../workspace")
	if err != nil {
		t.Fatal(err)
	}
	bindingSource, err := os.ReadFile("fixtures/web_browse_binding.json")
	if err != nil {
		t.Fatal(err)
	}
	var binding WebBrowseBinding
	dec := json.NewDecoder(bytes.NewReader(bindingSource))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&binding); err != nil {
		t.Fatal(err)
	}
	return source, fixture, cli, driver.NewResolver(loaded.World), binding
}

func TestGenerateWebBrowseCaseFromExistingScenarioAndDriver(t *testing.T) {
	source, fixtureSource, cli, resolver, binding := webCaseInputs(t)
	webCase, err := GenerateWebBrowseCase(source, fixtureSource, cli, resolver, binding)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateWebBrowseCase(source, fixtureSource, cli, resolver, binding)
	if err != nil || !reflect.DeepEqual(webCase, second) {
		t.Fatalf("non-deterministic Web projection: %v", err)
	}
	if webCase.ScenarioSHA256 != cli.ScenarioSHA256 ||
		webCase.QueryID != "order_entry.recent_orders" ||
		webCase.StepID != "browse-synthetic" ||
		webCase.Method != "GET" ||
		webCase.Route != "/test/order-entry/recent-orders" ||
		len(webCase.UncoveredSteps) != 5 {
		t.Fatalf("Web step is not bound to the original CLI/Driver Scenario: %#v", webCase)
	}

	rawFixture, err := scenariotest.ParseBrowseFixture(fixtureSource)
	if err != nil {
		t.Fatal(err)
	}
	appID, err := core.ParseCanonicalID("order_entry")
	if err != nil {
		t.Fatal(err)
	}
	patternID, err := core.ParseCanonicalID(webCase.PatternID)
	if err != nil {
		t.Fatal(err)
	}
	// Actual in-memory HTTP round trip through the EXISTING Driver Resolver
	// and Canonical Query result mapping, not a static expected JSON copy.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != webCase.Method || r.URL.Path != webCase.Route {
			http.NotFound(w, r)
			return
		}
		params := make(map[string]any, len(rawFixture.Parameters))
		for name, value := range rawFixture.Parameters {
			v, err := value.Decode()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			params[name] = v
		}
		execution, err := resolver.ExecuteBrowseQuery(
			r.Context(), appID, patternID, webFixtureExecutor{fixture: rawFixture}, params,
		)
		if err != nil {
			http.Error(w, "Driver synthetic Query failed", http.StatusBadRequest)
			return
		}
		rows := make([]map[string]any, len(execution.Result.Mapped.Rows))
		for i, row := range execution.Result.Mapped.Rows {
			columns := make(map[string]any, len(row))
			for j, column := range execution.Result.Mapped.Columns {
				columns[webCase.ResponseFields[string(column.Attribute)]] = row[j]
			}
			rows[i] = columns
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(rows); err != nil {
			t.Errorf("encode synthetic HTTP result: %v", err)
		}
	}))
	defer server.Close()

	response, err := server.Client().Get(server.URL + webCase.Route)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("synthetic Web route status=%d", response.StatusCode)
	}
	var got []map[string]any
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	expected := make([]map[string]any, len(webCase.ExpectedRows))
	for i, row := range webCase.ExpectedRows {
		expected[i] = make(map[string]any, len(row))
		for attribute, value := range row {
			decoded, err := value.Decode()
			if err != nil {
				t.Fatal(err)
			}
			expected[i][webCase.ResponseFields[attribute]] = decoded
		}
	}
	data, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	var normalized []map[string]any
	expectedDecoder := json.NewDecoder(bytes.NewReader(data))
	expectedDecoder.UseNumber()
	if err := expectedDecoder.Decode(&normalized); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, normalized) {
		t.Fatalf("synthetic HTTP mapped result differs from Scenario SSOT: got=%v expected=%v", got, normalized)
	}
}

func TestGenerateWebBrowseCaseRejectsMissingBindingAndUnsupportedStep(t *testing.T) {
	source, fixture, cli, resolver, binding := webCaseInputs(t)
	negativeSource, err := os.ReadFile("fixtures/web_browse_binding_missing_total.json")
	if err != nil {
		t.Fatal(err)
	}
	var missing WebBrowseBinding
	if err := json.Unmarshal(negativeSource, &missing); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateWebBrowseCase(source, fixture, cli, resolver, missing); err == nil ||
		!strings.Contains(err.Error(), "missing or extra Semantic attribute binding") {
		t.Fatalf("missing Web response field was silently skipped: %v", err)
	}

	// CLI supports this Scenario Validation, but Web binding/behavior is not
	// implemented. It must not become an empty successful Web test.
	binding.StepID = "validate-positive"
	if _, err := GenerateWebBrowseCase(source, fixture, cli, resolver, binding); err == nil ||
		!strings.Contains(err.Error(), "stale or missing CLI Fixture fingerprint") {
		t.Fatalf("unqualified step unexpectedly passed: %v", err)
	}
	cli.FixtureSHA256["validate-positive"] = cli.FixtureSHA256["browse-synthetic"]
	if _, err := GenerateWebBrowseCase(source, fixture, cli, resolver, binding); err == nil ||
		!strings.Contains(err.Error(), "unsupported or unknown Web Scenario step") {
		t.Fatalf("unsupported Web operation produced success: %v", err)
	}

	_, _, pinnedPlan, _, valid := webCaseInputs(t)
	pinnedPlan.ScenarioSHA256 = strings.Repeat("0", 64)
	if _, err := GenerateWebBrowseCase(source, fixture, pinnedPlan, resolver, valid); err == nil ||
		!strings.Contains(err.Error(), "stale or incomplete CLI Scenario plan") {
		t.Fatalf("stale CLI plan accepted: %v", err)
	}
}

func TestGenerateWebBrowseCaseRejectsAmbiguousTargetBindings(t *testing.T) {
	source, fixture, cli, resolver, binding := webCaseInputs(t)
	binding.ResponseFields["sales.order.total"] = binding.ResponseFields["sales.order.status"]
	if _, err := GenerateWebBrowseCase(source, fixture, cli, resolver, binding); err == nil ||
		!strings.Contains(err.Error(), "missing or ambiguous Web binding") {
		t.Fatalf("two semantic attributes bound to one JSON key: %v", err)
	}
}
