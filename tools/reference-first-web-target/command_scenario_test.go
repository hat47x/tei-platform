package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"net/http"
	"net/http/httptest"

	"tei/internal/core"
	"tei/internal/workspace"
)

func TestReferenceCommandScenarioCompletesWithSeparateBusinessResultsAndObservation(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID:    "legacy-order-editor",
		Credential: "editor-secret",
		From:       "2026-09-18T08:30:00Z",
		RowIndex:   0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stage != "completed" {
		t.Fatalf("stage=%s result=%+v", result.Stage, result)
	}
	if result.Authentication == nil ||
		result.Authentication.Capability != "security.authenticate" ||
		result.Authentication.Outcome != "authenticated" ||
		result.Authentication.Payload["subjectId"] != "subject-order-editor" {
		t.Fatalf("authentication=%+v", result.Authentication)
	}
	if result.Read == nil || result.Read.SelectedRow != 0 {
		t.Fatalf("read=%+v", result.Read)
	}
	if len(result.Validations) != 2 {
		t.Fatalf("validations=%+v", result.Validations)
	}
	for _, validation := range result.Validations {
		if validation.Result != "satisfied" {
			t.Fatalf("validation=%+v", validation)
		}
	}
	if result.Authorization == nil ||
		result.Authorization.Capability != "security.authorize" ||
		result.Authorization.Outcome != "allowed" {
		t.Fatalf("authorization=%+v", result.Authorization)
	}
	if result.Command == nil ||
		result.Command.Capability != "sales.order.submit" ||
		result.Command.Mode != "output" ||
		result.Command.Outcome != "" ||
		result.Command.Payload["accepted"] != true ||
		result.Command.Payload["orderId"] != int64(1001) {
		t.Fatalf("command=%+v", result.Command)
	}
	if result.Observation == nil ||
		result.Observation.ReadIdentity != "synthetic.order_state.read" ||
		result.Observation.Nature != "synthetic" ||
		result.Observation.Result != "submitted" {
		t.Fatalf("observation=%+v", result.Observation)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "editor-secret") {
		t.Fatal("credential leaked into command scenario result")
	}
}

func TestReferenceCommandScenarioKeepsAuthenticationDenialAsBusinessOutcome(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID:    "legacy-order-editor",
		Credential: "wrong-secret",
		From:       "2026-09-18T08:30:00Z",
		RowIndex:   0,
	})
	if err != nil {
		t.Fatalf("business denial became Execution Error: %v", err)
	}
	if result.Stage != "authentication-denied" ||
		result.Authentication == nil ||
		result.Authentication.Outcome != "denied" ||
		result.Authentication.Payload["reason"] != "credential_mismatch" {
		t.Fatalf("authentication denial=%+v", result)
	}
	if result.Read != nil || result.Authorization != nil || result.Command != nil || result.Observation != nil {
		t.Fatalf("authentication denial continued downstream: %+v", result)
	}
}

func TestReferenceCommandScenarioKeepsAuthorizationDenialAsBusinessOutcome(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID:    "legacy-order-viewer",
		Credential: "viewer-secret",
		From:       "2026-09-18T08:30:00Z",
		RowIndex:   0,
	})
	if err != nil {
		t.Fatalf("authorization denial became Execution Error: %v", err)
	}
	if result.Stage != "authorization-denied" ||
		result.Authentication == nil ||
		result.Authentication.Outcome != "authenticated" ||
		result.Authorization == nil ||
		result.Authorization.Outcome != "denied" ||
		result.Authorization.Payload["reason"] != "not_authorized" {
		t.Fatalf("authorization denial=%+v", result)
	}
	if result.Read == nil || len(result.Validations) != 2 {
		t.Fatalf("pre-authorization Driver/Validation path was not retained: %+v", result)
	}
	if result.Command != nil || result.Observation != nil {
		t.Fatalf("denied authorization continued to Command/Observation: %+v", result)
	}
}

