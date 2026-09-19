package main

import (
	"context"
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

type recordingCommandScenarioRunner struct {
	calls  []commandScenarioInput
	result commandScenarioResult
	err    error
}

func (r *recordingCommandScenarioRunner) RunCommand(
	_ context.Context,
	input commandScenarioInput,
) (commandScenarioResult, error) {
	r.calls = append(r.calls, input)
	return r.result, r.err
}
