package webcomposition

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

type referenceBindingResolver func(context.Context, realization.CustomGoBinding) (capabilityinvoke.Adapter, error)

func (f referenceBindingResolver) ResolveAdapter(ctx context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	return f(ctx, binding)
}

func referenceAuthorizationDependencies(t *testing.T) (driver.Resolver, driver.BindingAdapterResolver) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate reference fixtures")
	}
	reference := filepath.Join(filepath.Dir(filename), "..", "..")
	loaded, err := workspace.Load(filepath.Join(reference, "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	users, err := legacyauth.LoadFixtureStore(filepath.Join(reference, "legacy-auth", "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := legacyauthorization.LoadFixtureRule(filepath.Join(reference, "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	adapter := legacyauthorization.Adapter{Store: users, Rule: rules}
	return driver.NewResolver(loaded.World), referenceBindingResolver(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		if binding.Identity() != "order_entry/security.authorize" {
			return nil, errors.New("unexpected or missing canonical authorization binding")
		}
		return adapter, nil
	})
}

func TestCanonicalAuthorizationUsesExistingBusinessRulesAcrossIdentityProviders(t *testing.T) {
	resolver, adapters := referenceAuthorizationDependencies(t)
	corporateEditor := Principal{ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1", Subject: "employee-oidc-sub", PrincipalID: "employee-1"}
	legacyEditor := Principal{ProviderID: "legacy", Issuer: "legacy:order_entry", Subject: "legacy-editor", PrincipalID: "employee-1"}
	// A different external origin links to a different application principal.
	corporateViewer := Principal{ProviderID: "corporate", Issuer: corporateEditor.Issuer, Subject: "viewer-oidc-sub", PrincipalID: "employee-viewer-1"}
	// A malformed session principal must not override the authoritative link.
	spoofedViewer := corporateViewer
	spoofedViewer.PrincipalID = "employee-1"
	unknown := Principal{ProviderID: "corporate", Issuer: corporateEditor.Issuer, Subject: "unmapped-sub", PrincipalID: "employee-1"}
	links := map[ExternalSubject]SubjectLink{
		{ProviderID: corporateEditor.ProviderID, Issuer: corporateEditor.Issuer, Subject: corporateEditor.Subject}: {PrincipalID: corporateEditor.PrincipalID, SubjectID: "subject-order-editor"},
		{ProviderID: legacyEditor.ProviderID, Issuer: legacyEditor.Issuer, Subject: legacyEditor.Subject}:          {PrincipalID: legacyEditor.PrincipalID, SubjectID: "subject-order-editor"},
		{ProviderID: corporateViewer.ProviderID, Issuer: corporateViewer.Issuer, Subject: corporateViewer.Subject}: {PrincipalID: corporateViewer.PrincipalID, SubjectID: "subject-order-viewer"},
	}
	revoked := false
	lookupError := false
	lookupCalls := 0
	authorizer, err := NewCanonicalActionAuthorizer(resolver, "order_entry", "order_entry.submit_order",
		func(_ context.Context, origin ExternalSubject) (SubjectLink, bool, error) {
			lookupCalls++
			if lookupError {
				return SubjectLink{}, false, errors.New("subject link service unavailable")
			}
			if revoked && origin.Subject == corporateEditor.Subject {
				return SubjectLink{}, false, nil
			}
			link, ok := links[origin]
			return link, ok, nil
		}, adapters)
	if err != nil {
		t.Fatal(err)
	}
	current := corporateEditor
	audits := []AuditRecord{}
	calls := 0
	host, err := NewHost(Policy{
		Path:      "/scenario/submit",
		Method:    http.MethodPost,
		ActionID:  "order_entry.submit_order",
		Providers: map[string]string{"corporate": corporateEditor.Issuer, "legacy": legacyEditor.Issuer},
	}, Dependencies{
		VerifySession: func(*http.Request) (Principal, error) { return current, nil }, // synthetic verifier, not real IdP
		VerifyCSRF:    func(*http.Request, Principal) bool { return true },            // synthetic only
		Authorize:     authorizer,
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			verified, ok := PrincipalFromContext(r.Context())
			if !ok || verified != current {
				t.Errorf("Action did not receive verified principal: %+v", verified)
			}
			w.WriteHeader(http.StatusAccepted)
		}),
		Audit: func(a AuditRecord) { audits = append(audits, a) },
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                 string
		principal            Principal
		revoked, lookupError bool
		want                 int
	}{
		{"corporate editor allowed", corporateEditor, false, false, http.StatusAccepted},
		{"legacy editor allowed", legacyEditor, false, false, http.StatusAccepted},
		{"viewer denied under existing business rules", corporateViewer, false, false, http.StatusForbidden},
		{"mismatched verified principal is an integrity error", spoofedViewer, false, false, http.StatusServiceUnavailable},
		{"missing link denied", unknown, false, false, http.StatusForbidden},
		{"revoked link denied", corporateEditor, true, false, http.StatusForbidden},
		{"link service failure is an error", corporateEditor, false, true, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current = tc.principal
			revoked, lookupError = tc.revoked, tc.lookupError
			beforeCalls, beforeLinks, beforeAudits := calls, lookupCalls, len(audits)
			w := request(host, "/scenario/submit", http.MethodPost, "matching")
			if w.Code != tc.want {
				t.Fatalf("got HTTP %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
			if lookupCalls != beforeLinks+1 || len(audits) != beforeAudits+1 {
				t.Fatalf("subject link and audit must run once: links=%d audits=%d", lookupCalls-beforeLinks, len(audits)-beforeAudits)
			}
			if (tc.want == http.StatusAccepted) != (calls == beforeCalls+1) {
				t.Fatalf("Action ran in wrong permission state: calls=%d", calls-beforeCalls)
			}
			if audits[len(audits)-1].Status != tc.want {
				t.Fatalf("audit status does not match: %+v", audits[len(audits)-1])
			}
		})
	}
	// A different action cannot reuse this command's grant.
	if ok, err := authorizer(context.Background(), corporateEditor, "order_entry.other"); err == nil || ok {
		t.Fatalf("authorizer accepted a different action: allowed=%v err=%v", ok, err)
	}
}

func TestCanonicalAuthorizationRejectsUnresolvedCommandsAndMissingDependencies(t *testing.T) {
	resolver, adapters := referenceAuthorizationDependencies(t)
	link := SubjectLinkLookup(func(context.Context, ExternalSubject) (SubjectLink, bool, error) {
		return SubjectLink{PrincipalID: "employee-1", SubjectID: "subject-order-editor"}, true, nil
	})
	if _, err := NewCanonicalActionAuthorizer(resolver, "order_entry", "missing.command", link, adapters); err == nil {
		t.Fatal("accepted missing command Binding")
	}
	if _, err := NewCanonicalActionAuthorizer(resolver, "order_entry", "order_entry.submit_order", nil, adapters); err == nil {
		t.Fatal("accepted missing subject linker")
	}
	if _, err := NewCanonicalActionAuthorizer(resolver, "order_entry", "order_entry.submit_order", link, nil); err == nil {
		t.Fatal("accepted missing Capability Binding adapter resolver")
	}
}