func TestReferenceCommandScenarioStopsOnCanonicalValidationNotSatisfied(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	readResult, err := newReferenceReadScenarioRunner(loaded.World).Run(
		context.Background(),
		readScenarioInput{From: "2026-09-18T08:30:00Z", RowIndex: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !setSemanticRowValue(
		&readResult.OrderLineRows[0],
		"sales.order_line.quantity",
		"0",
	) {
		t.Fatal("test setup: quantity attribute not found")
	}
	runner.read = staticReadScenarioRunner{result: readResult}

	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID:    "legacy-order-editor",
		Credential: "editor-secret",
		From:       "2026-09-18T08:30:00Z",
		RowIndex:   0,
	})
	if err != nil {
		t.Fatalf("Validation not-satisfied became Execution Error: %v", err)
	}
	if result.Stage != "validation-not-satisfied" {
		t.Fatalf("stage=%s result=%+v", result.Stage, result)
	}
	if len(result.Validations) != 1 ||
		result.Validations[0].Validation != "sales.order_line.quantity_positive" ||
		result.Validations[0].Result != "not-satisfied" {
		t.Fatalf("validations=%+v", result.Validations)
	}
	if result.Authorization != nil || result.Command != nil || result.Observation != nil {
		t.Fatalf("Validation failure continued downstream: %+v", result)
	}
}

func TestReferenceCommandScenarioUsesRuntimeFixtureForValidationNotSatisfied(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID:    "legacy-order-editor",
		Credential: "editor-secret",
		From:       "2026-09-18T08:00:00Z",
		RowIndex:   1,
	})
	if err != nil {
		t.Fatalf("Validation not-satisfied became Execution Error: %v", err)
	}
	if result.Read == nil {
		t.Fatal("runtime read result is missing")
	}
	if got, ok := semanticRowValueByAttribute(result.Read.SelectedOrder, "sales.order.order_id"); !ok || got != int64(1002) {
		t.Fatalf("selected order=%#v ok=%v", got, ok)
	}
	if result.Stage != "validation-not-satisfied" ||
		len(result.Validations) != 1 ||
		result.Validations[0].Validation != "sales.order_line.quantity_positive" ||
		result.Validations[0].Result != "not-satisfied" {
		t.Fatalf("validation result=%+v", result)
	}
	if result.Authorization != nil || result.Command != nil || result.Observation != nil {
		t.Fatalf("Validation not-satisfied continued downstream: %+v", result)
	}
}

func TestReferenceCommandScenarioPreservesCommandInvocationExecutionError(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID:    "legacy-order-editor",
		Credential: "editor-secret",
		From:       "2026-09-18T07:00:00Z",
		RowIndex:   2,
	})
	if err == nil {
		t.Fatal("synthetic Command adapter failure was converted into a normal Business Result")
	}
	if !strings.Contains(err.Error(), "synthetic submit adapter unavailable for order 1003") {
		t.Fatalf("Execution Error=%v", err)
	}
	if result.Read == nil {
		t.Fatal("pre-Command read result is missing")
	}
	if got, ok := semanticRowValueByAttribute(result.Read.SelectedOrder, "sales.order.order_id"); !ok || got != int64(1003) {
		t.Fatalf("selected order=%#v ok=%v", got, ok)
	}
	if result.Stage != "command" ||
		result.Authentication == nil ||
		result.Authentication.Outcome != "authenticated" ||
		len(result.Validations) != 2 ||
		result.Authorization == nil ||
		result.Authorization.Outcome != "allowed" {
		t.Fatalf("partial Scenario result=%+v", result)
	}
	if result.Command != nil || result.Observation != nil {
		t.Fatalf("Execution Error produced Command/Observation result: %+v", result)
	}
}

func TestSubmitScenarioHTTPPreservesExecutionErrorProvenanceAndPartialStage(t *testing.T) {
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
	commandRunner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(
		projection,
		newReferenceReadScenarioRunner(loaded.World),
		commandRunner,
	)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(
		recorder,
		httptest.NewRequest(
			http.MethodPost,
			"/scenario/submit",
			strings.NewReader(`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T07:00:00Z","row":2}`),
		),
	)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Execution Error status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "editor-secret") {
		t.Fatal("credential leaked into Execution Error response")
	}

	var response webCommandScenarioErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.HandoverBaselineID != testBaselineID ||
		response.BaselineRepositoryRevision != testBaselineRevision ||
		response.TargetBuildRevision != testTargetRevision ||
		response.TargetAdapter != "reference-first-web-target" ||
		response.Host != "go-net-http" {
		t.Fatalf("Execution Error provenance=%+v", response)
	}
	if response.Result.Stage != "command" ||
		response.Result.Command != nil ||
		response.Result.Observation != nil ||
		!strings.Contains(response.ExecutionError, "synthetic submit adapter unavailable for order 1003") {
		t.Fatalf("Execution Error envelope=%+v", response)
	}
}

