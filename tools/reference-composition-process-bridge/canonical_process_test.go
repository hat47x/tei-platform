package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"tei/internal/driver"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
	webcomposition "tei/reference/web-composition/go-host"
)

// This is a transport acceptance probe, NOT the implementation of order_entry:
// the existing process plugin understands deployment.execute only. Its 202
// acknowledges a synthetic intent; it does not create or commit an order.
type canonicalProcessBindingResolver struct {
	adapter legacyauthorization.Adapter
}

func (r canonicalProcessBindingResolver) ResolveAdapter(_ context.Context, b realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	if b.Identity() != "order_entry/security.authorize" {
		return nil, errors.New("canonical authorization binding missing")
	}
	return r.adapter, nil
}

type canonicalProcessRecorder struct {
	calls int
	last  pruntime.Request
}

func (r *canonicalProcessRecorder) Invoke(ctx context.Context, command pruntime.ProcessCommand, req pruntime.Request) (pruntime.Response, error) {
	r.calls++
	r.last = req
	return (pruntime.StdioProcessLauncher{}).Invoke(ctx, command, req)
}

// Integration with actual order_entry Workspace/Driver/Canonical authorization,
// Host-only Action approval, selected Composition generation, digest-verified
// external process and the legacy deployment transport. All identity, session,
// CSRF and grant mutation is synthetic; no business effect occurs.
func TestCanonicalAuthorizationGatesRealSelectedProcessReceipt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference process launcher does not allow whitespace in executable paths")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "canonical-process-receiver")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../plugins/reference-process-executor")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build independent plugin: %v\n%s", err, output)
	}
	raw, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	loaded, err := workspace.Load("../../reference/workspace")
	if err != nil {
		t.Fatal(err)
	}
	users, err := legacyauth.LoadFixtureStore("../../reference/legacy-auth/users.json")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := legacyauthorization.LoadFixtureRule("../../reference/legacy-auth/authorization.json")
	if err != nil {
		t.Fatal(err)
	}
	const actionID = "order_entry.submit_order"
	corporate := webcomposition.Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1",
		Subject: "corporate-editor", PrincipalID: "employee-1",
	}
	legacy := webcomposition.Principal{
		ProviderID: "legacy", Issuer: "legacy:order_entry",
		Subject: "legacy-editor", PrincipalID: "employee-1",
	}
	viewer := webcomposition.Principal{
		ProviderID: "corporate", Issuer: corporate.Issuer,
		Subject: "corporate-viewer", PrincipalID: "employee-viewer-1",
	}
	links := map[webcomposition.ExternalSubject]webcomposition.SubjectLink{
		{ProviderID: corporate.ProviderID, Issuer: corporate.Issuer, Subject: corporate.Subject}: {
			PrincipalID: corporate.PrincipalID, SubjectID: "subject-order-editor",
		},
		{ProviderID: legacy.ProviderID, Issuer: legacy.Issuer, Subject: legacy.Subject}: {
			PrincipalID: legacy.PrincipalID, SubjectID: "subject-order-editor",
		},
		{ProviderID: viewer.ProviderID, Issuer: viewer.Issuer, Subject: viewer.Subject}: {
			PrincipalID: viewer.PrincipalID, SubjectID: "subject-order-viewer",
		},
	}
	revokeLink, linkUnavailable := false, false
	authorizer, err := webcomposition.NewCanonicalActionAuthorizer(
		driver.NewResolver(loaded.World), "order_entry", "order_entry.submit_order",
		func(_ context.Context, external webcomposition.ExternalSubject) (webcomposition.SubjectLink, bool, error) {
			if linkUnavailable {
				return webcomposition.SubjectLink{}, false, errors.New("private identity store unavailable")
			}
			if revokeLink {
				return webcomposition.SubjectLink{}, false, nil
			}
			link, ok := links[external]
			return link, ok, nil
		},
		canonicalProcessBindingResolver{
			adapter: legacyauthorization.Adapter{Store: users, Rule: rules},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	pluginID := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	capabilityID := core.MustCanonicalID("process.reference")
	capability := plugin.Capability{
		Kind: plugin.CapabilityDeploymentExecutor, ID: capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "canonical-process-reference-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: pluginID, DescriptorFingerprint: "canonical-process-reference-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: binary, Args: []string{referenceArgvMarker},
			SandboxPolicy: plugin.SandboxNone,
			Artifact:      plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: pluginID, Capability: capability, Boundary: pruntime.BoundaryProcess,
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	recorder := &canonicalProcessRecorder{}
	adapter := pruntime.DeploymentExecutor{
		Plan: plan,
		Registry: pruntime.Registry{
			Processes: recorder, Verifier: fileVerifier{path: binary},
		},
		CapabilityID: capabilityID,
	}
	// The selected contract describes this synthetic transport receipt, not
	// a capability to implement the order_entry.submit_order business effect.
	contract := composition.Contract{
		ID: actionID, Version: "1.0.0", Shape: "reference-deployment-ack-only",
	}
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "canonical-probe", Scope: "test/canonical-process", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("canonical-probe", "test/canonical-process")
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: selection.ProviderID, ArtifactDigest: digest, Generation: 1,
	}
	currentHostAuthority := true
	coordinator, err := composition.NewSelectedCoordinator(registry, selection, binding, composition.NewScope(),
		func(actual composition.Binding) error {
			if !currentHostAuthority || actual != binding {
				return errors.New("private host execution authority revoked")
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	command := executor.Command{
		ID: "reference-receipt-only", ActionID: actionID,
		IdempotencyKey: "reference-receipt-intent", Kind: string(deployment.ActionDatabaseRelease),
		Database: core.MustCanonicalID("reference_only"), Version: "1.0.0",
	}
	selected, err := webcomposition.NewSelectedActionInvoker(coordinator, actionID, contract.Version, []webcomposition.SelectedActionProvider{{
		ProviderID: selection.ProviderID, ArtifactDigest: digest,
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := webcomposition.PrincipalFromContext(r.Context())
			if !ok || p.PrincipalID == "" {
				http.Error(w, "missing Host subject", http.StatusServiceUnavailable)
				return
			}
			// Never copy the HTTP body, client headers, raw Principal, session
			// token, or a Host-local approval marker into the process request.
			result, err := adapter.Execute(r.Context(), command)
			if err != nil {
				http.Error(w, "process dispatch uncertain; inspect state before retry", http.StatusServiceUnavailable)
				return
			}
			if result.Outcome != executor.OutcomeAccepted ||
				result.ActionID != actionID ||
				result.PluginInvocation == nil ||
				!result.PluginInvocation.DigestVerified ||
				result.PluginInvocation.ArtifactDigest != digest {
				http.Error(w, "invalid process acknowledgment", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"outcome":"accepted","businessCompletion":"not-performed"}`))
		}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	active := corporate
	sessionValid, csrfValid := true, true
	var audits []webcomposition.AuditRecord
	host, err := webcomposition.NewHost(webcomposition.Policy{
		Path: "/reference/intent", Method: http.MethodPost, ActionID: actionID,
		Providers: map[string]string{
			"corporate": corporate.Issuer, "legacy": legacy.Issuer,
		},
	}, webcomposition.Dependencies{
		VerifySession: func(*http.Request) (webcomposition.Principal, error) {
			if !sessionValid {
				return webcomposition.Principal{}, errors.New("synthetic session expired")
			}
			return active, nil
		},
		VerifyCSRF: func(*http.Request, webcomposition.Principal) bool { return csrfValid },
		Authorize:  authorizer,
		Invoke:     selected,
		Audit:      func(record webcomposition.AuditRecord) { audits = append(audits, record) },
	})
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(t *testing.T, want, newCalls int, stage string) {
		t.Helper()
		before, beforeAudits := recorder.calls, len(audits)
		request := httptest.NewRequest(http.MethodPost, "/reference/intent",
			strings.NewReader(`{"principalId":"admin","role":"admin","providerId":"forged","authorization":"secret-from-browser"}`))
		request.Header.Set("Authorization", "Bearer secret-from-browser")
		request.Header.Set("X-Principal-ID", "admin")
		request.Header.Set("X-Action-ID", "order_entry.submit_order")
		response := httptest.NewRecorder()
		host.ServeHTTP(response, request)
		if response.Code != want || recorder.calls != before+newCalls {
			t.Fatalf("status=%d want=%d processCalls=%d want=%d body=%s",
				response.Code, want, recorder.calls-before, newCalls, response.Body.String())
		}
		if len(audits) != beforeAudits+1 || audits[beforeAudits].Stage != stage {
			t.Fatalf("audit mismatch: %+v", audits)
		}
		if newCalls == 1 {
			if !strings.Contains(response.Body.String(), `"businessCompletion":"not-performed"`) {
				t.Fatal("receipt was incorrectly represented as completed business Action")
			}
			wire, err := json.Marshal(recorder.last)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"secret-from-browser", "corporate-editor", "legacy-editor",
				"subject-order-editor", "employee-1", "Bearer", "forged"} {
				if strings.Contains(string(wire), secret) {
					t.Fatalf("raw Host identity or browser data crossed the process boundary (%s)", secret)
				}
			}
		}
	}
	t.Run("corporate-editor-accepted-intent-only", func(t *testing.T) {
		invoke(t, http.StatusAccepted, 1, "completed")
	})
	t.Run("legacy-editor-same-business-rule", func(t *testing.T) {
		active = legacy
		invoke(t, http.StatusAccepted, 1, "completed")
	})
	t.Run("viewer-denied-by-canonical-business-rule", func(t *testing.T) {
		active = viewer
		invoke(t, http.StatusForbidden, 0, "rejected")
	})
	t.Run("subject-link-revoked", func(t *testing.T) {
		active, revokeLink = corporate, true
		invoke(t, http.StatusForbidden, 0, "rejected")
		revokeLink = false
	})
	t.Run("subject-link-service-failed", func(t *testing.T) {
		linkUnavailable = true
		invoke(t, http.StatusServiceUnavailable, 0, "error")
		linkUnavailable = false
	})
	t.Run("csrf-denied-before-provider", func(t *testing.T) {
		csrfValid = false
		invoke(t, http.StatusForbidden, 0, "rejected")
		csrfValid = true
	})
	t.Run("session-denied-before-provider", func(t *testing.T) {
		sessionValid = false
		invoke(t, http.StatusUnauthorized, 0, "rejected")
		sessionValid = true
	})
	t.Run("host-Authority-revoked-before-process", func(t *testing.T) {
		currentHostAuthority = false
		invoke(t, http.StatusServiceUnavailable, 0, "error")
		currentHostAuthority = true
	})
	t.Run("artifact-verifier-unavailable", func(t *testing.T) {
		// Runtime's own safety boundary fails before actual child-process
		// launch, independent of the Web authorization and composition guard.
		adapter.Registry.Verifier = nil
		invoke(t, http.StatusServiceUnavailable, 0, "error")
		adapter.Registry.Verifier = fileVerifier{path: binary}
	})
	t.Run("provider-withdrawn", func(t *testing.T) {
		if _, err := registry.Apply(snapshot.Revision, nil, []string{"canonical-probe"}); err != nil {
			t.Fatal(err)
		}
		invoke(t, http.StatusServiceUnavailable, 0, "error")
	})
}
