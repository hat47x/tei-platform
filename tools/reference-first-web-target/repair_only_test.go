//go:build tei_repair_only && !tei_order_only

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/workspace"
)

func repairOnlyReferenceWorld(t *testing.T) ir.World {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot find reference source path")
	}
	loaded, err := workspace.Load(filepath.Join(filepath.Dir(source), "..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	return loaded.World
}

func TestRepairOnlyBinaryUsesRepairRunnerAndNoOrderRoutes(t *testing.T) {
	world := repairOnlyReferenceWorld(t)
	runner := newReferenceCustomerLookupScenarioRunner(world)
	bindings, ok := runner.contracts.(driver.ContractQueryExecutorMap)
	if !ok || len(bindings) != 1 {
		t.Fatalf("repair-only Runner does not have a one-Provider registry: %T / %+v", runner.contracts, runner.contracts)
	}
	result, err := runner.RunCustomerLookup(context.Background(), customerLookupInput{CustomerID: "11"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ApplicationID != "repair_intake" || len(result.CustomerRows) != 1 {
		t.Fatalf("repair-only business meaning changed: %+v", result)
	}
	projection, err := buildWebTargetProjection(world, projectionConfig{
		HandoverBaselineID: "B",
		BaselineRepositoryRevision: "e5231839a5dcc62cce1181a2731fcb7c70d16736",
		TargetBuildRevision: "target-test-revision",
		ApplicationID: core.MustCanonicalID("repair_intake"),
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := newWebTargetServer(projection, nil, nil, withCustomerLookup(runner))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/manifest.json", http.StatusOK},
		{http.MethodGet, "/scenario/customer-lookup?customerId=11", http.StatusOK},
		{http.MethodGet, "/scenario/customer-lookup?customerId=not-a-number", http.StatusBadRequest},
		{http.MethodGet, "/customer-lookup", http.StatusOK},
		{http.MethodPost, "/scenario/customer-lookup", http.StatusMethodNotAllowed},
		{http.MethodGet, "/scenario/read", http.StatusNotFound},
		{http.MethodPost, "/scenario/submit", http.StatusNotFound},
		{http.MethodGet, "/custom-submit", http.StatusNotFound},
	} {
		reply := httptest.NewRecorder()
		target.Handler().ServeHTTP(reply, httptest.NewRequest(tc.method, tc.path, nil))
		if reply.Code != tc.status {
			t.Errorf("%s %s: got %d, want %d: %s", tc.method, tc.path, reply.Code, tc.status, reply.Body.String())
		}
	}
}

// A repair-only binary must not advertise inaccessible order-specific links.
// This checks rendered HTML, not just route HTTP status.
func TestRepairOnlyIndexExposesOnlyRepairNavigation(t *testing.T) {
	world := repairOnlyReferenceWorld(t)
	projection, err := buildWebTargetProjection(world, projectionConfig{
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "e5231839a5dcc62cce1181a2731fcb7c70d16736",
		TargetBuildRevision:        "target-test-revision",
		ApplicationID:              core.MustCanonicalID("repair_intake"),
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(projection, nil, nil,
		withCustomerLookup(newReferenceCustomerLookupScenarioRunner(world)))
	if err != nil {
		t.Fatal(err)
	}
	reply := httptest.NewRecorder()
	server.Handler().ServeHTTP(reply, httptest.NewRequest(http.MethodGet, "/", nil))
	if reply.Code != http.StatusOK {
		t.Fatalf("repair-only index returned %d", reply.Code)
	}
	page := reply.Body.String()
	if !strings.Contains(page, `href="/customer-lookup"`) {
		t.Fatal("repair-only index does not link to its customer lookup UI")
	}
	for _, unavailable := range []string{
		`href="/scenario/read"`,
		`href="/custom-submit"`,
		"POST /scenario/submit",
	} {
		if strings.Contains(page, unavailable) {
			t.Fatalf("repair-only index advertises excluded route %q", unavailable)
		}
	}
}

func TestCompiledRepairEntryRejectsRuntimeApplicationOverride(t *testing.T) {
	before := os.Args
	defer func() { os.Args = before }()
	os.Args = []string{"repair-only", "-application", "order_entry"}
	called := false
	err := runSingleApplicationTarget("repair_intake", func(_ ir.World, _ singleApplicationConfig) (
		readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
	) {
		called = true
		return nil, nil, nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "refuses requested Application") {
		t.Fatalf("runtime Application override accepted: %v", err)
	}
	if called {
		t.Fatal("untrusted runtime Application override reached the business Runner factory")
	}
}
