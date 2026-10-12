package webcomposition

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The independent quote is a read-only semantic Capability, not a second
// implementation of order_entry.submit_order. No process execution occurs.
func TestStandaloneQuotePatternHasSeparateCanonicalBindingAndAuthorization(t *testing.T) {
	resolver, adapters := referenceAuthorizationDependencies(t)
	const quoteAction = "order_entry.preview_order_quote"
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
		Path: "/reference/quote",
		Method: http.MethodPost,
		ActionID: quoteAction,
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
		name string
		principal Principal
		revoked bool
		broken bool
		session bool
		csrf bool
		want int
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
