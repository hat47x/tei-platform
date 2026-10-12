package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	webcomposition "tei/reference/web-composition/go-host"
)

// This vertical test intentionally uses a synthetic session/CSRF/authorization
// verifier and the real existing process plugin. It is not an OIDC, browser,
// canonical authorization, or durable database-effect conformance test.
func TestProtectedWebRouteToSelectedProcessPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference stdio process launcher forbids spaces in executable paths on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	binaryPath := filepath.Join(t.TempDir(), "tei-protected-process-reference")
	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "../reference-process-plugin")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build existing process plugin: %v\n%s", err, output)
	}
	bytes, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bytes)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	pluginID := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	capabilityID := core.MustCanonicalID("process.reference")
	capability := plugin.Capability{
		Kind: plugin.CapabilityDeploymentExecutor, ID: capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	plan := pruntime.EnvironmentPlan{
		Environment: core.MustCanonicalID("reference"),
		PluginLockFingerprint: "protected-reference-lock",
		RuntimePolicyFingerprint: "protected-reference-policy",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: pluginID, DescriptorFingerprint: "protected-reference-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: binaryPath, Args: []string{referenceArgvMarker},
			SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: pluginID, Capability: capability, Boundary: pruntime.BoundaryProcess,
		}},
	}
	launcher := &countingLauncher{}
	adapter := pruntime.DeploymentExecutor{
		Plan: plan,
		Registry: pruntime.Registry{
			Processes: launcher, Verifier: fileVerifier{path: binaryPath},
		},
		CapabilityID: capabilityID,
	}
	contract := composition.Contract{
		ID: "std.deployment.executor", Version: "1.0.0",
		Shape: "reference-execute-request-result-v1",
	}
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "protected-process-v1", Scope: "web-reference", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("protected-process-v1", "web-reference")
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: selection.ProviderID, ArtifactDigest: digest, Generation: 1,
	}
	permission := true
	denied := errors.New("current host permission denied")
	guard := func(b composition.Binding) error {
		if !permission {
			return denied
		}
		if b != binding {
			return errors.New("host binding does not match selected artifact")
		}
		return nil
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selection, binding, composition.NewScope(), guard)
	if err != nil {
		t.Fatal(err)
	}
	const actionID = "reference.deployment.execute"
	command := executor.Command{
		ID: "protected-web-process", ActionID: actionID,
		IdempotencyKey: "protected-web-process-request",
		Kind: string(deployment.ActionDatabaseRelease),
		Database: core.MustCanonicalID("sales_core"), Version: "1.0.0",
	}
	// These switches stand in for host-owned checks; they are never populated
	// from browser-supplied credentials, headers or request bodies.
	sessionValid, csrfValid, actionAllowed, authorizationFails := true, true, true, false
	principal := webcomposition.Principal{
		ProviderID: "corporate-oidc", Issuer: "https://issuer.example.test",
		Subject: "external-user-1", PrincipalID: "application-principal-1",
	}
	var audit []webcomposition.AuditRecord
	host, err := webcomposition.NewHost(webcomposition.Policy{
		Path: "/reference/deployment", Method: http.MethodPost, ActionID: actionID,
		Providers: map[string]string{"corporate-oidc": "https://issuer.example.test"},
	}, webcomposition.Dependencies{
		VerifySession: func(*http.Request) (webcomposition.Principal, error) {
			if !sessionValid {
				return webcomposition.Principal{}, errors.New("synthetic invalid session")
			}
			return principal, nil
		},
		VerifyCSRF: func(*http.Request, webcomposition.Principal) bool {
			return csrfValid
		},
		Authorize: func(_ context.Context, p webcomposition.Principal, action string) (bool, error) {
			if p != principal || action != actionID {
				return false, errors.New("host identity or action mismatch")
			}
			if authorizationFails {
				return false, errors.New("synthetic authorization backend failure")
			}
			return actionAllowed, nil
		},
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verified, ok := webcomposition.PrincipalFromContext(r.Context())
			if !ok || verified != principal {
				http.Error(w, "trusted principal unavailable", http.StatusInternalServerError)
				return
			}
			result, err := invokeSelected(r.Context(), coordinator, adapter, command)
			if err != nil {
				// No assertion that a failed/unknown dispatch undid an external
				// effect, and no untrusted error detail is returned to clients.
				http.Error(w, "dispatch unavailable; inspect state before retry", http.StatusServiceUnavailable)
				return
			}
			if result.Outcome != executor.OutcomeAccepted ||
				result.ActionID != actionID ||
				result.IdempotencyKey != command.IdempotencyKey ||
				!strings.Contains(result.Message, referenceArgvMarker) ||
				result.PluginInvocation == nil ||
				result.PluginInvocation.ArtifactDigest != digest ||
				!result.PluginInvocation.DigestVerified ||
				result.PluginInvocation.SignatureVerified {
				http.Error(w, "unverified process acknowledgement", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("{\"outcome\":\"accepted\",\"durability\":\"unconfirmed\"}"))
		}),
		Audit: func(record webcomposition.AuditRecord) {
			audit = append(audit, record)
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		session, csrf, authorize, authError, permit bool
		wantStatus int
		wantStage string
		wantCalls int
	}{
		{"unauthenticated", false, true, true, false, true, http.StatusUnauthorized, "rejected", 0},
		{"csrf-denied", true, false, true, false, true, http.StatusForbidden, "rejected", 0},
		{"action-denied", true, true, false, false, true, http.StatusForbidden, "rejected", 0},
		{"authorization-error", true, true, true, true, true, http.StatusServiceUnavailable, "error", 0},
		{"host-resource-revoked", true, true, true, false, false, http.StatusServiceUnavailable, "error", 0},
		{"authorized-spoofed-headers-ignored", true, true, true, false, true, http.StatusAccepted, "completed", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionValid, csrfValid, actionAllowed = tt.session, tt.csrf, tt.authorize
			authorizationFails, permission = tt.authError, tt.permit
			before, audits := launcher.calls, len(audit)
			request := httptest.NewRequest(http.MethodPost, "/reference/deployment", strings.NewReader(
				"{\"principalId\":\"attacker\",\"providerId\":\"attacker\"}",
			))
			request.Header.Set("X-Principal-Id", "attacker")
			request.Header.Set("X-Provider-Id", "attacker")
			response := httptest.NewRecorder()
			host.ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if launcher.calls-before != tt.wantCalls {
				t.Fatalf("process invocation count: got %d, want %d", launcher.calls-before, tt.wantCalls)
			}
			if len(audit) != audits+1 || audit[audits].Stage != tt.wantStage || audit[audits].Status != tt.wantStatus {
				t.Fatalf("missing/wrong audit for %s: %+v", tt.name, audit)
			}
			if tt.wantStatus == http.StatusAccepted && !strings.Contains(response.Body.String(), "\"durability\":\"unconfirmed\"") {
				t.Fatal("an accepted process response must not assert durable completion")
			}
		})
	}

	// Removing the selected provider invalidates the generation even if a
	// previously authorized session and unchanged artifact are still present.
	sessionValid, csrfValid, actionAllowed, authorizationFails, permission = true, true, true, false, true
	if _, err := registry.Apply(snapshot.Revision, nil, []string{"protected-process-v1"}); err != nil {
		t.Fatal(err)
	}
	before, audits := launcher.calls, len(audit)
	response := httptest.NewRecorder()
	host.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/reference/deployment", nil))
	if response.Code != http.StatusServiceUnavailable || launcher.calls != before {
		t.Fatalf("withdrawn provider dispatched: status=%d calls=%d before=%d", response.Code, launcher.calls, before)
	}
	if len(audit) != audits+1 || audit[audits].Stage != "error" {
		t.Fatalf("withdrawal must be audited as an error: %+v", audit)
	}
}
