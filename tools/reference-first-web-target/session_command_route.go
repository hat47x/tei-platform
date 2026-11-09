//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"tei/internal/composition"
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
	// SelectedAction is an optional trusted Host binding, never sourced from
	// request payload or an untrusted plugin declaration.
	SelectedAction *selectedSessionActionConfig
}

// Only the Composition selection changes here: all registered providers
// invoke the same existing canonical Action Runner in this reference target.
// This does not demonstrate interchangeable physical Action implementations.
type selectedSessionActionConfig struct {
	Coordinator     *composition.Coordinator
	ContractVersion string
	ProviderDigests map[string]string
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
	invoke := http.Handler(http.HandlerFunc(s.handleProtectedSessionCommand))
	if selected := config.SelectedAction; selected != nil {
		registered := make([]webcomposition.SelectedActionProvider, 0, len(selected.ProviderDigests))
		for providerID, digest := range selected.ProviderDigests {
			registered = append(registered, webcomposition.SelectedActionProvider{
				ProviderID: providerID, ArtifactDigest: digest, Invoke: invoke,
			})
		}
		invoke, err = webcomposition.NewSelectedActionInvoker(
			selected.Coordinator, "order_entry.submit_order", selected.ContractVersion, registered)
		if err != nil {
			return fmt.Errorf("session command selection cannot be constructed: %w", err)
		}
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
		Invoke:        invoke,
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
	// The shared Read Runner requires RFC3339. Reject malformed client input
	// before making a business read instead of reporting it as a backend fault.
	if _, err := time.Parse(time.RFC3339, request.From); err != nil {
		http.Error(w, "from must be RFC3339", http.StatusBadRequest)
		return
	}
	result, err := s.protectedSessionConfig.Runner.RunVerifiedSessionCommand(
		r.Context(), s.protectedSessionConfig.SubjectLinks, request.From, request.RowIndex)
	s.writeProtectedSessionCommandHTTPResult(w, result, err)
}

// writeProtectedSessionCommandHTTPResult is intentionally shared by the
// synthetic Workspace HTTP path and the small result-contract test. This
// mapping cannot infer durable Commit completion from a Driver error.
func (s *webTargetServer) writeProtectedSessionCommandHTTPResult(
	w http.ResponseWriter,
	result commandScenarioResult,
	err error,
) {
	// A successful HTTP authorization is not evidence of a completed Action.
	// Map final business denials separately from uncompleted or uncertain
	// execution, and never echo an internal Adapter/Driver error to a browser.
	switch {
	case err == nil && result.Stage == "completed" &&
		result.Command != nil && result.Observation != nil:
		// The normal success envelope has its existing HTTP 200 status.
	case err == nil && result.Stage == "validation-not-satisfied":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
	case err == nil && (result.Stage == "identity-link-denied" || result.Stage == "authorization-denied"):
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
	case err != nil && (result.Stage == "identity-link" ||
		result.Stage == "authorization" || result.Stage == "effect-authorization"):
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		err = errors.New("trusted identity or authorization resolution unavailable")
	default:
		// A Command/Observation execution error does not prove the effect
		// failed: the remote effect may have been committed before an error.
		// Do not recommend blind retries or classify this as an input 422.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		err = errors.New("action completion unconfirmed; inspect authoritative state before retrying")
	}
	s.writeCommandScenarioHTTPResult(w, result, err)
}
