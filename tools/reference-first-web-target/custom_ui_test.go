//go:build !tei_repair_only && !tei_api_only

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

func TestCustomSubmitUIUsesSameCommandScenarioRunnerAsJSONEndpoint(t *testing.T) {
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
	recording := &recordingCommandScenarioRunner{
		result: commandScenarioResult{
			ScenarioID:    "order-entry.submit-order-web-slice",
			ApplicationID: "order_entry",
			Stage:         "completed",
			Command: &businessResultProjection{
				Capability: "sales.order.submit",
				Mode:       "output",
				Payload: map[string]any{
					"accepted": true,
					"orderId":  int64(1001),
				},
			},
			Observation: &observationProjection{
				ReadIdentity: "synthetic.order_state.read",
				Nature:       "synthetic",
				Result:       "submitted",
			},
		},
	}
	server, err := newWebTargetServer(
		projection,
		newReferenceReadScenarioRunner(loaded.World),
		recording,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()

	jsonRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		jsonRecorder,
		httptest.NewRequest(
			http.MethodPost,
			"/scenario/submit",
			strings.NewReader("{\"loginId\":\"legacy-order-editor\",\"credential\":\"editor-secret\",\"from\":\"2026-09-18T08:30:00Z\",\"row\":0}"),
		),
	)
	if jsonRecorder.Code != http.StatusOK {
		t.Fatalf("JSON endpoint status=%d body=%s", jsonRecorder.Code, jsonRecorder.Body.String())
	}

	form := url.Values{}
	form.Set("loginId", "legacy-order-editor")
	form.Set("credential", "editor-secret")
	form.Set("from", "2026-09-18T08:30:00Z")
	form.Set("row", "0")
	customRequest := httptest.NewRequest(
		http.MethodPost,
		"/custom-submit",
		strings.NewReader(form.Encode()),
	)
	customRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	customRecorder := httptest.NewRecorder()
	handler.ServeHTTP(customRecorder, customRequest)
	if customRecorder.Code != http.StatusOK {
		t.Fatalf("Custom UI status=%d body=%s", customRecorder.Code, customRecorder.Body.String())
	}

	if len(recording.calls) != 2 {
		t.Fatalf("shared command runner calls=%d want=2", len(recording.calls))
	}
	if !reflect.DeepEqual(recording.calls[0], recording.calls[1]) {
		t.Fatalf("JSON and Custom UI passed different semantic input: json=%+v custom=%+v", recording.calls[0], recording.calls[1])
	}
	if recording.calls[0].LoginID != "legacy-order-editor" ||
		recording.calls[0].Credential != "editor-secret" ||
		recording.calls[0].From != "2026-09-18T08:30:00Z" ||
		recording.calls[0].RowIndex != 0 {
		t.Fatalf("shared semantic input=%+v", recording.calls[0])
	}

	body := customRecorder.Body.String()
	for _, want := range []string{
		"Custom Submit UI",
		"order-entry.submit-order-web-slice",
		"sales.order.submit",
		"synthetic.order_state.read",
		testBaselineRevision,
		testTargetRevision,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Custom UI result does not contain %q", want)
		}
	}
	if strings.Contains(body, "editor-secret") {
		t.Fatal("Custom UI echoed credential")
	}
}

func TestCustomSubmitUIGetIsPresentationOnlyAndContainsNoPresetCredential(t *testing.T) {
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
	server, err := newWebTargetServer(
		projection,
		newReferenceReadScenarioRunner(loaded.World),
		&recordingCommandScenarioRunner{},
	)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/custom-submit", nil),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET Custom UI status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"<form method=\"post\" action=\"/custom-submit\"",
		"name=\"loginId\"",
		"name=\"credential\" type=\"password\"",
		"commandScenarioRunner",
		`name="row" type="number" min="0" required`,
		testBaselineRevision,
		testTargetRevision,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET Custom UI does not contain %q", want)
		}
	}
	for _, forbidden := range []string{"editor-secret", "viewer-secret", "credential-ref://"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("GET Custom UI contains preset credential material %q", forbidden)
		}
	}
}

