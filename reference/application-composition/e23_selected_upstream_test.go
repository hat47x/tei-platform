// E22 binds an Action selected by the existing composition registry to
// the existing HTTP Host, generic Action handler, and delegated HTTPS Store.
// Both credentials and the remote receiver are deliberately test fixtures.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/driver"
	"tei/internal/workspace"
	webcomposition "tei/reference/web-composition/go-host"
	"tei/tools/reference-sui-action-boundary/actionboundary"
)

type e23SessionSubject struct{}

func (e23SessionSubject) Subject(ctx context.Context) (string, error) {
	p, ok := webcomposition.PrincipalFromContext(ctx)
	if !ok {
		return "", errors.New("no verified Host principal")
	}
	return p.PrincipalID, nil
}

type e23SessionTenant struct{}

func (e23SessionTenant) Tenant(ctx context.Context) (string, error) {
	p, ok := webcomposition.PrincipalFromContext(ctx)
	if !ok || p.Issuer != "https://issuer.example.invalid/tenant-1" {
		return "", errors.New("no verified reference tenant")
	}
	return "tenant-1", nil
}

type e23Delegation struct {
	mismatch bool
	calls    int
}

func (d *e23Delegation) Delegation(ctx context.Context, in actionboundary.Intent) (actionboundary.TrustedDelegation, error) {
	d.calls++
	p, ok := webcomposition.PrincipalFromContext(ctx)
	if !ok || in.ApplicationID != "sui" || in.ActionID != "sui.move" {
		return actionboundary.TrustedDelegation{}, errors.New("unverified or unbound delegation")
	}
	subject := p.PrincipalID
	if d.mismatch {
		subject = "different-human"
	}
	return actionboundary.TrustedDelegation{
		SubjectID: subject, TenantID: "tenant-1", Audience: "sui-test-audience",
		BearerToken: "delegated-fixture-credential",
	}, nil
}

type e23Harness struct {
	protected   http.Handler
	selected    http.Handler
	registry    *composition.Registry
	revision    string
	upstream    int
	guardCalls  int
	delegation  *e23Delegation
	sessionOK   bool
	csrfOK      bool
	allowed     bool
	guardOK     bool
	outcomeLost bool
	audits      []webcomposition.AuditRecord
}

