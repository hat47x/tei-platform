package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"tei/internal/realization"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
	webcomposition "tei/reference/web-composition/go-host"
)

// protectedSessionCommandConfig is explicitly assembled by a trusted Host.
// It is not read from request headers, parameters, or a plugin declaration.
// The normal server constructor does not expose a session route by default.
type protectedSessionCommandConfig struct {
	Runner        *referenceCommandScenarioRunner
	Providers     map[string]string
	VerifySession func(*http.Request) (webcomposition.Principal, error)
	VerifyCSRF    func(*http.Request, webcomposition.Principal) bool
	SubjectLinks  webcomposition.SubjectLinkLookup
	AccountNow    func() time.Time
	Audit         func(webcomposition.AuditRecord)
}

func withProtectedSessionCommand(config protectedSessionCommandConfig) webTargetOption {
	return func(s *webTargetServer) {
		s.protectedSessionOptionCount++
		s.protectedSessionConfig = &config
	}
}

// configureProtectedSessionCommand composes a protected route exactly once.
// A caller cannot insert a bare http.Handler in place of the required guards.
func (s *webTargetServer) configureProtectedSessionCommand() error {
	if s.protectedSessionOptionCount > 1 {
		return errors.New("duplicate protected session command route configuration")
	}
	config := s.protectedSessionConfig
	if config == nil {
		return nil
	}
	if config.Runner == nil || config.SubjectLinks == nil || s.projection.ApplicationID != "order_entry" {
		return errors.New("session command needs order_entry, a bound runner and trusted subject links")
	}
	activeLinks, err := webcomposition.NewActiveLegacySubjectLinks(config.SubjectLinks, config.Runner.store, config.AccountNow)
	if err != nil {
		return fmt.Errorf("session command account eligibility cannot be constructed: %w", err)
	}
	// Both the Host early authorization and the command execution resolve current
	// account eligibility; a revoked link cannot be reused after early approval.
	config.SubjectLinks = activeLinks
	legacyRunner, ok := s.commandScenario.(*referenceCommandScenarioRunner)
	if !ok || legacyRunner != config.Runner {
		return errors.New("session command must share the configured order_entry command runner")
	}
	adapter := legacyauthorization.Adapter{Store: config.Runner.store, Rule: config.Runner.authorization}
	adapters := bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		if binding.Identity() != "order_entry/security.authorize" {
			return nil, fmt.Errorf("unexpected authorization Binding %s", binding.Identity())
		}
		return adapter, nil
	})
	authorize, err := webcomposition.NewCanonicalActionAuthorizer(
		config.Runner.resolver, "order_entry", "order_entry.submit_order", config.SubjectLinks, adapters)
	if err != nil {
		return fmt.Errorf("session command authorization cannot be constructed: %w", err)
	}
	handler, err := webcomposition.NewHost(webcomposition.Policy{
		Path:      "/scenario/session-submit",
		Method:    http.MethodPost,
		ActionID:  "order_entry.submit_order",
		Providers: config.Providers,
	}, webcomposition.Dependencies{
		VerifySession: config.VerifySession,
		VerifyCSRF:    config.VerifyCSRF,
		Authorize:     authorize,
		Invoke:        http.HandlerFunc(s.handleProtectedSessionCommand),
		Audit:         config.Audit,
	})
	if err != nil {
		return fmt.Errorf("session command route is missing required protection: %w", err)
	}
	s.protectedSessionHandler = handler
	return nil
}

type webSessionCommandRequest struct {
	From     string `json:"from"`
	RowIndex int    `json:"row"`
}

// handleProtectedSessionCommand is only registered behind NewHost's
// route, session, issuer, CSRF and Action authorization enforcement.
// decodeWebSessionCommandRequest accepts only the two explicitly named fields.
// A struct Decoder with DisallowUnknownFields accepts duplicate members and
// case-folded aliases; neither is appropriate at an authenticated command boundary.
func decodeWebSessionCommandRequest(w http.ResponseWriter, r *http.Request) (webSessionCommandRequest, error) {
	var request webSessionCommandRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return request, errors.New("session command expects one JSON object")
	}
	seen := map[string]bool{}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return request, errors.New("invalid session command member")
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return request, errors.New("invalid or duplicate session command member")
		}
		seen[name] = true
		switch name {
		case "from":
			var value *string
			if err := decoder.Decode(&value); err != nil || value == nil {
				return request, errors.New("session command from must be a string")
			}
			request.From = *value
		case "row":
			var value *int
			if err := decoder.Decode(&value); err != nil || value == nil {
				return request, errors.New("session command row must be an integer")
			}
			request.RowIndex = *value
		default:
			return request, errors.New("unknown session command member")
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return request, errors.New("invalid session command object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return request, errors.New("session command expects one JSON document")
	}
	return request, nil
}

func (s *webTargetServer) handleProtectedSessionCommand(w http.ResponseWriter, r *http.Request) {
	request, err := decodeWebSessionCommandRequest(w, r)
	if err != nil {
		http.Error(w, "invalid session command request", http.StatusBadRequest)
		return
	}
	if request.RowIndex < 0 {
		http.Error(w, "row must be a non-negative integer", http.StatusBadRequest)
		return
	}
	if request.From == "" {
		request.From = "2026-09-01T00:00:00Z"
	}
	result, err := s.protectedSessionConfig.Runner.RunVerifiedSessionCommand(
		r.Context(), s.protectedSessionConfig.SubjectLinks, request.From, request.RowIndex)
	// A subject link may be revoked or the account may become ineligible
	// between the Host's early authorization and the Runner's fresh check.
	// Preserve the Business Result / Execution Error envelope while making
	// the HTTP status reflect that the request was not completed.
	if (result.Stage == "identity-link-denied" || result.Stage == "authorization-denied") && err == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
	} else if err != nil && (result.Stage == "identity-link" ||
		result.Stage == "authorization" || result.Stage == "effect-authorization") {
		// Do not return messages from trusted account and permission backends.
		// Both the initial authorization and the effect-time recheck are
		// security infrastructure checks, not untrusted input validation.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		err = errors.New("trusted identity or authorization resolution unavailable")
	}
	s.writeCommandScenarioHTTPResult(w, result, err)
}
