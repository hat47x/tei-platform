package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
	web "tei/reference/web-composition/go-host"
)

type changedQuoteAuthorizationAdapter struct{ adapter legacyauthorization.Adapter }

func (a changedQuoteAuthorizationAdapter) ResolveAdapter(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	if binding.Identity() != "order_entry/security.authorize" {
		return nil, errors.New("unexpected authorization binding")
	}
	return a.adapter, nil
}

func changedQuoteAuthorizer(t *testing.T, world ir.World) (func(context.Context, web.Principal, string) (bool, error), web.Principal, web.Principal) {
	t.Helper()
	users, err := legacyauth.LoadFixtureStore("../../reference/legacy-auth/users.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../reference/legacy-auth/authorization.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Schema string            `json:"schema"`
		Rules  []json.RawMessage `json:"rules"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	rule, err := os.ReadFile("../../reference/candidates/order-quote/authorization-rule.json")
	if err != nil {
		t.Fatal(err)
	}
	doc.Rules = append(doc.Rules, json.RawMessage(rule))
	raw, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(rulePath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	rules, err := legacyauthorization.LoadFixtureRule(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	editor := web.Principal{ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1", Subject: "corporate-editor", PrincipalID: "employee-1"}
	viewer := web.Principal{ProviderID: "corporate", Issuer: editor.Issuer, Subject: "corporate-viewer", PrincipalID: "employee-2"}
	links := map[web.ExternalSubject]web.SubjectLink{
		{ProviderID: editor.ProviderID, Issuer: editor.Issuer, Subject: editor.Subject}: {PrincipalID: editor.PrincipalID, SubjectID: "subject-order-editor"},
		{ProviderID: viewer.ProviderID, Issuer: viewer.Issuer, Subject: viewer.Subject}: {PrincipalID: viewer.PrincipalID, SubjectID: "subject-order-viewer"},
	}
	authorize, err := web.NewCanonicalActionAuthorizer(driver.NewResolver(world), "order_entry", "order_entry.preview_order_quote", func(_ context.Context, key web.ExternalSubject) (web.SubjectLink, bool, error) {
		link, ok := links[key]
		return link, ok, nil
	}, changedQuoteAuthorizationAdapter{adapter: legacyauthorization.Adapter{Store: users, Rule: rules}})
	if err != nil {
		t.Fatal(err)
	}
	return authorize, editor, viewer
}