func e23Setup(t *testing.T) *e23Harness {
	t.Helper()
	e := &e23Harness{
		revision: "sui-v1", delegation: &e23Delegation{},
		sessionOK: true, csrfOK: true, allowed: true, guardOK: true,
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.upstream++
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/docs/doc-1/action-commit" ||
			r.Header.Get("Authorization") != "Bearer delegated-fixture-credential" ||
			r.Header.Get("Cookie") != "" || r.Header.Get("X-Actor-Ref") != "" ||
			r.Header.Get("X-Forwarded-User") != "" || r.Header.Get("X-Provider-ID") != "" {
			t.Errorf("untrusted transport content passed to upstream: path=%s headers=%v", r.URL.EscapedPath(), r.Header)
		}
		var intent struct {
			Version  string                     `json:"protocolVersion"`
			App      string                     `json:"applicationID"`
			Resource string                     `json:"resourceID"`
			Action   string                     `json:"actionID"`
			Expected string                     `json:"expectedRevision"`
			Payload  actionboundary.MovePayload `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&intent); err != nil ||
			intent.Version != "1" || intent.App != "sui" ||
			intent.Resource != "doc-1" || intent.Action != "sui.move" ||
			intent.Payload.CardID != "card-1" || intent.Payload.X != 12 || intent.Payload.Y != 34 {
			t.Errorf("unexpected upstream Action envelope: %+v err=%v", intent, err)
			http.Error(w, "invalid", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if intent.Expected != e.revision {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"protocolVersion":"1","error":"revision_conflict"}`))
			return
		}
		e.revision = "sui-v2"
		if e.outcomeLost {
			// Upstream might already have committed; Host cannot retry.
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"protocolVersion":"1","error":"execution_failed"}`))
			return
		}
		_, _ = w.Write([]byte(`{"protocolVersion":"1","revision":"sui-v2"}`))
	}))
	t.Cleanup(server.Close)

	store := actionboundary.HTTPSApplicationStore{
		ApplicationID: "sui", ActionID: "sui.move",
		Origin: server.URL, Prefix: "/docs/", Suffix: "/action-commit",
		Client: server.Client(), Audience: "sui-test-audience",
		Subjects: e23SessionSubject{}, Tenants: e23SessionTenant{},
		Delegations: e.delegation,
	}
	referenceHandler, _, _ := e20Fixture(t, false)
	// E22's temporary parsed DSL shell is the canonical SUI candidate.
	// No SUI fake resolver is used for the declared Action lookup.
	candidate, err := workspace.Load(e22Workspace(t, true, false))
	if err != nil {
		t.Fatalf("load native candidate Workspace: %v", err)
	}
	referenceHandler.Host.CommandGate = &actionboundary.CanonicalCommandGate{
		Resolver: driver.NewResolver(candidate.World),
	}
	referenceHandler.Host.Bindings = []actionboundary.Binding{{
		ApplicationID: "sui", ActionID: "sui.move", Store: store,
	}}
	// E20 provides the actual order_entry resolver and the SUI test-only
	// candidate, without creating a new Application or Capability in TEI.
	decoder := referenceHandler
	actionEndpoint := http.HandlerFunc(decoder.ServeHTTP)

	const action = "sui.move"
	const version = "1.0.0"
	contract := composition.Contract{ID: action, Version: version, Shape: "sui-candidate-move-v1"}
	registry := composition.NewRegistry()
	_, err = registry.Apply(0, []composition.Provider{{
		ID: "sui-https-provider", Scope: "reference/sui",
		Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("sui-https-provider", "reference/sui")
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{
		ContractID: action, ContractVersion: version,
		ProviderID:     "sui-https-provider",
		ArtifactDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1,
	}
	coordinator, err := composition.NewSelectedCoordinator(
		registry, selection, binding, composition.NewScope(),
		func(actual composition.Binding) error {
			e.guardCalls++
			if !e.guardOK || actual != binding {
				return errors.New("selected provider no longer authorized")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := webcomposition.NewSelectedActionInvoker(
		coordinator, action, version, []webcomposition.SelectedActionProvider{{
			ProviderID: binding.ProviderID, ArtifactDigest: binding.ArtifactDigest,
			Invoke: actionEndpoint,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	e.selected = selected
	e.registry = registry
	principal := webcomposition.Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant-1",
		Subject: "human-subject-1", PrincipalID: "human-1",
	}
	e.protected, err = webcomposition.NewHost(webcomposition.Policy{
		Path: "/action/sui/move", Method: http.MethodPost, ActionID: action,
		Providers: map[string]string{"corporate": principal.Issuer},
	}, webcomposition.Dependencies{
		VerifySession: func(*http.Request) (webcomposition.Principal, error) {
			if !e.sessionOK {
				return webcomposition.Principal{}, errors.New("session invalid")
			}
			return principal, nil
		},
		VerifyCSRF: func(*http.Request, webcomposition.Principal) bool { return e.csrfOK },
		Authorize: func(_ context.Context, _ webcomposition.Principal, actual string) (bool, error) {
			return e.allowed && actual == action, nil
		},
		Invoke: selected,
		Audit:  func(record webcomposition.AuditRecord) { e.audits = append(e.audits, record) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func e23Send(t *testing.T, target http.Handler, revision, app string) int {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"protocolVersion": "1", "applicationID": app,
		"resourceID": "doc-1", "actionID": "sui.move",
		"expectedRevision": revision,
		"payload":          map[string]any{"cardId": "card-1", "x": 12, "y": 34},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost,
		"http://tei.example.test/action/sui/move", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TEI-Action", "commit")
	req.Header.Set("X-Provider-ID", "attacker-provider")
	req.Header.Set("X-Actor-Ref", "admin")
	req.Header.Set("X-Forwarded-User", "admin")
	recorder := httptest.NewRecorder()
	target.ServeHTTP(recorder, req)
	return recorder.Code
}

func TestE23SelectedSUIHTTPSActionRequiresEveryBoundary(t *testing.T) {
	e := e23Setup(t)
	baselineGuards := e.guardCalls
	if got := e23Send(t, e.selected, "sui-v1", "sui"); got != http.StatusForbidden ||
		e.guardCalls != baselineGuards || e.upstream != 0 {
		t.Fatalf("bare selected provider reachable: status=%d guard=%d->%d upstream=%d", got, baselineGuards, e.guardCalls, e.upstream)
	}
	for _, tc := range []struct {
		name  string
		apply func()
		reset func()
		want  int
	}{
		{"no session", func() { e.sessionOK = false }, func() { e.sessionOK = true }, http.StatusUnauthorized},
		{"CSRF rejected", func() { e.csrfOK = false }, func() { e.csrfOK = true }, http.StatusForbidden},
		{"Action denied", func() { e.allowed = false }, func() { e.allowed = true }, http.StatusForbidden},
		{"selected guard revoked", func() { e.guardOK = false }, func() { e.guardOK = true }, http.StatusServiceUnavailable},
		{"delegated subject mismatch", func() { e.delegation.mismatch = true }, func() { e.delegation.mismatch = false }, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.apply()
			before := e.upstream
			got := e23Send(t, e.protected, "sui-v1", "sui")
			tc.reset()
			if got != tc.want || e.upstream != before {
				t.Fatalf("denial: status=%d expected=%d upstream=%d->%d", got, tc.want, before, e.upstream)
			}
		})
	}
	if got := e23Send(t, e.protected, "sui-v1", "order_entry"); got != http.StatusBadRequest ||
		e.upstream != 0 {
		t.Fatalf("cross-application Action reached upstream: status=%d upstream=%d", got, e.upstream)
	}
	if got := e23Send(t, e.protected, "sui-v1", "sui"); got != http.StatusOK ||
		e.revision != "sui-v2" || e.upstream != 1 {
		t.Fatalf("authorized selected Action failed: status=%d revision=%s upstream=%d", got, e.revision, e.upstream)
	}
	if got := e23Send(t, e.protected, "sui-v1", "sui"); got != http.StatusConflict ||
		e.upstream != 2 || e.revision != "sui-v2" {
		t.Fatalf("stale write was accepted: status=%d upstream=%d revision=%s", got, e.upstream, e.revision)
	}
	_, err := e.registry.Apply(1, nil, []string{"sui-https-provider"})
	if err != nil {
		t.Fatal(err)
	}
	if got := e23Send(t, e.protected, "sui-v2", "sui"); got != http.StatusServiceUnavailable ||
		e.upstream != 2 {
		t.Fatalf("withdrawn provider invoked: status=%d upstream=%d", got, e.upstream)
	}
	if len(e.audits) == 0 {
		t.Fatal("protected Host did not audit")
	}
}

func TestE23UnknownRemoteOutcomeIsNeverAutomaticallyRetried(t *testing.T) {
	e := e23Setup(t)
	e.outcomeLost = true
	got := e23Send(t, e.protected, "sui-v1", "sui")
	if got != http.StatusInternalServerError || e.upstream != 1 || e.revision != "sui-v2" {
		t.Fatalf("uncertain result was retried or accepted: HTTP %d sends=%d revision=%s",
			got, e.upstream, e.revision)
	}
	// The client must resolve an indeterminate result through its own
	// authorized observation path, not by blindly replaying a committed Move.
	if e.delegation.calls != 1 {
		t.Fatalf("unexpected implicit delegation retry count=%d", e.delegation.calls)
	}
}
