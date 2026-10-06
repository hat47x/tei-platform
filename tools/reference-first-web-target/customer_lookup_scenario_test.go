package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/workspace"
)

func loadReferenceWorld(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	return loaded.World
}

func repairIntakeProjection(t *testing.T, world ir.World) webTargetProjection {
	t.Helper()
	projection, err := buildWebTargetProjection(world, projectionConfig{
		HandoverBaselineID:         testBaselineID,
		BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision:        testTargetRevision,
		ApplicationID:              core.MustCanonicalID("repair_intake"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return projection
}

func TestCustomerLookupRunsRepairIntakeReferencePatternThroughDriver(t *testing.T) {
	runner := newReferenceCustomerLookupScenarioRunner(loadReferenceWorld(t))

	result, err := runner.RunCustomerLookup(context.Background(), customerLookupInput{CustomerID: " 11 "})
	if err != nil {
		t.Fatal(err)
	}

	if result.ScenarioID != "repair-intake.customer-lookup-web-slice" ||
		result.Environment != "development" ||
		result.ApplicationID != "repair_intake" ||
		result.CustomerID != "11" {
		t.Fatalf("scenario identity=%+v", result)
	}
	// The key is read from the mapped intake row through the Relation, not from the request.
	if got := semanticValue(result.Intake, "repair_intake.intake.customer_id"); got != int64(11) {
		t.Fatalf("intake customer id=%#v", got)
	}
	if len(result.CustomerRows) != 1 {
		t.Fatalf("customer rows=%d", len(result.CustomerRows))
	}
	if got := semanticValue(result.CustomerRows[0], "customer.customer.customer_id"); got != int64(11) {
		t.Fatalf("customer id=%#v", got)
	}
	if got := semanticValue(result.CustomerRows[0], "customer.customer.name"); got != "Reference Customer 11" {
		t.Fatalf("customer name=%#v", got)
	}

	if len(result.RuntimeBindings) != 1 {
		t.Fatalf("runtime bindings=%+v", result.RuntimeBindings)
	}
	binding := result.RuntimeBindings[0]
	if binding.Pattern != "repair_intake.customer_reference" ||
		binding.Environment != "development" ||
		binding.Instance != "master_data.dev" ||
		binding.ContractQuery != "master_data/master_data.customer/find_by_id" {
		t.Fatalf("runtime binding=%+v", binding)
	}

	// A synthetic Executor must never read as a real Database connection.
	var sawTestDouble, sawProductionGap bool
	for _, limit := range result.AuthorityLimits {
		if limit == "executor-is-a-test-double-not-a-real-database-connection" {
			sawTestDouble = true
		}
	}
	for _, item := range result.Unresolved {
		if item == "production-environment-has-no-contract-query-executor-binding" {
			sawProductionGap = true
		}
	}
	if !sawTestDouble || !sawProductionGap {
		t.Fatalf("limits=%v unresolved=%v", result.AuthorityLimits, result.Unresolved)
	}
}

// The second Application reuses order_entry's customer meaning and Contract Query. The same
// customer id must resolve to the same customer attributes through the same Executor binding.
func TestCustomerLookupReusesTheCustomerMeaningAndContractOfOrderEntry(t *testing.T) {
	world := loadReferenceWorld(t)

	orderEntry, err := newReferenceReadScenarioRunner(world).Run(context.Background(), readScenarioInput{
		From:     "2026-09-18T08:30:00Z",
		RowIndex: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	repairIntake, err := newReferenceCustomerLookupScenarioRunner(world).RunCustomerLookup(
		context.Background(),
		customerLookupInput{CustomerID: "11"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(orderEntry.CustomerRows) != 1 || len(repairIntake.CustomerRows) != 1 {
		t.Fatalf("customer rows order_entry=%d repair_intake=%d", len(orderEntry.CustomerRows), len(repairIntake.CustomerRows))
	}
	left, right := orderEntry.CustomerRows[0], repairIntake.CustomerRows[0]
	if len(left.Attributes) != len(right.Attributes) {
		t.Fatalf("attribute sets differ: %+v vs %+v", left, right)
	}
	for index := range left.Attributes {
		if left.Attributes[index] != right.Attributes[index] {
			t.Fatalf("attribute %d differs: %+v vs %+v", index, left.Attributes[index], right.Attributes[index])
		}
	}

	byPattern := runtimeBindingsByPattern(orderEntry.RuntimeBindings)
	a := byPattern["order_entry.customer_reference"]
	b := runtimeBindingsByPattern(repairIntake.RuntimeBindings)["repair_intake.customer_reference"]
	a.Pattern, b.Pattern = "", ""
	if a != b {
		t.Fatalf("the two Applications did not resolve the same Executor binding:\n order_entry=%+v\n repair_intake=%+v", a, b)
	}
}

func TestCustomerLookupRejectsInputTheTargetCannotUse(t *testing.T) {
	runner := newReferenceCustomerLookupScenarioRunner(loadReferenceWorld(t))
	for _, raw := range []string{"", "   ", "abc", "1.5", "0x10", "11; DROP", "9223372036854775808"} {
		_, err := runner.RunCustomerLookup(context.Background(), customerLookupInput{CustomerID: raw})
		if !errors.Is(err, errCustomerLookupInput) {
			t.Fatalf("customerId %q: err=%v", raw, err)
		}
	}
}

// Production has no Contract Query executor binding in the Reference. The lookup must fail there,
// not fall back to the development Executor.
func TestCustomerLookupDoesNotFallBackWhenTheEnvironmentHasNoExecutorBinding(t *testing.T) {
	runner := newReferenceCustomerLookupScenarioRunner(loadReferenceWorld(t))
	runner.environmentID = core.MustCanonicalID("production")

	result, err := runner.RunCustomerLookup(context.Background(), customerLookupInput{CustomerID: "11"})
	if err == nil {
		t.Fatalf("production lookup succeeded: %+v", result)
	}
	if errors.Is(err, errCustomerLookupInput) {
		t.Fatalf("an environment gap was reported as bad input: %v", err)
	}
	if len(result.CustomerRows) != 0 {
		t.Fatalf("rows returned despite the error: %+v", result)
	}
}

type undeclaredRunner struct{}

func (undeclaredRunner) Run(context.Context, readScenarioInput) (readScenarioResult, error) {
	return readScenarioResult{}, nil
}

func TestServerRefusesARunnerForAnotherApplicationOrOneThatDoesNotDeclareItsApplication(t *testing.T) {
	world := loadReferenceWorld(t)
	projection := repairIntakeProjection(t, world)

	// order_entry's read runner in front of repair_intake's manifest.
	if _, err := newWebTargetServer(projection, newReferenceReadScenarioRunner(world), nil); err == nil ||
		!strings.Contains(err.Error(), "order_entry") || !strings.Contains(err.Error(), "repair_intake") {
		t.Fatalf("mismatched read runner accepted or unclear: %v", err)
	}
	// A runner that cannot say which Application it executes is not trusted.
	if _, err := newWebTargetServer(projection, undeclaredRunner{}, nil); err == nil ||
		!strings.Contains(err.Error(), "does not declare") {
		t.Fatalf("undeclared runner accepted: %v", err)
	}
	// repair_intake's lookup in front of order_entry's manifest.
	orderEntry := mustOrderEntryProjection(t, world)
	if _, err := newWebTargetServer(orderEntry, nil, nil, withCustomerLookup(newReferenceCustomerLookupScenarioRunner(world))); err == nil {
		t.Fatal("repair_intake lookup accepted for the order_entry manifest")
	}
	// The matching pairs are accepted.
	if _, err := newWebTargetServer(projection, nil, nil, withCustomerLookup(newReferenceCustomerLookupScenarioRunner(world))); err != nil {
		t.Fatal(err)
	}
	if _, err := newWebTargetServer(orderEntry, newReferenceReadScenarioRunner(world), nil); err != nil {
		t.Fatal(err)
	}
}

func mustOrderEntryProjection(t *testing.T, world ir.World) webTargetProjection {
	t.Helper()
	projection, err := buildWebTargetProjection(world, projectionConfig{
		HandoverBaselineID:         testBaselineID,
		BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision:        testTargetRevision,
		ApplicationID:              core.MustCanonicalID("order_entry"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return projection
}

func TestCustomerLookupIsServedAsJSONAndAsAScreenFromTheSameRunner(t *testing.T) {
	world := loadReferenceWorld(t)
	server, err := newWebTargetServer(
		repairIntakeProjection(t, world),
		nil,
		nil,
		withCustomerLookup(newReferenceCustomerLookupScenarioRunner(world)),
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()

	jsonRecorder := httptest.NewRecorder()
	handler.ServeHTTP(jsonRecorder, httptest.NewRequest(http.MethodGet, "/scenario/customer-lookup?customerId=12", nil))
	if jsonRecorder.Code != http.StatusOK {
		t.Fatalf("JSON status=%d body=%s", jsonRecorder.Code, jsonRecorder.Body.String())
	}
	var response webCustomerLookupResponse
	if err := json.Unmarshal(jsonRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.ApplicationID != "repair_intake" ||
		response.TargetBuildRevision != testTargetRevision ||
		response.BaselineRepositoryRevision != testBaselineRevision {
		t.Fatalf("response provenance=%+v", response)
	}
	if got := semanticValue(response.Result.CustomerRows[0], "customer.customer.name"); got != "Reference Customer 12" {
		t.Fatalf("customer name=%#v", got)
	}

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/customer-lookup?customerId=12", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("page status=%d body=%s", page.Code, page.Body.String())
	}
	for _, want := range []string{
		"Reference Customer 12",
		"repair_intake.customer_reference",
		"master_data.dev",
		"executor-is-a-test-double-not-a-real-database-connection",
		"production-environment-has-no-contract-query-executor-binding",
	} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("page does not contain %q", want)
		}
	}

	form := httptest.NewRecorder()
	handler.ServeHTTP(form, httptest.NewRequest(http.MethodGet, "/customer-lookup", nil))
	if form.Code != http.StatusOK || strings.Contains(form.Body.String(), "Reference Customer") {
		t.Fatalf("empty form status=%d", form.Code)
	}
}

func TestCustomerLookupHTTPStatusesKeepBadInputApartFromExecutionFailure(t *testing.T) {
	world := loadReferenceWorld(t)
	configured, err := newWebTargetServer(
		repairIntakeProjection(t, world),
		nil,
		nil,
		withCustomerLookup(newReferenceCustomerLookupScenarioRunner(world)),
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := configured.Handler()

	for _, path := range []string{"/scenario/customer-lookup?customerId=abc", "/scenario/customer-lookup", "/customer-lookup?customerId=abc"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d", path, recorder.Code)
		}
	}
	for _, path := range []string{"/scenario/customer-lookup?customerId=1", "/customer-lookup"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("POST %s status=%d", path, recorder.Code)
		}
	}

	// An Application without a lookup runner answers "not configured", not an empty result.
	unconfigured, err := newWebTargetServer(repairIntakeProjection(t, world), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/scenario/customer-lookup?customerId=1", "/customer-lookup", "/scenario/read", "/scenario/submit"} {
		method := http.MethodGet
		if path == "/scenario/submit" {
			method = http.MethodPost
		}
		recorder := httptest.NewRecorder()
		unconfigured.Handler().ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s status=%d", method, path, recorder.Code)
		}
	}
}
