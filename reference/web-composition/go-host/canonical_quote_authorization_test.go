package webcomposition

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
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

// The accepted reference Workspace is a historical baseline and must not be
// changed just to test a new business Action. Overlay only the candidate files
// in a temporary copy; the existing authorizer still evaluates its own
// explicit fixture rule, never a hand-written second permission evaluator.
func quoteCandidateAuthorizationDependencies(t *testing.T) (driver.Resolver, driver.BindingAdapterResolver) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("quote test cannot locate reference files")
	}
	reference := filepath.Join(filepath.Dir(filename), "..", "..")
	source := filepath.Join(reference, "workspace")
	target := filepath.Join(t.TempDir(), "workspace")
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(reference, "candidates", "order-quote")
	for _, relative := range []string{
		"semantic/sales/quote.tei",
		"applications/order-entry/patterns/preview-order-quote.tei",
		"applications/order-entry/realizations/quote-lines.tei",
	} {
		data, err := os.ReadFile(filepath.Join(candidate, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(target, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := workspace.Load(target)
	if err != nil {
		t.Fatal(err)
	}
	users, err := legacyauth.LoadFixtureStore(filepath.Join(reference, "legacy-auth", "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	originalRules, err := os.ReadFile(filepath.Join(reference, "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Schema string            `json:"schema"`
		Rules  []json.RawMessage `json:"rules"`
	}
	if err := json.Unmarshal(originalRules, &document); err != nil {
		t.Fatal(err)
	}
	newRule, err := os.ReadFile(filepath.Join(candidate, "authorization-rule.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(newRule) {
		t.Fatal("candidate authorization rule is not valid JSON")
	}
	document.Rules = append(document.Rules, json.RawMessage(newRule))
	ruleBytes, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(t.TempDir(), "authorization.json")
	if err := os.WriteFile(rulePath, ruleBytes, 0600); err != nil {
		t.Fatal(err)
	}
	rules, err := legacyauthorization.LoadFixtureRule(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	adapter := legacyauthorization.Adapter{Store: users, Rule: rules}
	return driver.NewResolver(loaded.World), referenceBindingResolver(
		func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			if binding.Identity() != "order_entry/security.authorize" {
				return nil, errors.New("unexpected canonical authorization binding")
			}
			return adapter, nil
		},
	)
}

// The independent quote is a read-only semantic Capability, not a second
// implementation of order_entry.submit_order. No process execution occurs.
func TestStandaloneQuotePatternHasSeparateCanonicalBindingAndAuthorization(t *testing.T) {
	baseline, _ := referenceAuthorizationDependencies(t)
	const quoteAction = "order_entry.preview_order_quote"
	if _, err := baseline.ResolveCommandBinding("order_entry", quoteAction); err == nil {
		t.Fatal("candidate quote was already present in accepted baseline")
	}
	resolver, adapters := quoteCandidateAuthorizationDependencies(t)
	const quoteCapability = "sales.order.quote_lines"
	resolved, err := resolver.ResolveCommandBinding("order_entry", quoteAction)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Pattern.Capability == nil ||
		string(resolved.Pattern.Capability.ID) != quoteCapability ||
		string(resolved.Binding.Binding.Capability) != quoteCapability {
		t.Fatalf("quote Pattern is not independently bound: %+v", resolved)
	}
	if resolved.Binding.Binding.Identity() != "order_entry/sales.order.quote_lines" {
		t.Fatalf("quote Binding identity drifted: %+v", resolved.Binding.Binding)
	}
	meaning := resolved.Pattern.Capability
	if len(meaning.Inputs) != 1 || meaning.Inputs[0].Name != "request" ||
		meaning.Inputs[0].Type != "sales.quote_request" ||
		meaning.Output != "sales.quote_result" || len(meaning.Outcomes) != 0 ||
		meaning.Effect != nil {
		t.Fatalf("read-only quote semantics drifted: %+v", meaning)
	}
	submit, err := resolver.ResolveCommandBinding("order_entry", "order_entry.submit_order")
	if err != nil {
		t.Fatal(err)
	}
	if submit.Pattern.Capability == nil || submit.Pattern.Capability.ID == meaning.ID {
		t.Fatal("quote Capability was collapsed into the order submit Command")
	}

	corporateEditor := Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1",
		Subject: "corporate-editor", PrincipalID: "employee-1",
	}
	legacyEditor := Principal{
		ProviderID: "legacy", Issuer: "legacy:order_entry",
		Subject: "legacy-editor", PrincipalID: "employee-1",
	}
	viewer := Principal{
		ProviderID: "corporate", Issuer: corporateEditor.Issuer,
		Subject: "corporate-viewer", PrincipalID: "employee-2",
	}
	links := map[ExternalSubject]SubjectLink{
		{ProviderID: corporateEditor.ProviderID, Issuer: corporateEditor.Issuer, Subject: corporateEditor.Subject}: {
			PrincipalID: corporateEditor.PrincipalID, SubjectID: "subject-order-editor",
		},
		{ProviderID: legacyEditor.ProviderID, Issuer: legacyEditor.Issuer, Subject: legacyEditor.Subject}: {
			PrincipalID: legacyEditor.PrincipalID, SubjectID: "subject-order-editor",
		},
		{ProviderID: viewer.ProviderID, Issuer: viewer.Issuer, Subject: viewer.Subject}: {
			PrincipalID: viewer.PrincipalID, SubjectID: "subject-order-viewer",
		},
	}
	revoke := false
	unavailable := false
	authorizer, err := NewCanonicalActionAuthorizer(
		resolver, "order_entry", quoteAction,
		func(_ context.Context, key ExternalSubject) (SubjectLink, bool, error) {
			if unavailable {
				return SubjectLink{}, false, errors.New("private mapping service failure")
			}
			if revoke {
				return SubjectLink{}, false, nil
			}
			l, ok := links[key]
			return l, ok, nil
		}, adapters,
	)
	if err != nil {
		t.Fatal(err)
	}
	// It is the quote's own grant, not the submit grant, that is required.
	if ok, err := authorizer(context.Background(), corporateEditor, "order_entry.submit_order"); err == nil || ok {
		t.Fatalf("submit grant reused as quote: allowed=%v err=%v", ok, err)
	}

	current := corporateEditor
	csrfOK := true
	sessionOK := true
	calls := 0
	var records []AuditRecord
	host, err := NewHost(Policy{
		Path:      "/reference/quote",
		Method:    http.MethodPost,
		ActionID:  quoteAction,
		Providers: map[string]string{"corporate": corporateEditor.Issuer, "legacy": legacyEditor.Issuer},
	}, Dependencies{
		VerifySession: func(*http.Request) (Principal, error) {
			if !sessionOK {
				return Principal{}, errors.New("synthetic session invalid")
			}
			return current, nil
		},
		VerifyCSRF: func(*http.Request, Principal) bool { return csrfOK },
		Authorize:  authorizer,
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if p, ok := PrincipalFromContext(r.Context()); !ok || p != current {
				t.Fatal("Host lost verified quote principal")
			}
			w.WriteHeader(http.StatusAccepted)
		}),
		Audit: func(record AuditRecord) { records = append(records, record) },
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		principal Principal
		revoked   bool
		broken    bool
		session   bool
		csrf      bool
		want      int
	}{
		{"corporate editor", corporateEditor, false, false, true, true, http.StatusAccepted},
		{"legacy editor", legacyEditor, false, false, true, true, http.StatusAccepted},
		{"viewer denied without quote grant", viewer, false, false, true, true, http.StatusForbidden},
		{"revoked mapping", corporateEditor, true, false, true, true, http.StatusForbidden},
		{"mapping unavailable", corporateEditor, false, true, true, true, http.StatusServiceUnavailable},
		{"session invalid", corporateEditor, false, false, false, true, http.StatusUnauthorized},
		{"origin invalid", corporateEditor, false, false, true, false, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current, revoke, unavailable, sessionOK, csrfOK =
				tc.principal, tc.revoked, tc.broken, tc.session, tc.csrf
			beforeCalls, beforeRecords := calls, len(records)
			r := httptest.NewRequest(http.MethodPost, "/reference/quote", nil)
			r.Header.Set("X-Action-ID", "order_entry.submit_order")
			r.Header.Set("X-Principal-ID", "employee-1")
			w := httptest.NewRecorder()
			host.ServeHTTP(w, r)
			if w.Code != tc.want || len(records) != beforeRecords+1 {
				t.Fatalf("quote authorization: status=%d want=%d audits=%d", w.Code, tc.want, len(records)-beforeRecords)
			}
			if (w.Code == http.StatusAccepted) != (calls == beforeCalls+1) {
				t.Fatalf("quote handler called outside its own grant: calls=%d", calls-beforeCalls)
			}
			if records[beforeRecords].Action != quoteAction || records[beforeRecords].Status != tc.want {
				t.Fatalf("wrong quote audit: %+v", records[beforeRecords])
			}
		})
	}
}
