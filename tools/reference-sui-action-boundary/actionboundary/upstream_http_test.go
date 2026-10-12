package actionboundary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fixedDelegation struct {
	value TrustedDelegation
	err   error
	calls int
}

func (d *fixedDelegation) Delegation(context.Context, Intent) (TrustedDelegation, error) {
	d.calls++
	return d.value, d.err
}

func upstreamIntent() Intent {
	return Intent{
		ApplicationID: "sui", ResourceID: "doc-1", ActionID: "sui.move",
		ExpectedRevision: strings.Repeat("a", 64),
		Payload: map[string]any{"cardId": "card-1", "x": 20, "y": 25},
	}
}

func upstreamStore(server *httptest.Server, delegation *fixedDelegation) HTTPSApplicationStore {
	return HTTPSApplicationStore{
		ApplicationID: "sui", ActionID: "sui.move",
		Origin: server.URL, Prefix: "/docs/", Suffix: "/action-commit",
		Client: server.Client(), Subjects: fixedSubject{id: "human-1"},
		Delegations: delegation,
	}
}

func validDelegation() *fixedDelegation {
	return &fixedDelegation{value: TrustedDelegation{
		SubjectID: "human-1", TenantID: "tenant-1", BearerToken: "audience-scoped-test-token",
	}}
}