func TestSubmitScenarioHTTPKeepsGateCProvenanceAndDoesNotEchoCredential(t *testing.T) {
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
	commandRunner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(
		projection,
		newReferenceReadScenarioRunner(loaded.World),
		commandRunner,
	)
	if err != nil {
		t.Fatal(err)
	}

	requestBody := `{
		"loginId":"legacy-order-editor",
		"credential":"editor-secret",
		"from":"2026-09-18T08:30:00Z",
		"row":0
	}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/scenario/submit", strings.NewReader(requestBody))
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("submit status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "editor-secret") {
		t.Fatal("credential leaked into HTTP response")
	}

	var response webCommandScenarioResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.HandoverBaselineID != testBaselineID ||
		response.BaselineRepositoryRevision != testBaselineRevision ||
		response.TargetBuildRevision != testTargetRevision ||
		response.Result.Stage != "completed" {
		t.Fatalf("submit response=%+v", response)
	}
}

func TestSubmitScenarioHTTPSeparatesBusinessDenialFromBadRequestAndExecutionError(t *testing.T) {
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
	commandRunner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(
		projection,
		newReferenceReadScenarioRunner(loaded.World),
		commandRunner,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()

	denied := httptest.NewRecorder()
	handler.ServeHTTP(
		denied,
		httptest.NewRequest(
			http.MethodPost,
			"/scenario/submit",
			strings.NewReader(`{"loginId":"legacy-order-editor","credential":"wrong-secret","from":"2026-09-18T08:30:00Z","row":0}`),
		),
	)
	if denied.Code != http.StatusOK {
		t.Fatalf("Business denial HTTP status=%d body=%s", denied.Code, denied.Body.String())
	}
	var deniedResponse webCommandScenarioResponse
	if err := json.Unmarshal(denied.Body.Bytes(), &deniedResponse); err != nil {
		t.Fatal(err)
	}
	if deniedResponse.Result.Stage != "authentication-denied" {
		t.Fatalf("Business denial result=%+v", deniedResponse.Result)
	}

	badRequest := httptest.NewRecorder()
	handler.ServeHTTP(
		badRequest,
		httptest.NewRequest(
			http.MethodPost,
			"/scenario/submit",
			strings.NewReader(`{"loginId":"legacy-order-editor","credential":"editor-secret","unknown":true}`),
		),
	)
	if badRequest.Code != http.StatusBadRequest {
		t.Fatalf("unknown-field status=%d body=%s", badRequest.Code, badRequest.Body.String())
	}

	executionError := httptest.NewRecorder()
	handler.ServeHTTP(
		executionError,
		httptest.NewRequest(
			http.MethodPost,
			"/scenario/submit",
			strings.NewReader(`{"credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`),
		),
	)
	if executionError.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing login execution status=%d body=%s", executionError.Code, executionError.Body.String())
	}

	wrongMethod := httptest.NewRecorder()
	handler.ServeHTTP(
		wrongMethod,
		httptest.NewRequest(http.MethodGet, "/scenario/submit", nil),
	)
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET submit status=%d", wrongMethod.Code)
	}
}

type staticReadScenarioRunner struct {
	result readScenarioResult
	err    error
}

func (staticReadScenarioRunner) ApplicationID() string { return "order_entry" }

func (s staticReadScenarioRunner) Run(context.Context, readScenarioInput) (readScenarioResult, error) {
	return s.result, s.err
}

func setSemanticRowValue(row *semanticRowProjection, attribute string, value any) bool {
	for index := range row.Attributes {
		if row.Attributes[index].Attribute != attribute {
			continue
		}
		row.Attributes[index].Value = value
		return true
	}
	return false
}

func legacyFixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(referenceWorkspacePath(t), "..", "legacy-auth", name)
}
