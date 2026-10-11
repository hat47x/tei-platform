package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/security/legacyauthorization"
	webcomposition "tei/reference/web-composition/go-host"
)

// These cases use the real reference Workspace, Host route, Runner and
// security.authorize Capability. The mutation is synthetic and occurs after
// the first business authorization but before the Command's effect boundary.
func TestSessionCommandRechecksBusinessPermissionAtEffectBoundary(t *testing.T) {
	for _, tc := range []struct {
		name       string
		brokenRule bool
		wantStage  string
		wantHTTP   int
		wantAudit  string
	}{
		{"role permission revoked", false, "authorization-denied", http.StatusForbidden, "rejected"},
		{"authorization adapter becomes unavailable", true, "effect-authorization", http.StatusServiceUnavailable, "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projection, runner, read := sessionWebProjectionAndRunner(t)
			if !tc.brokenRule {
				path := filepath.Join(t.TempDir(), "revoked-authorization.json")
				if err := os.WriteFile(path, []byte(`{"schema":"test-authorization-revoked","rules":[]}`), 0600); err != nil {
					t.Fatal(err)
				}
				revoked, err := legacyauthorization.LoadFixtureRule(path)
				if err != nil {
					t.Fatal(err)
				}
				// The actual pointer is assigned at the third live SubjectLink lookup.
				configureAndAssertEffectReauthorization(t, projection, runner, read, tc, func() {
					runner.authorization = revoked
				})
				return
			}
			configureAndAssertEffectReauthorization(t, projection, runner, read, tc, func() {
				runner.authorization = nil // typed nil Rule should fail closed
			})
		})
	}
}

func configureAndAssertEffectReauthorization(
	t *testing.T,
	projection webTargetProjection,
	runner *referenceCommandScenarioRunner,
	read readScenarioRunner,
	tc struct {
		name       string
		brokenRule bool
		wantStage  string
		wantHTTP   int
		wantAudit  string
	},
	revoke func(),
) {
	t.Helper()
	config := syntheticProtectedSessionConfiguration(runner)
	originalLinks := config.SubjectLinks
	calls := 0
	config.SubjectLinks = func(ctx context.Context, external webcomposition.ExternalSubject) (webcomposition.SubjectLink, bool, error) {
		calls++
		if calls == 3 {
			revoke()
		}
		return originalLinks(ctx, external)
	}
	var audit []webcomposition.AuditRecord
	config.Audit = func(record webcomposition.AuditRecord) { audit = append(audit, record) }
	server, err := newWebTargetServer(projection, read, runner, withProtectedSessionCommand(config))
	if err != nil {
		t.Fatal(err)
	}
	rec := postSyntheticSession(server.Handler(), "/scenario/session-submit", "corporate-editor", "matching",
		`{"from":"2026-09-18T08:30:00Z","row":0}`)
	if rec.Code != tc.wantHTTP || calls != 3 {
		t.Fatalf("HTTP=%d want=%d mapping calls=%d body=%s", rec.Code, tc.wantHTTP, calls, rec.Body.String())
	}
	var response webCommandScenarioErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.Stage != tc.wantStage || response.Result.Authentication != nil ||
		response.Result.Authorization == nil || response.Result.Authorization.Outcome != "allowed" ||
		response.Result.Command != nil || response.Result.Observation != nil ||
		len(response.Result.Validations) != 3 ||
		len(audit) != 1 || audit[0].Stage != tc.wantAudit || audit[0].Status != tc.wantHTTP {
		t.Fatalf("permission drift reached the effect or corrupts evidence: response=%+v audit=%+v", response, audit)
	}
	if tc.brokenRule {
		if response.Result.EffectAuthorization != nil ||
			response.ExecutionError != "trusted identity or authorization resolution unavailable" ||
			strings.Contains(rec.Body.String(), "legacy authorization") {
			t.Fatalf("effect-time authorization backend error leaked: %+v", response)
		}
	} else if response.Result.EffectAuthorization == nil ||
		response.Result.EffectAuthorization.Outcome != "denied" ||
		response.ExecutionError != "" {
		t.Fatalf("effect-time permission denial was not retained as a business outcome: %+v", response)
	}
}
