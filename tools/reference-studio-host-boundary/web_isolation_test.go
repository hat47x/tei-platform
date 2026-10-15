package hostboundary

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	webcomposition "tei/reference/web-composition/go-host"
)

// These are synthetic sessions and documents for a bounded test of the
// existing #1043 HTTP guard. They are NOT a production authentication scheme
// or a replacement for Studio's resource authorization.
func TestReferenceHostTwoConnectionsAndRevokedSession(t *testing.T) {
	const issuer = "https://issuer.example.invalid/tei-studio"
	a := webcomposition.Principal{ProviderID: "test", Issuer: issuer, Subject: "subject-a", PrincipalID: "principal-a"}
	b := webcomposition.Principal{ProviderID: "test", Issuer: issuer, Subject: "subject-b", PrincipalID: "principal-b"}

	var mu sync.RWMutex
	sessions := map[string]webcomposition.Principal{"synthetic-a": a, "synthetic-b": b}
	privateResults := map[string]string{
		a.PrincipalID: "workspace-a:private-result-a",
		b.PrincipalID: "workspace-b:private-result-b",
	}
	var invoked int

	host, err := webcomposition.NewHost(webcomposition.Policy{
		Path: "/studio/reference-read", Method: http.MethodPost,
		ActionID:  "studio.reference-read",
		Providers: map[string]string{"test": issuer},
	}, webcomposition.Dependencies{
		VerifySession: func(r *http.Request) (webcomposition.Principal, error) {
			cookie, err := r.Cookie("synthetic-session")
			if err != nil {
				return webcomposition.Principal{}, errors.New("no fixture session")
			}
			mu.RLock()
			p, ok := sessions[cookie.Value]
			mu.RUnlock()
			if !ok {
				return webcomposition.Principal{}, errors.New("revoked fixture session")
			}
			return p, nil
		},
		VerifyCSRF: func(r *http.Request, _ webcomposition.Principal) bool {
			return r.Header.Get("X-Synthetic-CSRF") == "valid"
		},
		Authorize: func(_ context.Context, p webcomposition.Principal, action string) (bool, error) {
			// This test only admits the fixed read-only Action; real resource
			// scope must be enforced by the actual Host/Operation integration.
			return action == "studio.reference-read" && privateResults[p.PrincipalID] != "", nil
		},
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := webcomposition.PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, "missing verified principal", http.StatusInternalServerError)
				return
			}
			mu.Lock()
			invoked++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, privateResults[p.PrincipalID])
		}),
		Audit: func(webcomposition.AuditRecord) {},
	})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(host)
	defer server.Close()

	// Separate clients use separate TCP connections. Neither the request
	// header nor a previously verified request may overwrite Host identity.
	send := func(client *http.Client, session, forgedPrincipal, csrf string) (int, string, error) {
		request, err := http.NewRequest(http.MethodPost, server.URL+"/studio/reference-read", nil)
		if err != nil {
			return 0, "", err
		}
		request.AddCookie(&http.Cookie{Name: "synthetic-session", Value: session})
		request.Header.Set("X-Synthetic-CSRF", csrf)
		request.Header.Set("X-Principal-ID", forgedPrincipal)
		request.Header.Set("X-Workspace", "untrusted-workspace-b")
		response, err := client.Do(request)
		if err != nil {
			return 0, "", err
		}
		defer response.Body.Close()
		payload, err := io.ReadAll(io.LimitReader(response.Body, 4096))
		return response.StatusCode, string(payload), err
	}

	clientA := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	clientB := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	defer clientA.CloseIdleConnections()
	defer clientB.CloseIdleConnections()

	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for i := 0; i < 12; i++ {
		for _, c := range []struct {
			client                    *http.Client
			token, spoof, want, other string
		}{
			{clientA, "synthetic-a", b.PrincipalID, privateResults[a.PrincipalID], privateResults[b.PrincipalID]},
			{clientB, "synthetic-b", a.PrincipalID, privateResults[b.PrincipalID], privateResults[a.PrincipalID]},
		} {
			wg.Add(1)
			go func(c struct {
				client                    *http.Client
				token, spoof, want, other string
			}) {
				defer wg.Done()
				status, body, err := send(c.client, c.token, c.spoof, "valid")
				if err != nil || status != http.StatusOK || body != c.want || strings.Contains(body, c.other) {
					failures <- fmt.Errorf("two-connection isolation: status=%d body=%q expected=%q err=%v", status, body, c.want, err)
				}
			}(c)
		}
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if t.Failed() {
		return
	}

	// A denied CSRF request must not re-use identity/approval from the
	// previous successful request.
	mu.RLock()
	before := invoked
	mu.RUnlock()
	if status, _, err := send(clientA, "synthetic-a", b.PrincipalID, "invalid"); err != nil || status != http.StatusForbidden {
		t.Fatalf("CSRF failure: status=%d err=%v", status, err)
	}
	mu.RLock()
	afterCSRF := invoked
	mu.RUnlock()
	if afterCSRF != before {
		t.Fatalf("denied request invoked Action: before=%d after=%d", before, afterCSRF)
	}

	// A reconnect represented by a new HTTP client still needs a current
	// session. No cached verified Principal may survive revocation.
	mu.Lock()
	delete(sessions, "synthetic-a")
	mu.Unlock()
	reconnectedA := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	defer reconnectedA.CloseIdleConnections()
	status, body, err := send(reconnectedA, "synthetic-a", b.PrincipalID, "valid")
	if err != nil || status != http.StatusUnauthorized ||
		strings.Contains(body, privateResults[a.PrincipalID]) ||
		strings.Contains(body, privateResults[b.PrincipalID]) {
		t.Fatalf("reconnected revoked session: status=%d body=%q err=%v", status, body, err)
	}
	status, body, err = send(clientB, "synthetic-b", a.PrincipalID, "valid")
	if err != nil || status != http.StatusOK || body != privateResults[b.PrincipalID] {
		t.Fatalf("other session after revocation: status=%d body=%q err=%v", status, body, err)
	}
}

// Characterization only: #1043's NewHost protects the Action and Principal
// handoff, but does not interpret or authorize a caller-supplied Workspace.
// A future Studio Host must never treat that header as a resource grant.
func TestReferenceHostLeavesWorkspaceScopeToOperation(t *testing.T) {
	const issuer = "fixture-issuer"
	seenWorkspace := ""
	host, err := webcomposition.NewHost(webcomposition.Policy{
		Path: "/studio/reference-read", Method: http.MethodPost,
		ActionID:  "studio.reference-read",
		Providers: map[string]string{"fixture": issuer},
	}, webcomposition.Dependencies{
		VerifySession: func(*http.Request) (webcomposition.Principal, error) {
			return webcomposition.Principal{ProviderID: "fixture", Issuer: issuer, Subject: "a", PrincipalID: "p"}, nil
		},
		VerifyCSRF: func(*http.Request, webcomposition.Principal) bool { return true },
		Authorize:  func(context.Context, webcomposition.Principal, string) (bool, error) { return true, nil },
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenWorkspace = r.Header.Get("X-Workspace")
			w.WriteHeader(http.StatusAccepted)
		}),
		Audit: func(webcomposition.AuditRecord) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://host.test/studio/reference-read", nil)
	request.Header.Set("X-Workspace", "../other-workspace")
	recorder := httptest.NewRecorder()
	host.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted || seenWorkspace != "../other-workspace" {
		t.Fatalf("unexpected guarded dispatch: status=%d workspace=%q", recorder.Code, seenWorkspace)
	}
	// A passing test DOES NOT prove Workspace scope protection. It verifies
	// that no such promise should be inferred from NewHost's Action guard.
}
