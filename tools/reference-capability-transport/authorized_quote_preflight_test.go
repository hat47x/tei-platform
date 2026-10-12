package capabilitytransport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/composition"
	"tei/internal/core"
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

type quotePreflightBindingAdapters struct {
	adapter legacyauthorization.Adapter
}

func (a quotePreflightBindingAdapters) ResolveAdapter(_ context.Context, b realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	if b.Identity() != "order_entry/security.authorize" {
		return nil, errors.New("unknown canonical authorization binding")
	}
	return a.adapter, nil
}

type recordedQuoteLauncher struct {
	calls int
	last  pruntime.Request
}

func (l *recordedQuoteLauncher) Invoke(ctx context.Context, command pruntime.ProcessCommand, request pruntime.Request) (pruntime.Response, error) {
	l.calls++
	l.last = request
	return (pruntime.StdioProcessLauncher{}).Invoke(ctx, command, request)
}

// The accepted canonical Action authorizes an intent to submit an order, not
// an independent quote endpoint. This *only* probes a read-only quote coeffect
// inside that Action's protected request. It never submits an order or claims
// standalone canonical authorization for reference.order.quote-lines/v1.
func TestCanonicalSubmitIntentMayInvokeSelectedReadOnlyQuoteCoeffect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
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
	corporate := webcomposition.Principal{
		ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1",
		Subject: "oidc-editor", PrincipalID: "employee-1",
	}
	legacy := webcomposition.Principal{
		ProviderID: "legacy", Issuer: "legacy:order_entry",
		Subject: "legacy-editor", PrincipalID: "employee-1",
	}
	viewer := webcomposition.Principal{
		ProviderID: "corporate", Issuer: corporate.Issuer,
		Subject: "oidc-viewer", PrincipalID: "employee-2",
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
	linkRevoked := false
	authorize, err := webcomposition.NewCanonicalActionAuthorizer(
		driver.NewResolver(loaded.World), "order_entry", "order_entry.submit_order",
		func(_ context.Context, source webcomposition.ExternalSubject) (webcomposition.SubjectLink, bool, error) {
			if linkRevoked {
				return webcomposition.SubjectLink{}, false, nil
			}
			value, ok := links[source]
			return value, ok, nil
		},
		quotePreflightBindingAdapters{adapter: legacyauthorization.Adapter{Store: users, Rule: rules}},
	)
	if err != nil {
		t.Fatal(err)
	}

	work := t.TempDir()
	source, err := os.ReadFile("../../plugins/reference-order-quote/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "main.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "go.mod"), []byte("module example.invalid/quote-preflight\n\ngo 1.23\n"), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "order-quote")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = work
	build.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build independent quote provider: %v\n%s", err, output)
	}
	executable, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	digest := digestBytes(executable)
	identity := GenericProviderIdentity{
		PluginID: "tei.reference.order_quote", PluginVersion: "1.0.0",
		ImplementationID: "order-quote.v1", ArtifactDigest: digest,
	}
	const parentAction = "order_entry.submit_order"
	const quoteContract = "reference.order.quote-lines/v1"
	dispatch := GenericDispatch{
		Schema: dispatchSchema, ContractID: quoteContract,
		ContractFingerprint: digestBytes([]byte("reference.order.quote-lines.contract/v1")),
		Target: "reference/order_entry", Provider: identity,
		RequestShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.request/v1")),
		ResultShapeFingerprint: digestBytes([]byte("reference.order.quote-lines.result/v1")),
		PayloadCodec: "application/json",
	}
	plan, err := buildTransportDispatchPlan(genericDispatchIdentity(dispatch))
	if err != nil {
		t.Fatal(err)
	}
	carrier := LegacyCarrier{
		Plugin: plugin.Identity{ID: core.MustCanonicalID(identity.PluginID), Version: identity.PluginVersion},
		CapabilityKind: plugin.CapabilityContractQueryExecutor,
		CapabilityID: core.MustCanonicalID("transport.order_quote_carrier"),
		DescriptorFingerprint: digestBytes([]byte("reference.order-quote.transport-carrier/v1")),
		Operation: "generic-dispatch-v0",
	}
	contract := composition.Contract{
		ID: quoteContract, Version: "1.0.0",
		Shape: dispatch.RequestShapeFingerprint + "/" + dispatch.ResultShapeFingerprint,
	}
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "quote-coeffect", Scope: dispatch.Target, Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("quote-coeffect", dispatch.Target)
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{
		ContractID: quoteContract, ContractVersion: contract.Version,
		ProviderID: selection.ProviderID, ArtifactDigest: digest, Generation: 1,
	}
	currentAuthority := true
	guardCalls := 0
	coordinator, err := composition.NewSelectedCoordinator(
		registry, selection, binding, composition.NewScope(),
		func(actual composition.Binding) error {
			guardCalls++
			if !currentAuthority || actual != binding {
				return errors.New("current Host execution approval unavailable")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	launcher := &recordedQuoteLauncher{}
	seq := 0
	selected, err := webcomposition.NewHostAuthorizedSelectedCapabilityInvoker(
		coordinator, parentAction, contract.ID, contract.Version,
		[]webcomposition.SelectedActionProvider{{
			ProviderID: selection.ProviderID, ArtifactDigest: digest,
			Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Only the typed quote input crosses the process boundary.
				// Neither session headers nor the Host-local Action approval
				// are serialized as execution Authority.
				if _, ok := webcomposition.PrincipalFromContext(r.Context()); !ok {
					http.Error(w, "verified caller required", http.StatusForbidden)
					return
				}
				if r.Header.Get("Content-Type") != "application/json" {
					http.Error(w, "JSON quote required", http.StatusUnsupportedMediaType)
					return
				}
				data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
				if err != nil {
					http.Error(w, "invalid quote request", http.StatusBadRequest)
					return
				}
				var input struct {
					Currency string `json:"currency"`
					Lines []struct {
						SKU string `json:"sku"`
						Quantity int64 `json:"quantity"`
						UnitPriceMinor int64 `json:"unitPriceMinor"`
					} `json:"lines"`
				}
				if err := decodeJSONExact(data, &input); err != nil ||
					input.Currency != "JPY" || len(input.Lines) == 0 || len(input.Lines) > 50 {
					http.Error(w, "invalid quote shape", http.StatusBadRequest)
					return
				}
				// Executable checksum is a local Reference boundary. Runtime
				// signature verification and provenance remain Host-owned.
				actual, err := os.ReadFile(binary)
				if err != nil || digestBytes(actual) != binding.ArtifactDigest {
					http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
					return
				}
				toSend := dispatch
				toSend.Payload = data
				seq++
				execution, err := invokePlannedOverProcessTransport(
					r.Context(), launcher, pruntime.ProcessCommand{Executable: binary},
					plan, carrier, "quote-preflight-"+string(rune('0'+seq)), toSend,
				)
				if err != nil {
					http.Error(w, "quote calculation unavailable", http.StatusBadGateway)
					return
				}
				var value struct {
					Currency *string `json:"currency"`
					LineCount *int `json:"lineCount"`
					SubtotalMinor *int64 `json:"subtotalMinor"`
				}
				if err := decodeJSONExact(execution.ResultPayload, &value); err != nil ||
					value.Currency == nil || *value.Currency != "JPY" ||
					value.LineCount == nil || *value.LineCount != len(input.Lines) ||
					value.SubtotalMinor == nil || *value.SubtotalMinor < 0 ||
					*value.SubtotalMinor > 1_000_000_000_000 {
					http.Error(w, "quote result violates consumer contract", http.StatusBadGateway)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(struct {
					Preview bool `json:"preview"`
					Submitted bool `json:"submitted"`
					SubtotalMinor int64 `json:"subtotalMinor"`
				}{Preview: true, Submitted: false, SubtotalMinor: *value.SubtotalMinor})
			}),
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	current := corporate
	sessionOK, csrfOK := true, true
	var audits []webcomposition.AuditRecord
	host, err := webcomposition.NewHost(webcomposition.Policy{
		Path: "/reference/submit-intent/quote-preview",
		Method: http.MethodPost, ActionID: parentAction,
		Providers: map[string]string{"corporate": corporate.Issuer, "legacy": legacy.Issuer},
	}, webcomposition.Dependencies{
		VerifySession: func(*http.Request) (webcomposition.Principal, error) {
			if !sessionOK {
				return webcomposition.Principal{}, errors.New("session expired")
			}
			return current, nil
		},
		VerifyCSRF: func(*http.Request, webcomposition.Principal) bool { return csrfOK },
		Authorize: authorize,
		Invoke: selected,
		Audit: func(record webcomposition.AuditRecord) { audits = append(audits, record) },
	})
	if err != nil {
		t.Fatal(err)
	}
	requestBody := `{"currency":"JPY","lines":[{"sku":"PEN-01","quantity":3,"unitPriceMinor":250},{"sku":"BOOK-02","quantity":2,"unitPriceMinor":400}]}`
	send := func(handler http.Handler, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/reference/submit-intent/quote-preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer fake-user-secret")
		req.Header.Set("X-Principal-ID", "administrator")
		req.Header.Set("X-Provider-ID", "forged")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	for _, p := range []webcomposition.Principal{corporate, legacy} {
		current = p
		before := launcher.calls
		reply := send(host, requestBody)
		if reply.Code != http.StatusOK || launcher.calls != before+1 ||
			!strings.Contains(reply.Body.String(), `"subtotalMinor":1550`) ||
			!strings.Contains(reply.Body.String(), `"submitted":false`) {
			t.Fatalf("protected read-only preflight: HTTP=%d calls=%d body=%s",
				reply.Code, launcher.calls-before, reply.Body.String())
		}
		raw, err := json.Marshal(launcher.last)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"fake-user-secret", "administrator", "oidc-editor",
			"legacy-editor", "subject-order-editor", "employee-1", "forged"} {
			if strings.Contains(string(raw), secret) {
				t.Fatalf("Host identity or user-supplied header crossed process boundary: %s", secret)
			}
		}
	}
	denied := func(label string, handler http.Handler, body string, want int) {
		t.Helper()
		before := launcher.calls
		reply := send(handler, body)
		if reply.Code != want || launcher.calls != before {
			t.Fatalf("%s: HTTP=%d expected=%d calls=%d body=%s",
				label, reply.Code, want, launcher.calls-before, reply.Body.String())
		}
	}
	denied("bare capability cannot escape parent Action", selected, requestBody, http.StatusForbidden)
	current = viewer
	denied("viewer denied by canonical business rules", host, requestBody, http.StatusForbidden)
	current = corporate
	sessionOK = false
	denied("session invalid", host, requestBody, http.StatusUnauthorized)
	sessionOK, csrfOK = true, false
	denied("CSRF invalid", host, requestBody, http.StatusForbidden)
	csrfOK, linkRevoked = true, true
	denied("current subject link revoked", host, requestBody, http.StatusForbidden)
	linkRevoked = false
	denied("untrusted identity payload refused locally", host,
		`{"currency":"JPY","lines":[{"sku":"PEN","quantity":1,"unitPriceMinor":1}],"principalId":"admin"}`, http.StatusBadRequest)
	bad := `{"currency":"JPY","lines":[{"sku":"PEN-01","quantity":-1,"unitPriceMinor":250}]}`
	before := launcher.calls
	reply := send(host, bad)
	if reply.Code != http.StatusBadGateway || launcher.calls != before+1 {
		t.Fatalf("invalid domain request unexpectedly succeeded: HTTP=%d calls=%d", reply.Code, launcher.calls-before)
	}
	currentAuthority = false
	denied("Host resource authority revoked", host, requestBody, http.StatusServiceUnavailable)
	currentAuthority = true
	if _, err := registry.Apply(snapshot.Revision, nil, []string{"quote-coeffect"}); err != nil {
		t.Fatal(err)
	}
	denied("provider withdrawn", host, requestBody, http.StatusServiceUnavailable)
	if len(audits) == 0 || audits[len(audits)-1].Stage != "error" {
		t.Fatalf("selected quote failures must remain Host-audited: %+v", audits)
	}
	if guardCalls < 2 {
		t.Fatal("selected quote did not use the existing current Host guard")
	}
}
