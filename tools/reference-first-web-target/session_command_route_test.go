package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/workspace"
	webcomposition "tei/reference/web-composition/go-host"
)

func sessionWebProjectionAndRunner(t *testing.T) (webTargetProjection, *referenceCommandScenarioRunner, readScenarioRunner) {
	t.Helper()
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
	runner, err := newReferenceCommandScenarioRunner(loaded.World,
		legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	return projection, runner, newReferenceReadScenarioRunner(loaded.World)
}

// These are explicitly synthetic fixtures. The example header is not a
// recommended real session verifier and must never be deployed as one.
func syntheticProtectedSessionConfiguration(runner *referenceCommandScenarioRunner) protectedSessionCommandConfig {
	corporateIssuer := "https://issuer.example.invalid/tenant1"
	identities := map[string]webcomposition.Principal{
		"corporate-editor": {ProviderID: "corporate", Issuer: corporateIssuer, Subject: "oidc-editor", PrincipalID: "employee-1"},
		"corporate-viewer": {ProviderID: "corporate", Issuer: corporateIssuer, Subject: "oidc-viewer", PrincipalID: "employee-2"},
		"legacy-editor":    {ProviderID: "legacy", Issuer: "legacy:order_entry", Subject: "legacy-editor", PrincipalID: "employee-1"},
		"corporate-locked": {ProviderID: "corporate", Issuer: corporateIssuer, Subject: "oidc-locked", PrincipalID: "employee-locked"},
		"corporate-disabled": {ProviderID: "corporate", Issuer: corporateIssuer, Subject: "oidc-disabled", PrincipalID: "employee-disabled"},
	}
	links := map[webcomposition.ExternalSubject]webcomposition.SubjectLink{
		{ProviderID: "corporate", Issuer: corporateIssuer, Subject: "oidc-editor"}:     {PrincipalID: "employee-1", SubjectID: "subject-order-editor"},
		{ProviderID: "corporate", Issuer: corporateIssuer, Subject: "oidc-viewer"}:     {PrincipalID: "employee-2", SubjectID: "subject-order-viewer"},
		{ProviderID: "legacy", Issuer: "legacy:order_entry", Subject: "legacy-editor"}: {PrincipalID: "employee-1", SubjectID: "subject-order-editor"},
		{ProviderID: "corporate", Issuer: corporateIssuer, Subject: "oidc-locked"}: {PrincipalID: "employee-locked", SubjectID: "subject-order-locked"},
		{ProviderID: "corporate", Issuer: corporateIssuer, Subject: "oidc-disabled"}: {PrincipalID: "employee-disabled", SubjectID: "subject-order-disabled"},
	}
	return protectedSessionCommandConfig{
		Runner:    runner,
		Providers: map[string]string{"corporate": corporateIssuer, "legacy": "legacy:order_entry"},
		VerifySession: func(r *http.Request) (webcomposition.Principal, error) {
			identity, ok := identities[r.Header.Get("X-Synthetic-Session")]
			if !ok {
				return webcomposition.Principal{}, errors.New("session missing")
			}
			return identity, nil
		},
		VerifyCSRF: func(r *http.Request, _ webcomposition.Principal) bool {
			return r.Header.Get("X-Synthetic-CSRF") == "matching"
		},
		SubjectLinks: func(_ context.Context, subject webcomposition.ExternalSubject) (webcomposition.SubjectLink, bool, error) {
			link, ok := links[subject]
			return link, ok, nil
		},
		AccountNow: func() time.Time { return runner.now }, // deterministic fixture clock, not production expiry
		Audit: func(webcomposition.AuditRecord) {},
	}
}

func postSyntheticSession(h http.Handler, url, identity, csrf, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if identity != "" {
		req.Header.Set("X-Synthetic-Session", identity)
	}
	if csrf != "" {
		req.Header.Set("X-Synthetic-CSRF", csrf)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestWebTargetProtectedSessionRoutePreservesLegacyAndBusinessSemantics(t *testing.T) {
	projection, runner, read := sessionWebProjectionAndRunner(t)
	config := syntheticProtectedSessionConfiguration(runner)
	var records []webcomposition.AuditRecord
	config.Audit = func(record webcomposition.AuditRecord) { records = append(records, record) }
	server, err := newWebTargetServer(projection, read, runner, withProtectedSessionCommand(config))
	if err != nil {
		t.Fatal(err)
	}
	h := server.Handler()
	for _, id := range []string{"corporate-editor", "legacy-editor"} {
		t.Run(id, func(t *testing.T) {
			rec := postSyntheticSession(h, "/scenario/session-submit", id, "matching",
				`{"from":"2026-09-18T08:30:00Z","row":0}`)
			if rec.Code != http.StatusOK {
				t.Fatalf("session submit: status=%d response=%s", rec.Code, rec.Body.String())
			}
			var response webCommandScenarioResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			result := response.Result
			if response.Host != projection.Host || result.Stage != "completed" || result.Authentication != nil ||
				result.Read == nil || len(result.Validations) != 3 ||
				result.Authorization == nil || result.Authorization.Outcome != "allowed" ||
				result.Command == nil || result.Observation == nil {
				t.Fatalf("session result does not preserve existing business outcome: %+v", response)
			}
		})
	}
	// The legacy credential path remains independent and continues to run
	// security.authenticate; it is never silently used by the new session path.
	legacy := postSyntheticSession(h, "/scenario/submit", "", "",
		`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`)
	if legacy.Code != http.StatusOK {
		t.Fatalf("legacy submit status=%d body=%s", legacy.Code, legacy.Body.String())
	}
	var response webCommandScenarioResponse
	if err := json.Unmarshal(legacy.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.Authentication == nil || response.Result.Authentication.Outcome != "authenticated" {
		t.Fatalf("legacy authentication lost: %+v", response.Result)
	}
	if len(records) != 2 || records[0].Stage != "completed" || records[1].Stage != "completed" {
		t.Fatalf("synthetic Host auditing failed: %+v", records)
	}
}

func TestWebTargetProtectedSessionRouteDeniesUntrustedAndMalformedRequests(t *testing.T) {
	projection, runner, read := sessionWebProjectionAndRunner(t)
	config := syntheticProtectedSessionConfiguration(runner)
	var audits []webcomposition.AuditRecord
	config.Audit = func(record webcomposition.AuditRecord) { audits = append(audits, record) }
	server, err := newWebTargetServer(projection, read, runner, withProtectedSessionCommand(config))
	if err != nil {
		t.Fatal(err)
	}
	h := server.Handler()
	for _, tc := range []struct {
		name, identity, csrf, body string
		want                       int
	}{
		{"unauthenticated", "", "matching", `{"row":0}`, http.StatusUnauthorized},
		{"fake principal header alone is not session", "not-a-session", "matching", `{"row":0}`, http.StatusUnauthorized},
		{"csrf missing", "corporate-editor", "", `{"row":0}`, http.StatusForbidden},
		{"viewer denied by existing business rule", "corporate-viewer", "matching", `{"row":0}`, http.StatusForbidden},
		{"locked editor denied before business capability", "corporate-locked", "matching", `{"row":0}`, http.StatusForbidden},
		{"disabled account denied", "corporate-disabled", "matching", `{"row":0}`, http.StatusForbidden},
		{"credential rejected on session route", "corporate-editor", "matching", `{"loginId":"legacy-order-editor","credential":"editor-secret","row":0}`, http.StatusBadRequest},
		{"subject cannot come from JSON", "corporate-editor", "matching", `{"subjectId":"subject-order-editor","row":0}`, http.StatusBadRequest},
		{"two JSON objects rejected", "corporate-editor", "matching", `{"row":0} {"row":0}`, http.StatusBadRequest},
		{"JSON null must not invoke a command", "corporate-editor", "matching", `null`, http.StatusBadRequest},
		{"negative row rejected", "corporate-editor", "matching", `{"row":-1}`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := len(audits)
			rec := postSyntheticSession(h, "/scenario/session-submit", tc.identity, tc.csrf, tc.body)
			if rec.Code != tc.want || len(audits) != before+1 {
				t.Fatalf("status=%d want=%d audits=%d", rec.Code, tc.want, len(audits)-before)
			}
			if audits[len(audits)-1].Stage != "rejected" {
				t.Fatalf("request must be audited as rejected: %+v", audits[len(audits)-1])
			}
		})
	}
	req := httptest.NewRequest(http.MethodGet, "/scenario/session-submit", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("session endpoint unexpectedly accepts GET: %d", w.Code)
	}
}

func TestWebTargetSessionRouteIsOptInAndFailsClosed(t *testing.T) {
	projection, runner, read := sessionWebProjectionAndRunner(t)
	server, err := newWebTargetServer(projection, read, runner)
	if err != nil {
		t.Fatal(err)
	}
	rec := postSyntheticSession(server.Handler(), "/scenario/session-submit", "corporate-editor", "matching", `{"row":0}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unconfigured route unexpectedly exposed: %d", rec.Code)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*protectedSessionCommandConfig)
	}{
		{"no verifier", func(c *protectedSessionCommandConfig) { c.VerifySession = nil }},
		{"no csrf", func(c *protectedSessionCommandConfig) { c.VerifyCSRF = nil }},
		{"no subject links", func(c *protectedSessionCommandConfig) { c.SubjectLinks = nil }},
		{"no audit", func(c *protectedSessionCommandConfig) { c.Audit = nil }},
		{"no providers", func(c *protectedSessionCommandConfig) { c.Providers = nil }},
		{"no eligibility clock", func(c *protectedSessionCommandConfig) { c.AccountNow = nil }},
		{"no runner", func(c *protectedSessionCommandConfig) { c.Runner = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := syntheticProtectedSessionConfiguration(runner)
			tc.mutate(&config)
			if _, err := newWebTargetServer(projection, read, runner, withProtectedSessionCommand(config)); err == nil {
				t.Fatal("unsafe session route configuration accepted")
			}
		})
	}
	differentRunner := sessionReferenceRunner(t)
	config := syntheticProtectedSessionConfiguration(differentRunner)
	if _, err := newWebTargetServer(projection, read, runner, withProtectedSessionCommand(config)); err == nil {
		t.Fatal("session route accepted a different command runner")
	}
	config = syntheticProtectedSessionConfiguration(runner)
	if _, err := newWebTargetServer(projection, read, runner,
		withProtectedSessionCommand(config), withProtectedSessionCommand(config)); err == nil {
		t.Fatal("duplicate protected session declarations were silently overwritten")
	}
}
