package actionboundary

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// PayloadDecoder translates one application-owned JSON payload to the native
// command input. It must reject unknown actions and invalid payloads itself.
type PayloadDecoder func(actionID string, raw json.RawMessage) (any, error)

// JSONCommitHandler is a reference-only HTTP transport for native UI actions.
// Production runtime must supply Host authorization backed by a trusted session,
// a command resolver, and an application-owned persistent Store.
type JSONCommitHandler struct {
	Host          Host
	DecodePayload PayloadDecoder
}

type httpCommitIntent struct {
	ApplicationID    string          `json:"applicationID"`
	DocumentID       string          `json:"documentID"`
	ActionID         string          `json:"actionID"`
	ExpectedRevision string          `json:"expectedRevision"`
	Payload          json.RawMessage `json:"payload"`
}

// ServeHTTP does not use claims of subject identity from the request body.
// Requiring a non-simple header and same-origin requests constrains browser
// CSRF, but does not replace runtime authorization or a deployment CSRF policy.
func (h JSONCommitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	reject := func(status int, code string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		reject(http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if r.Header.Get("X-TEI-Action") != "commit" || !sameHostOrigin(r) {
		reject(http.StatusForbidden, "request_origin_denied")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		reject(http.StatusUnsupportedMediaType, "unsupported_content_type")
		return
	}
	if h.DecodePayload == nil {
		reject(http.StatusServiceUnavailable, "payload_decoder_unavailable")
		return
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	var input httpCommitIntent
	if err := decoder.Decode(&input); err != nil {
		reject(http.StatusBadRequest, "invalid_intent")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		reject(http.StatusBadRequest, "invalid_intent")
		return
	}
	if input.ApplicationID == "" || input.DocumentID == "" || input.ActionID == "" || input.ExpectedRevision == "" ||
		len(input.Payload) == 0 || string(input.Payload) == "null" {
		reject(http.StatusBadRequest, "invalid_intent")
		return
	}
	payload, err := h.DecodePayload(input.ActionID, input.Payload)
	if err != nil {
		reject(http.StatusBadRequest, "invalid_payload")
		return
	}
	revision, err := h.Host.Commit(r.Context(), Intent{
		ApplicationID: input.ApplicationID, DocumentID: input.DocumentID, ActionID: input.ActionID,
		ExpectedRevision: input.ExpectedRevision, Payload: payload,
	})
	switch {
	case errors.Is(err, ErrDenied):
		reject(http.StatusForbidden, "action_denied")
	case errors.Is(err, ErrConflict):
		reject(http.StatusConflict, "revision_conflict")
	case errors.Is(err, ErrInvalid):
		reject(http.StatusBadRequest, "invalid_action")
	case err != nil || revision == "":
		reject(http.StatusInternalServerError, "execution_failed")
	default:
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"revision": revision})
	}
}

func sameHostOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") &&
		u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == "" &&
		strings.EqualFold(u.Host, r.Host)
}
