package interfaceadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	webcomposition "tei/reference/web-composition/go-host"
)

// HTTPIdentity is a fixed fixture identity. Tokens and roles are host-owned and
// are never read from the request body.
type HTTPIdentity struct {
	Subject string
	Role    string
}

// HTTPOptions composes the protected HTTP route.
type HTTPOptions struct {
	Core       Core
	Path       string
	ActionID   string
	ProviderID string
	Issuer     string
	// Tokens maps a fixture bearer token to an identity. A real deployment
	// replaces this with a session verifier; this harness does not.
	Tokens map[string]HTTPIdentity
	Audit  func(webcomposition.AuditRecord)
}

// HTTPServer is the Web transport adapter. It uses the existing protected host
// for session, CSRF, route and Action authorization, then calls Core.
type HTTPServer struct {
	handler http.Handler
	roles   map[string]string
	invoked *int64
	mu      *sync.Mutex
}

// NewHTTPServer builds the protected route. The host rejects a request before
// Core is reached when session, CSRF or authorization fails.
func NewHTTPServer(opts HTTPOptions) (*HTTPServer, error) {
	if opts.Core.Revision == "" || opts.Path == "" || opts.ActionID == "" {
		return nil, fmt.Errorf("interface adapter: core revision, path and action are required")
	}
	roles := map[string]string{}
	for token, identity := range opts.Tokens {
		if token == "" || identity.Subject == "" {
			return nil, fmt.Errorf("interface adapter: fixture tokens need a token and subject")
		}
		roles[identity.Subject] = identity.Role
	}
	server := &HTTPServer{roles: roles, invoked: new(int64), mu: &sync.Mutex{}}
	audit := opts.Audit
	if audit == nil {
		audit = func(webcomposition.AuditRecord) {}
	}
	handler, err := webcomposition.NewHost(webcomposition.Policy{
		Path:     opts.Path,
		Method:   http.MethodPost,
		ActionID: opts.ActionID,
		Providers: map[string]string{
			opts.ProviderID: opts.Issuer,
		},
	}, webcomposition.Dependencies{
		VerifySession: func(r *http.Request) (webcomposition.Principal, error) {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			identity, ok := opts.Tokens[token]
			if !ok {
				return webcomposition.Principal{}, fmt.Errorf("unknown fixture session")
			}
			return webcomposition.Principal{
				ProviderID:  opts.ProviderID,
				Issuer:      opts.Issuer,
				Subject:     identity.Subject,
				PrincipalID: identity.Subject,
			}, nil
		},
		VerifyCSRF: func(r *http.Request, _ webcomposition.Principal) bool {
			return r.Header.Get("X-CSRF-Fixture") == "matching"
		},
		Authorize: func(_ context.Context, principal webcomposition.Principal, action string) (bool, error) {
			if action != opts.ActionID {
				return false, nil
			}
			return roles[principal.PrincipalID] == "editor", nil
		},
		Invoke: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			server.mu.Lock()
			*server.invoked++
			server.mu.Unlock()
			principal, ok := webcomposition.PrincipalFromContext(r.Context())
			if !ok {
				writeHTTPResult(w, http.StatusUnauthorized, Result{Failure: &Failure{Code: "UNAUTHENTICATED", Message: "no verified principal"}})
				return
			}
			var request Request
			decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				writeHTTPResult(w, http.StatusBadRequest, Result{Failure: &Failure{Code: "INVALID_REQUEST", Message: err.Error()}})
				return
			}
			result := opts.Core.Invoke(r.Context(), Principal{
				Subject: principal.PrincipalID,
				Role:    roles[principal.PrincipalID],
			}, request)
			writeHTTPResult(w, statusFor(result), result)
		}),
		Audit: audit,
	})
	if err != nil {
		return nil, err
	}
	server.handler = handler
	return server, nil
}

// Handler exposes the protected route for httptest or a real server.
func (s *HTTPServer) Handler() http.Handler { return s.handler }

// Invocations is the number of requests that reached the Invoke handler. Host
// rejections happen before this counter moves.
func (s *HTTPServer) Invocations() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int(*s.invoked)
}

func statusFor(result Result) int {
	if result.Failure == nil {
		return http.StatusOK
	}
	switch result.Failure.Code {
	case "UNAUTHENTICATED":
		return http.StatusUnauthorized
	case "FORBIDDEN":
		return http.StatusForbidden
	case "STALE_REVISION":
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func writeHTTPResult(w http.ResponseWriter, status int, result Result) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

// HTTPAdapter is the client side of the Web transport. It carries a fixture
// session and CSRF header; the server derives the principal from them.
type HTTPAdapter struct {
	URL    string
	Token  string
	CSRF   string
	Client *http.Client
}

// Execute sends one typed request over HTTP.
func (a HTTPAdapter) Execute(ctx context.Context, request Request) (Result, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return Result{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, a.URL, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+a.Token)
	httpRequest.Header.Set("X-CSRF-Fixture", a.CSRF)
	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()
	var result Result
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&result); err != nil {
		return Result{Failure: failureForStatus(response.StatusCode)}, nil
	}
	if result.Failure == nil && response.StatusCode >= 400 {
		result.Failure = failureForStatus(response.StatusCode)
	}
	return result, nil
}

func failureForStatus(status int) *Failure {
	switch status {
	case http.StatusUnauthorized:
		return &Failure{Code: "UNAUTHENTICATED", Message: "the server rejected the session"}
	case http.StatusForbidden:
		return &Failure{Code: "FORBIDDEN", Message: "the server rejected the request"}
	case http.StatusConflict:
		return &Failure{Code: "STALE_REVISION", Message: "the server rejected the revision"}
	default:
		return &Failure{Code: "TRANSPORT_ERROR", Message: fmt.Sprintf("status %d without a typed result", status)}
	}
}