func TestCustomSubmitUIKeepsBusinessDenialAsRenderedResult(t *testing.T) {
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
	recording := &recordingCommandScenarioRunner{
		result: commandScenarioResult{
			ScenarioID:    "order-entry.submit-order-web-slice",
			ApplicationID: "order_entry",
			Stage:         "authorization-denied",
			Authorization: &businessResultProjection{
				Capability: "security.authorize",
				Mode:       "outcome",
				Outcome:    "denied",
				Payload:    map[string]any{"reason": "not_authorized"},
			},
		},
	}
	server, err := newWebTargetServer(
		projection,
		newReferenceReadScenarioRunner(loaded.World),
		recording,
	)
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("loginId", "legacy-order-viewer")
	form.Set("credential", "viewer-secret")
	form.Set("from", "2026-09-18T08:30:00Z")
	form.Set("row", "0")
	request := httptest.NewRequest(http.MethodPost, "/custom-submit", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Business denial Custom UI status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "authorization-denied") ||
		!strings.Contains(recorder.Body.String(), "not_authorized") {
		t.Fatalf("Business denial was not rendered as result: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "Execution error") {
		t.Fatal("Business denial was rendered as Execution Error")
	}
}

func TestCustomSubmitUIStructuresSuccessfulScenarioForVisualReview(t *testing.T) {
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

	recorder := postCustomSubmit(t, server.Handler(), url.Values{
		"loginId":    {"legacy-order-editor"},
		"credential": {"editor-secret"},
		"from":       {"2026-09-18T08:30:00Z"},
		"row":        {"0"},
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("successful Custom UI status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"Scenario state",
		"completed",
		"Authentication",
		"Selected order and relations",
		"Customer reference",
		"Order lines",
		"Canonical Validation",
		"sales.order_line.quantity_positive",
		"sales.order.lines.line_number_unique",
		"Authorization",
		"Command Business Result",
		"sales.order.submit",
		"Separate Observation",
		"synthetic.order_state.read",
		"submitted",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("successful Custom UI does not contain %q", want)
		}
	}
	if strings.Contains(body, "editor-secret") {
		t.Fatal("successful Custom UI echoed credential")
	}
}

func TestCustomSubmitUISeparatesValidationAuthorizationAndExecutionErrorStates(t *testing.T) {
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

	validation := postCustomSubmit(t, handler, url.Values{
		"loginId":    {"legacy-order-editor"},
		"credential": {"editor-secret"},
		"from":       {"2026-09-18T08:00:00Z"},
		"row":        {"1"},
	})
	if validation.Code != http.StatusOK {
		t.Fatalf("Validation not-satisfied status=%d body=%s", validation.Code, validation.Body.String())
	}
	validationBody := validation.Body.String()
	for _, want := range []string{
		"validation-not-satisfied",
		"Canonical Validation",
		"sales.order_line.quantity_positive",
		"not-satisfied",
	} {
		if !strings.Contains(validationBody, want) {
			t.Fatalf("Validation state does not contain %q", want)
		}
	}
	for _, forbidden := range []string{"Command Business Result", "Separate Observation", "Execution error"} {
		if strings.Contains(validationBody, forbidden) {
			t.Fatalf("Validation state incorrectly contains %q", forbidden)
		}
	}

	authorization := postCustomSubmit(t, handler, url.Values{
		"loginId":    {"legacy-order-viewer"},
		"credential": {"viewer-secret"},
		"from":       {"2026-09-18T08:30:00Z"},
		"row":        {"0"},
	})
	if authorization.Code != http.StatusOK {
		t.Fatalf("Authorization denied status=%d body=%s", authorization.Code, authorization.Body.String())
	}
	authorizationBody := authorization.Body.String()
	for _, want := range []string{
		"authorization-denied",
		"Authorization",
		"security.authorize",
		"denied",
		"not_authorized",
	} {
		if !strings.Contains(authorizationBody, want) {
			t.Fatalf("Authorization state does not contain %q", want)
		}
	}
	if strings.Contains(authorizationBody, "Execution error") {
		t.Fatal("Authorization denial was rendered as Execution Error")
	}

	execution := postCustomSubmit(t, handler, url.Values{
		"loginId":    {"legacy-order-editor"},
		"credential": {"editor-secret"},
		"from":       {"2026-09-18T07:00:00Z"},
		"row":        {"2"},
	})
	if execution.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Execution Error status=%d body=%s", execution.Code, execution.Body.String())
	}
	executionBody := execution.Body.String()
	for _, want := range []string{
		"Scenario state",
		"command",
		"Authorization",
		"allowed",
		"Execution error",
		"synthetic submit adapter unavailable for order 1003",
	} {
		if !strings.Contains(executionBody, want) {
			t.Fatalf("Execution Error state does not contain %q", want)
		}
	}
	for _, forbidden := range []string{"Command Business Result", "Separate Observation", "editor-secret"} {
		if strings.Contains(executionBody, forbidden) {
			t.Fatalf("Execution Error state incorrectly contains %q", forbidden)
		}
	}
}

func TestCustomSubmitUIRejectsMissingOrInvalidRowBeforeCommand(t *testing.T) {
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
	recording := &recordingCommandScenarioRunner{}
	server, err := newWebTargetServer(
		projection,
		newReferenceReadScenarioRunner(loaded.World),
		recording,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name       string
		includeRow bool
		row        string
		want       string
	}{
		{"missing", false, "", "selected row is required"},
		{"empty", true, "", "selected row is required"},
		{"not-integer", true, "abc", "row must be a non-negative integer"},
		{"negative", true, "-1", "row must be a non-negative integer"},
		{"overflow", true, "9999999999999999999999999999", "row must be a non-negative integer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{
				"loginId":    {"test-user"},
				"credential": {"test-credential"},
				"from":       {"2026-09-18T08:30:00Z"},
			}
			if tc.includeRow {
				form.Set("row", tc.row)
			}
			response := postCustomSubmit(t, server.Handler(), form)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want 400", response.Code)
			}
			body := response.Body.String()
			if !strings.Contains(body, `id="input-error"`) ||
				!strings.Contains(body, tc.want) {
				t.Fatalf("input diagnostic missing: %s", body)
			}
			for _, forbidden := range []string{
				`id="execution-error"`,
				"Command Business Result",
				"test-credential",
			} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("unexpected content %q", forbidden)
				}
			}
			if len(recording.calls) != 0 {
				t.Fatalf("invalid input invoked command runner: %d calls", len(recording.calls))
			}
		})
	}
}

