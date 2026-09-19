package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

const (
	testBaselineID       = "B"
	testBaselineRevision = "e5231839a5dcc62cce1181a2731fcb7c70d16736"
	testTargetRevision   = "target-test-revision"
)

func TestBuildWebTargetProjectionUsesDriverResolvedOrderEntryMeaning(t *testing.T) {
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

	if projection.HandoverBaselineID != testBaselineID ||
		projection.BaselineRepositoryRevision != testBaselineRevision ||
		projection.TargetBuildRevision != testTargetRevision {
		t.Fatalf("Gate C provenance changed: %+v", projection)
	}
	if projection.BaselineRepositoryRevision == projection.TargetBuildRevision {
		t.Fatal("baseline revision R and target build revision T were collapsed")
	}
	if projection.ApplicationID != "order_entry" ||
		projection.TargetAdapter != "reference-first-web-target" ||
		projection.Host != "go-net-http" ||
		projection.Stage != "standard-and-custom-ui-shared-boundary" {
		t.Fatalf("target projection identity=%+v", projection)
	}
	if len(projection.Patterns) != 5 {
		t.Fatalf("order_entry Pattern count=%d want=5: %+v", len(projection.Patterns), projection.Patterns)
	}

	byID := patternsByID(projection.Patterns)

	browse := byID["order_entry.recent_orders"]
	if browse.Kind != "browse" || browse.PresentationCandidate != "collection" ||
		browse.Data != "sales.order" || browse.Query != "order_entry.recent_orders" {
		t.Fatalf("browse projection=%+v", browse)
	}
	if browse.Datasource == nil || browse.Datasource.Alias != "sales" ||
		browse.Datasource.Target != "sales_core" || browse.Datasource.Access != "read-write" {
		t.Fatalf("browse datasource projection=%+v", browse.Datasource)
	}

	record := byID["order_entry.order_record"]
	if record.Kind != "record" || record.PresentationCandidate != "item" || record.Data != "sales.order" {
		t.Fatalf("record projection=%+v", record)
	}

	reference := byID["order_entry.customer_reference"]
	if reference.Kind != "reference" || reference.PresentationCandidate != "relation-selection" {
		t.Fatalf("reference projection=%+v", reference)
	}
	if reference.Relation == nil ||
		reference.Relation.ID != "sales.order.customer" ||
		reference.Relation.Source != "sales.order" ||
		reference.Relation.Target != "customer.customer" {
		t.Fatalf("reference Relation projection=%+v", reference.Relation)
	}
	if reference.Datasource == nil ||
		reference.Datasource.Alias != "master" ||
		reference.Datasource.Target != "master_data" ||
		reference.Datasource.Access != "read" {
		t.Fatalf("reference read-only datasource boundary changed: %+v", reference.Datasource)
	}
	if reference.ContractQuery == nil ||
		reference.ContractQuery.Contract != "master_data.customer" ||
		reference.ContractQuery.ContractVersion != "2.4" ||
		reference.ContractQuery.Query != "find_by_id" ||
		reference.ContractQuery.Result != "customer.customer" ||
		reference.ContractQuery.ResultCardinality != "zero-or-one" {
		t.Fatalf("reference Contract Query projection=%+v", reference.ContractQuery)
	}
	if len(reference.ContractQuery.Inputs) != 1 ||
		reference.ContractQuery.Inputs[0].SemanticAttribute != "customer.customer.customer_id" ||
		reference.ContractQuery.Inputs[0].QueryInput != "id" ||
		reference.ContractQuery.Inputs[0].QueryInputType != "customer.customer_id" {
		t.Fatalf("reference Contract Query inputs=%+v", reference.ContractQuery.Inputs)
	}

	headerDetail := byID["order_entry.order_lines"]
	if headerDetail.Kind != "header-detail" || headerDetail.PresentationCandidate != "detail-collection" {
		t.Fatalf("header-detail projection=%+v", headerDetail)
	}
	if headerDetail.Relation == nil || headerDetail.Relation.ID != "sales.order.lines" {
		t.Fatalf("header-detail Relation=%+v", headerDetail.Relation)
	}
	if headerDetail.ContractQuery == nil ||
		headerDetail.ContractQuery.Contract != "sales_core.order_line" ||
		headerDetail.ContractQuery.Query != "find_by_order" ||
		headerDetail.ContractQuery.Result != "sales.order_line" ||
		headerDetail.ContractQuery.ResultCardinality != "many" {
		t.Fatalf("header-detail Contract Query=%+v", headerDetail.ContractQuery)
	}

	command := byID["order_entry.submit_order"]
	if command.Kind != "command" ||
		command.PresentationCandidate != "command-invocation" ||
		command.Capability != "sales.order.submit" {
		t.Fatalf("command projection=%+v", command)
	}

	for _, unresolved := range []string{
		"browse-environment-instance-binding-not-exposed-by-current-driver-api",
		"actual-functional-evidence-not-yet-acquired",
		"visual-evidence-not-yet-acquired",
		"actual-external-command-observation-not-acquired",
	} {
		if !containsString(projection.Unresolved, unresolved) {
			t.Fatalf("projection hid unresolved boundary %q: %+v", unresolved, projection.Unresolved)
		}
	}
}

