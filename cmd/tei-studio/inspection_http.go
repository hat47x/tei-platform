package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tei/internal/studiohost"
)

const (
	inspectionHTTPRoute    = "/tei/read-only/v0/inspect"
	inspectionHTTPTokenEnv = "TEI_STUDIO_READ_ONLY_API_TOKEN"
)

// inspectionHTTPService is an opt-in, loopback-only transport projection.
// It is not a Plugin Provider or an independent authority for inspecting a
// workspace. The existing Studio Host retains workspace and meaning checks.
// The caller's bearer token is a local PoC boundary, not a production identity
// or a grant for broader HTTP publication.
type inspectionHTTPService struct {
	host  *studiohost.ReadOnly
	token string
}

func newInspectionHTTPService(host *studiohost.ReadOnly, token string) (*inspectionHTTPService, error) {
	if host == nil {
		return nil, errors.New("read-only Studio Host is required")
	}
	if len(token) < 32 || strings.TrimSpace(token) != token || strings.ContainsAny(token, "\r\n\t") {
		return nil, errors.New("read-only HTTP API token must be at least 32 bytes and contain no surrounding whitespace")
	}
	return &inspectionHTTPService{host: host, token: token}, nil
}

func (s *inspectionHTTPService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Vary", "Authorization")
	// Wails beta.29 strips the registered route prefix and passes an empty
	// path for the exact route. A direct net/http test sees the full path.
	if r.URL.Path != inspectionHTTPRoute && r.URL.Path != "/" && r.URL.Path != "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// No cross-origin browser clients in this first local, read-only witness.
	// CORS, a browser session and external publication need a separate owner.
	if r.Header.Get("Origin") != "" {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.token)) != 1 {
		http.Error(w, "authorization required", http.StatusUnauthorized)
		return
	}
	q := r.URL.Query()
	if len(q) != 3 || len(q["workspace"]) != 1 || len(q["kind"]) != 1 || len(q["subject"]) != 1 ||
		q.Get("workspace") == "" || q.Get("kind") == "" || q.Get("subject") == "" {
		http.Error(w, "workspace, kind and subject are required exactly once", http.StatusBadRequest)
		return
	}

	result := s.host.InspectSubject(q.Get("workspace"), q.Get("kind"), q.Get("subject"))
	status := http.StatusOK
	if !result.OK {
		status = http.StatusInternalServerError
		if result.Failure != nil {
			switch result.Failure.Code {
			case "WORKSPACE_NOT_ALLOWED", "WORKSPACE_ROOT_CHANGED":
				status = http.StatusForbidden
			case "SUBJECT_NOT_FOUND":
				status = http.StatusNotFound
			case "INVALID_SUBJECT", "UNSUPPORTED_SUBJECT_KIND", "SUBJECT_AMBIGUOUS":
				status = http.StatusBadRequest
			}
			// Do not leak local absolute paths, provider messages or other
			// details from the Studio error into a new HTTP audience.
			copy := *result.Failure
			copy.Message = "inspection rejected"
			result.Failure = &copy
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
