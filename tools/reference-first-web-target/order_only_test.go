//go:build tei_order_only && !tei_repair_only && !tei_api_only

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/workspace"
)

func TestOrderOnlyBinaryRetainsOrderRoutesWithoutRepairLookup(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{
		HandoverBaselineID:         testBaselineID,
		BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision:        testTargetRevision,
		ApplicationID:              core.MustCanonicalID("order_entry"),
	})
	if err != nil {
		t.Fatal(err)
	}
	command, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	target, err := newWebTargetServer(projection, newReferenceReadScenarioRunner(loaded.World), command)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/manifest.json", http.StatusOK},
		{http.MethodGet, "/scenario/read?from=2026-09-18T08:30:00Z&row=0", http.StatusOK},
		{http.MethodGet, "/custom-submit", http.StatusOK},
		{http.MethodPost, "/scenario/submit", http.StatusBadRequest},
		{http.MethodGet, "/scenario/customer-lookup?customerId=11", http.StatusNotFound},
		{http.MethodGet, "/customer-lookup", http.StatusNotFound},
	} {
		recorder := httptest.NewRecorder()
		target.Handler().ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
		if recorder.Code != tc.status {
			t.Errorf("%s %s: got %d, expected %d: %s",
				tc.method, tc.path, recorder.Code, tc.status, recorder.Body.String())
		}
	}
}

// Index navigation must match the physical order-only HTTP routing surface.
func TestOrderOnlyIndexExposesOrderNavigation(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{
		HandoverBaselineID:         testBaselineID,
		BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision:        testTargetRevision,
		ApplicationID:              core.MustCanonicalID("order_entry"),
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(projection, newReferenceReadScenarioRunner(loaded.World), nil)
	if err != nil {
		t.Fatal(err)
	}
	reply := httptest.NewRecorder()
	server.Handler().ServeHTTP(reply, httptest.NewRequest(http.MethodGet, "/", nil))
	if reply.Code != http.StatusOK {
		t.Fatalf("order-only index returned %d", reply.Code)
	}
	page := reply.Body.String()
	if !strings.Contains(page, `href="/scenario/read"`) ||
		!strings.Contains(page, `href="/custom-submit"`) {
		t.Fatal("order-only index lost its read or custom-submit navigation")
	}
	if strings.Contains(page, `href="/customer-lookup"`) {
		t.Fatal("order-only index advertises excluded repair lookup UI")
	}
}

func TestCompiledOrderEntryRejectsRuntimeApplicationOverride(t *testing.T) {
	before := os.Args
	defer func() { os.Args = before }()
	os.Args = []string{"order-only", "-application", "repair_intake"}
	called := false
	err := runSingleApplicationTarget("order_entry", func(_ ir.World, _ singleApplicationConfig) (
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
