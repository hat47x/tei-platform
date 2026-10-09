package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

func TestReferenceReadScenarioRunsBrowseRecordReferenceAndHeaderDetailThroughDriver(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := newReferenceReadScenarioRunner(loaded.World)

	result, err := runner.Run(context.Background(), readScenarioInput{
		From:     "2026-09-18T08:30:00Z",
		RowIndex: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.ScenarioID != "order-entry.read-only-web-slice" ||
		result.Environment != "development" ||
		result.ApplicationID != "order_entry" {
		t.Fatalf("scenario identity=%+v", result)
	}
	if len(result.BrowseRows) != 1 {
		t.Fatalf("typed from filter did not constrain browse rows: %+v", result.BrowseRows)
	}
	if got := semanticValue(result.SelectedOrder, "sales.order.order_id"); got != int64(1001) {
		t.Fatalf("selected order id=%#v", got)
	}
	if got := semanticValue(result.SelectedOrder, "sales.order.customer_id"); got != int64(11) {
		t.Fatalf("selected customer id=%#v", got)
	}

	if len(result.CustomerRows) != 1 {
		t.Fatalf("customer reference rows=%d", len(result.CustomerRows))
	}
	if got := semanticValue(result.CustomerRows[0], "customer.customer.customer_id"); got != int64(11) {
		t.Fatalf("customer reference did not use selected browse row: %#v", got)
	}

	if len(result.OrderLineRows) != 2 {
		t.Fatalf("order-line rows=%d", len(result.OrderLineRows))
	}
	if got := semanticValue(result.OrderLineRows[0], "sales.order_line.order_line_id"); got != int64(5001) {
		t.Fatalf("first order-line id=%#v", got)
	}

	resultType := reflect.TypeOf(readScenarioResult{})
	if _, found := resultType.FieldByName("SelectedRow"); found {
		t.Fatal("read scenario result reintroduced ambiguous SelectedRow runtime identity")
	}
	selectedRowField, found := resultType.FieldByName("SelectedRowIndex")
	if !found || selectedRowField.Tag.Get("json") != "selectedRowIndex" {
		t.Fatalf("runtime selection field is not explicitly labelled as an index: %+v", selectedRowField)
	}

	rowType := reflect.TypeOf(semanticRowProjection{})
	if rowType.NumField() != 1 || rowType.Field(0).Name != "Attributes" {
		t.Fatalf("runtime row projection acquired identity authority fields: %v", rowType)
	}
	for _, forbidden := range []string{"ID", "Identity", "Key", "BusinessID", "OccurrenceID"} {
		if _, found := rowType.FieldByName(forbidden); found {
			t.Fatalf("runtime row projection exposes forbidden identity field %q", forbidden)
		}
	}
	for _, limit := range []string{
		"runtime-row-index-is-selection-state-not-business-identity",
		"runtime-row-or-attribute-value-is-not-business-identity-proof",
		"explicit-result-mapping-is-semantic-correspondence-not-business-key-proof",
		"scenario-application-environment-and-binding-identities-are-runtime-context-not-business-data-identity",
	} {
		if !containsString(result.AuthorityLimits, limit) {
			t.Fatalf("runtime identity authority limit %q is missing: %+v", limit, result.AuthorityLimits)
		}
	}

	bindings := runtimeBindingsByPattern(result.RuntimeBindings)
	browse := bindings["order_entry.recent_orders"]
	if browse.Datasource != "sales" || browse.Database != "sales_core" || browse.Access != "read-write" {
		t.Fatalf("browse runtime boundary=%+v", browse)
	}
	if browse.Environment != "" || browse.Instance != "" {
		t.Fatalf("browse Environment was inferred beyond Driver API: %+v", browse)
	}

	customer := bindings["order_entry.customer_reference"]
	if customer.Environment != "development" ||
		customer.Instance != "master_data.dev" ||
		customer.Provider != "oracle" ||
		customer.ExecutorCapability != "database.query.oracle" ||
		customer.ContractQuery != "master_data/master_data.customer/find_by_id" {
		t.Fatalf("customer runtime binding=%+v", customer)
	}

	lines := bindings["order_entry.order_lines"]
	if lines.Environment != "development" ||
		lines.Instance != "sales_core.dev" ||
		lines.Provider != "postgres" ||
		lines.ExecutorCapability != "database.query.postgres" ||
		lines.ContractQuery != "sales_core/sales_core.order_line/find_by_order" {
		t.Fatalf("order-lines runtime binding=%+v", lines)
	}
}

func TestReferenceReadScenarioFailsClosedOnInvalidInputOrSelection(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := newReferenceReadScenarioRunner(loaded.World)

	for _, tc := range []struct {
		name  string
		input readScenarioInput
	}{
		{
			name:  "invalid from",
			input: readScenarioInput{From: "not-a-time", RowIndex: 0},
		},
		{
			name:  "negative row",
			input: readScenarioInput{From: "2026-09-01T00:00:00Z", RowIndex: -1},
		},
		{
			name:  "row outside filtered result",
			input: readScenarioInput{From: "2026-09-18T09:00:00Z", RowIndex: 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runner.Run(context.Background(), tc.input); err == nil {
				t.Fatalf("%s was silently accepted", tc.name)
			}
		})
	}
}

func TestReadScenarioHTTPKeepsGateCProvenanceSeparateFromSemanticResult(t *testing.T) {
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

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0",
		nil,
	)
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("read scenario status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var response webReadScenarioResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(recorder.Body.String(), "\"selectedRow\"") ||
		!strings.Contains(recorder.Body.String(), "\"selectedRowIndex\": 0") {
		t.Fatalf("read scenario JSON does not distinguish row selection state from identity: %s", recorder.Body.String())
	}
	if response.HandoverBaselineID != testBaselineID ||
		response.BaselineRepositoryRevision != testBaselineRevision ||
		response.TargetBuildRevision != testTargetRevision ||
		response.TargetAdapter != "reference-first-web-target" ||
		response.Host != "go-net-http" {
		t.Fatalf("Gate C provenance envelope=%+v", response)
	}
	if response.Result.ScenarioID != "order-entry.read-only-web-slice" {
		t.Fatalf("semantic result identity=%+v", response.Result)
	}
	if got := semanticValue(response.Result.SelectedOrder, "sales.order.order_id"); got != float64(1001) {
		t.Fatalf("JSON selected order id=%#v", got)
	}
}

func TestReadScenarioHTTPRejectsUnsupportedMethodAndBadRow(t *testing.T) {
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

	methodRecorder := httptest.NewRecorder()
	handler.ServeHTTP(methodRecorder, httptest.NewRequest(http.MethodPost, "/scenario/read", nil))
	if methodRecorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", methodRecorder.Code)
	}

	rowRecorder := httptest.NewRecorder()
	handler.ServeHTTP(rowRecorder, httptest.NewRequest(http.MethodGet, "/scenario/read?row=bad", nil))
	if rowRecorder.Code != http.StatusBadRequest {
		t.Fatalf("bad row status=%d body=%s", rowRecorder.Code, rowRecorder.Body.String())
	}
}

func semanticValue(row semanticRowProjection, attribute string) any {
	for _, value := range row.Attributes {
		if value.Attribute == attribute {
			return value.Value
		}
	}
	return nil
}

func runtimeBindingsByPattern(bindings []runtimeBindingProjection) map[string]runtimeBindingProjection {
	result := make(map[string]runtimeBindingProjection, len(bindings))
	for _, binding := range bindings {
		result[binding.Pattern] = binding
	}
	return result
}
