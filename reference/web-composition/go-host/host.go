// Package webcomposition is an isolated, synthetic HTTP security-boundary reference.
// It does not provide production authentication, session storage, or CSRF protection.
package webcomposition

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// Principal is returned by a trusted, explicitly supplied session verifier.
// Subject and issuer are never inferred from a display name or email address.
type Principal struct {
	ProviderID  string
	Issuer      string
	Subject     string
	PrincipalID string
}

// Policy is host-owned: callers must not derive its contents from a browser request.
type Policy struct {
	Path     string
	Method   string
	ActionID string
	// Providers is the exact provider ID -> issuer mapping allowed for the route.
	Providers map[string]string
}

type AuditRecord struct {
	Route      string
	Action     string
	Stage      string // completed, rejected, error
	Status     int
	ProviderID string
	// Do not include credentials, tokens, cookies, or request bodies.
}

type Dependencies struct {
	// VerifySession must check authenticity and current validity, not parse an untrusted ID.
	VerifySession func(*http.Request) (Principal, error)
	VerifyCSRF    func(*http.Request, Principal) bool
	Authorize     func(context.Context, Principal, string) (bool, error)
	Invoke        http.Handler
	Audit         func(AuditRecord)
}

// NewHost creates a single protected Action endpoint. Security steps belong to the
// host, not to interchangeable middleware that may declare its own guards.
func NewHost(p Policy, d Dependencies) (http.Handler, error) {
	if p.Path == "" || p.Path[0] != '/' || strings.ContainsAny(p.Path, "?#%\\\\") || strings.Contains(p.Path, "//") || strings.Contains(p.Path, "/./") || strings.Contains(p.Path, "/../") || strings.HasSuffix(p.Path, "/.") || strings.HasSuffix(p.Path, "/..") || p.Method != http.MethodPost || p.ActionID == "" || len(p.Providers) == 0 {
		return nil, errors.New("protected route needs exact POST path, action and providers")
	}
	for id, issuer := range p.Providers {
		if id == "" || issuer == "" {
			return nil, errors.New("provider ID and issuer must be explicit")
		}
	}
	// Freeze provider selection at compilation time; callers must not be able to
	// change the accepted issuer through a shared map after NewHost returns.
	selected := make(map[string]string, len(p.Providers))
	for id, issuer := range p.Providers {
		selected[id] = issuer
	}
	p.Providers = selected
	if d.VerifySession == nil || d.VerifyCSRF == nil || d.Authorize == nil || d.Invoke == nil || d.Audit == nil {
		return nil, errors.New("host-owned protections and audit must all be wired")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tw := &statusWriter{ResponseWriter: w}
		stage := "rejected"
		record := AuditRecord{Route: p.Method + " " + p.Path, Action: p.ActionID}
		defer func() {
			if recover() != nil {
				// A response already started cannot safely be rewritten as 500.
				stage = "error"
				if tw.status == 0 {
					http.Error(tw, "internal error", http.StatusInternalServerError)
				}
			}
			if tw.status == 0 {
				tw.status = http.StatusOK
			}
			record.Status = tw.status
			if stage == "error" || tw.status >= 500 {
				stage = "error"
			} else if tw.status >= 400 {
				stage = "rejected"
			}
			record.Stage = stage
			d.Audit(record)
		}()

		// Compare the encoded path, avoiding decoded alias paths for this reference.
		if r.URL.EscapedPath() != p.Path {
			http.NotFound(tw, r)
			return
		}
		if r.Method != p.Method {
			tw.Header().Set("Allow", p.Method)
			http.Error(tw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		subject, err := d.VerifySession(r)
		if err != nil || subject.ProviderID == "" || subject.Issuer == "" || subject.Subject == "" || subject.PrincipalID == "" {
			http.Error(tw, "authentication required", http.StatusUnauthorized)
			return
		}
		issuer, ok := p.Providers[subject.ProviderID]
		if !ok || issuer != subject.Issuer {
			http.Error(tw, "authentication required", http.StatusUnauthorized)
			return
		}
		record.ProviderID = subject.ProviderID
		if !d.VerifyCSRF(r, subject) {
			http.Error(tw, "request origin verification failed", http.StatusForbidden)
			return
		}
		allowed, err := d.Authorize(r.Context(), subject, p.ActionID)
		if err != nil {
			http.Error(tw, "authorization unavailable", http.StatusServiceUnavailable)
			return
		}
		if !allowed {
			http.Error(tw, "access denied", http.StatusForbidden)
			return
		}
		d.Invoke.ServeHTTP(tw, r)
		stage = "completed"
	}), nil
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