func TestWebTargetServerRendersHTMLAndJSONFromSameProjection(t *testing.T) {
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
	handler := server.Handler()

	htmlRecorder := httptest.NewRecorder()
	handler.ServeHTTP(htmlRecorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if htmlRecorder.Code != http.StatusOK {
		t.Fatalf("HTML status=%d body=%s", htmlRecorder.Code, htmlRecorder.Body.String())
	}
	htmlBody := htmlRecorder.Body.String()
	for _, want := range []string{
		"order_entry",
		testBaselineRevision,
		testTargetRevision,
		"order_entry.customer_reference",
		"master_data.customer",
		"standard-and-custom-ui-shared-boundary",
	} {
		if !strings.Contains(htmlBody, want) {
			t.Fatalf("HTML projection does not contain %q", want)
		}
	}

	jsonRecorder := httptest.NewRecorder()
	handler.ServeHTTP(jsonRecorder, httptest.NewRequest(http.MethodGet, "/manifest.json", nil))
	if jsonRecorder.Code != http.StatusOK {
		t.Fatalf("manifest status=%d body=%s", jsonRecorder.Code, jsonRecorder.Body.String())
	}
	var manifest webTargetProjection
	if err := json.Unmarshal(jsonRecorder.Body.Bytes(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ApplicationID != projection.ApplicationID ||
		manifest.HandoverBaselineID != projection.HandoverBaselineID ||
		manifest.BaselineRepositoryRevision != projection.BaselineRepositoryRevision ||
		manifest.TargetBuildRevision != projection.TargetBuildRevision ||
		len(manifest.Patterns) != len(projection.Patterns) {
		t.Fatalf("HTML/JSON server does not expose one shared projection: got=%+v want=%+v", manifest, projection)
	}

	healthRecorder := httptest.NewRecorder()
	handler.ServeHTTP(healthRecorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthRecorder.Code != http.StatusOK || healthRecorder.Body.String() != "ok\n" {
		t.Fatalf("health response=%d %q", healthRecorder.Code, healthRecorder.Body.String())
	}
}

func TestBuildWebTargetProjectionRequiresDistinctGateCProvenanceInputs(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	base := projectionConfig{
		HandoverBaselineID:         testBaselineID,
		BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision:        testTargetRevision,
		ApplicationID:              core.MustCanonicalID("order_entry"),
	}
	cases := []struct {
		name string
		edit func(*projectionConfig)
	}{
		{name: "baseline id", edit: func(c *projectionConfig) { c.HandoverBaselineID = "" }},
		{name: "baseline revision", edit: func(c *projectionConfig) { c.BaselineRepositoryRevision = "" }},
		{name: "target revision", edit: func(c *projectionConfig) { c.TargetBuildRevision = "" }},
		{name: "application id", edit: func(c *projectionConfig) { c.ApplicationID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.edit(&cfg)
			if _, err := buildWebTargetProjection(loaded.World, cfg); err == nil {
				t.Fatalf("missing %s was silently accepted", tc.name)
			}
		})
	}
}

func patternsByID(patterns []webPatternProjection) map[string]webPatternProjection {
	result := make(map[string]webPatternProjection, len(patterns))
	for _, pattern := range patterns {
		result[pattern.ID] = pattern
	}
	return result
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func referenceWorkspacePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "reference", "workspace")
}