func TestHTTPSApplicationStoreHostToApplication(t *testing.T) {
	writes := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writes++
		if r.Method != http.MethodPost ||
			r.URL.EscapedPath() != "/docs/doc-1/action-commit" ||
			r.Header.Get("X-TEI-Action") != "commit" ||
			r.Header.Get("Authorization") != "Bearer audience-scoped-test-token" ||
			r.Header.Get("Cookie") != "" || r.Header.Get("X-Actor-Ref") != "" ||
			r.Header.Get("X-Forwarded-User") != "" || r.Header.Get("Origin") != "" {
			t.Errorf("unexpected forwarded request: %s %s headers=%v", r.Method, r.URL, r.Header)
		}
		var input map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Errorf("request decode: %v", err)
		}
		if len(input) != 6 || string(input["protocolVersion"]) != `"1"` ||
			string(input["applicationID"]) != `"sui"` ||
			string(input["resourceID"]) != `"doc-1"` ||
			string(input["actionID"]) != `"sui.move"` {
			t.Errorf("malformed generic Action: %#v", input)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"protocolVersion":"1","revision":"%s"}`, strings.Repeat("b", 64))))
	}))
	defer server.Close()
	actor := validDelegation()
	store := upstreamStore(server, actor)
	host := Host{
		Authorizer: permit(true),
		Bindings: []Binding{{ApplicationID: "sui", ActionID: "sui.move", Store: store}},
	}
	result, err := host.Commit(context.Background(), upstreamIntent())
	if err != nil || result != strings.Repeat("b", 64) {
		t.Fatalf("commit: result=%q error=%v", result, err)
	}
	if writes != 1 || actor.calls != 1 {
		t.Fatalf("write count=%d delegation count=%d", writes, actor.calls)
	}
}

func TestHTTPSApplicationStoreRejectsUnverifiedDelegationBeforeNetwork(t *testing.T) {
	writes := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writes++
		_, _ = w.Write([]byte(`{"protocolVersion":"1","revision":"r2"}`))
	}))
	defer server.Close()
	for _, tc := range []struct {
		name   string
		change func(*HTTPSApplicationStore, *fixedDelegation, *Intent)
	}{
		{"missing subject", func(s *HTTPSApplicationStore, _ *fixedDelegation, _ *Intent) { s.Subjects = nil }},
		{"missing delegation", func(s *HTTPSApplicationStore, _ *fixedDelegation, _ *Intent) { s.Delegations = nil }},
		{"mismatched subject", func(_ *HTTPSApplicationStore, d *fixedDelegation, _ *Intent) { d.value.SubjectID = "other-user" }},
		{"missing tenant", func(_ *HTTPSApplicationStore, d *fixedDelegation, _ *Intent) { d.value.TenantID = "" }},
		{"missing token", func(_ *HTTPSApplicationStore, d *fixedDelegation, _ *Intent) { d.value.BearerToken = "" }},
		{"injected token", func(_ *HTTPSApplicationStore, d *fixedDelegation, _ *Intent) { d.value.BearerToken = "ok\nX-Actor-Ref: admin" }},
		{"missing client", func(s *HTTPSApplicationStore, _ *fixedDelegation, _ *Intent) { s.Client = nil }},
		{"unencrypted origin", func(s *HTTPSApplicationStore, _ *fixedDelegation, _ *Intent) { s.Origin = "http://example.com" }},
		{"userinfo origin", func(s *HTTPSApplicationStore, _ *fixedDelegation, _ *Intent) { s.Origin = "https://user@host.test" }},
		{"invalid prefix", func(s *HTTPSApplicationStore, _ *fixedDelegation, _ *Intent) { s.Prefix = "//evil.test/" }},
		{"path traversal", func(s *HTTPSApplicationStore, _ *fixedDelegation, _ *Intent) { s.Suffix = "/../commit" }},
		{"foreign application", func(_ *HTTPSApplicationStore, _ *fixedDelegation, i *Intent) { i.ApplicationID = "inventory" }},
		{"foreign action", func(_ *HTTPSApplicationStore, _ *fixedDelegation, i *Intent) { i.ActionID = "inventory.adjust" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := validDelegation()
			s := upstreamStore(server, d)
			input := upstreamIntent()
			tc.change(&s, d, &input)
			if _, err := s.Commit(context.Background(), input); !errors.Is(err, ErrDenied) && !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected fail-closed result, got %v", err)
			}
			if writes != 0 {
				t.Fatalf("unverified request reached upstream: writes=%d", writes)
			}
		})
	}
}

func TestHTTPSApplicationStoreUpstreamOutcomesAreNotRetried(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"conflict", 409, `{"protocolVersion":"1","error":"revision_conflict"}`, ErrConflict},
		{"denied", 403, `{"protocolVersion":"1","error":"action_denied"}`, ErrDenied},
		{"invalid", 400, `{"protocolVersion":"1","error":"invalid_payload"}`, ErrInvalid},
		{"500 after possible commit", 500, `{"protocolVersion":"1","error":"execution_failed"}`, ErrRemoteOutcomeUnknown},
		{"redirect", 302, `{"protocolVersion":"1","error":"action_denied"}`, ErrRemoteOutcomeUnknown},
		{"unknown status", 404, `{"protocolVersion":"1","error":"action_denied"}`, ErrRemoteOutcomeUnknown},
		{"duplicate result", 200, `{"protocolVersion":"1","revision":"r2","revision":"r3"}`, ErrRemoteOutcomeUnknown},
		{"foreign response fields", 200, `{"protocolVersion":"1","revision":"r2","admin":true}`, ErrRemoteOutcomeUnknown},
		{"unsupported version", 200, `{"protocolVersion":"2","revision":"r2"}`, ErrRemoteOutcomeUnknown},
		{"missing revision", 200, `{"protocolVersion":"1","error":"execution_failed"}`, ErrRemoteOutcomeUnknown},
		{"oversized response", 200, `{"protocolVersion":"1","revision":"` + strings.Repeat("z", 9000) + `"}`, ErrRemoteOutcomeUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			count := 0
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer s.Close()
			store := upstreamStore(s, validDelegation())
			_, err := store.Commit(context.Background(), upstreamIntent())
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, received %v", tc.want, err)
			}
			if count != 1 {
				t.Fatalf("unexpected resend: %d", count)
			}
		})
	}
}

func TestHTTPSApplicationStoreFailClosedOnDisconnectedApplication(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	store := upstreamStore(s, validDelegation())
	s.Close()
	_, err := store.Commit(context.Background(), upstreamIntent())
	if !errors.Is(err, ErrRemoteOutcomeUnknown) {
		t.Fatalf("disconnected application outcome must be unknown: %v", err)
	}
}

func TestHTTPSApplicationStoreUsesOneContractForIndependentInventoryApp(t *testing.T) {
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.EscapedPath() != "/items/sku-42/adjust" {
			t.Errorf("unexpected resource path: %q", r.URL.EscapedPath())
		}
		var action struct {
			ProtocolVersion string         `json:"protocolVersion"`
			ApplicationID   string         `json:"applicationID"`
			ResourceID      string         `json:"resourceID"`
			ActionID        string         `json:"actionID"`
			Payload         map[string]any `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&action); err != nil ||
			action.ProtocolVersion != "1" || action.ApplicationID != "inventory" ||
			action.ResourceID != "sku-42" || action.ActionID != "inventory.adjust" ||
			action.Payload["sku"] != "sku-42" || action.Payload["delta"] != float64(3) {
			t.Errorf("non-generic inventory envelope: %+v, err=%v", action, err)
		}
		_, _ = w.Write([]byte(`{"protocolVersion":"1","revision":"stock-v2"}`))
	}))
	defer server.Close()
	store := HTTPSApplicationStore{
		ApplicationID: "inventory", ActionID: "inventory.adjust",
		Origin: server.URL, Prefix: "/items/", Suffix: "/adjust",
		Client: server.Client(), Subjects: fixedSubject{id: "human-1"},
		Delegations: validDelegation(),
	}
	intent := Intent{
		ApplicationID: "inventory", ActionID: "inventory.adjust",
		ResourceID: "sku-42", ExpectedRevision: "stock-v1",
		Payload: map[string]any{"sku": "sku-42", "delta": 3},
	}
	rev, err := store.Commit(context.Background(), intent)
	if err != nil || rev != "stock-v2" || calls != 1 {
		t.Fatalf("inventory commit: revision=%s calls=%d error=%v", rev, calls, err)
	}
}