func TestCustomSubmitAndJSONSeparateCommitUncertaintyFromObservationFailure(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{
		HandoverBaselineID: testBaselineID, BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision: testTargetRevision, ApplicationID: core.MustCanonicalID("order_entry"),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name        string
		stage       string
		status      int
		wantCommand bool
		store       func() orderSubmitStore
	}{
		{"commit-uncertain", "command-commit-unconfirmed", http.StatusInternalServerError, false,
			func() orderSubmitStore { return &uncertainCommitOrderStore{} }},
		{"observation-failed", "observation", http.StatusUnprocessableEntity, true,
			func() orderSubmitStore { return &observationReadFailureStore{} }},
	} {
		for _, endpoint := range []string{"json", "custom"} {
			t.Run(test.name+"/"+endpoint, func(t *testing.T) {
				store := test.store()
				server, err := newWebTargetServer(projection, newReferenceReadScenarioRunner(loaded.World), newOrderSubmitRunner(t, store))
				if err != nil {
					t.Fatal(err)
				}
				var response *httptest.ResponseRecorder
				if endpoint == "custom" {
					response = postCustomSubmit(t, server.Handler(), url.Values{
						"loginId": {"legacy-order-editor"}, "credential": {"editor-secret"},
						"from": {"2026-09-18T08:30:00Z"}, "row": {"0"},
					})
					body := response.Body.String()
					if !strings.Contains(body, test.stage) ||
						strings.Contains(body, `id="command-result"`) != test.wantCommand || strings.Contains(body, "Separate Observation") {
						t.Fatal("Custom UI lost the execution stage or misrepresented Command/Observation")
					}
				} else {
					response = httptest.NewRecorder()
					server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/scenario/submit",
						strings.NewReader(`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`)))
					var got webCommandScenarioErrorResponse
					if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					if got.Result.Stage != test.stage || (got.Result.Command != nil) != test.wantCommand || got.Result.Observation != nil {
						t.Fatal("JSON lost the execution stage or misrepresented Command/Observation")
					}
				}
				if response.Code != test.status {
					t.Fatalf("HTTP status=%d want=%d", response.Code, test.status)
				}
				for _, private := range []string{"fixture-secret", "internal-db", "postgres://", "editor-secret"} {
					if strings.Contains(response.Body.String(), private) {
						t.Fatalf("response disclosed private detail %q", private)
					}
				}
				switch store := store.(type) {
				case *uncertainCommitOrderStore:
					if store.submissions != 1 || store.reads != 0 {
						t.Fatal("uncertain Commit retried or read state implicitly")
					}
				case *observationReadFailureStore:
					if store.submissions != 1 {
						t.Fatal("Observation failure retried the Command")
					}
				}
			})
		}
	}
}

func postCustomSubmit(t *testing.T, handler http.Handler, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/custom-submit", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

type recordingCommandScenarioRunner struct {
	calls  []commandScenarioInput
	result commandScenarioResult
	err    error
}

func (*recordingCommandScenarioRunner) ApplicationID() string { return "order_entry" }

func (r *recordingCommandScenarioRunner) RunCommand(
	_ context.Context,
	input commandScenarioInput,
) (commandScenarioResult, error) {
	r.calls = append(r.calls, input)
	return r.result, r.err
}
