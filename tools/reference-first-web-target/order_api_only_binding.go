//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import "net/http"

// This diagnostic endpoint checks the target process's actual B/R/T at the
// point of command dispatch. Browser->Host uses /scenario/submit unchanged;
// the diagnostic Host alone routes to this distinct API-only endpoint.
//
// Header values are NOT credentials or proof of Host authority. An untrusted
// caller can supply them; this only prevents accidentally forwarding a command
// to a different API build after a preceding manifest observation.
const (
	apiBoundCommandPath       = "/scenario/submit-bound"
	apiBoundReadPath          = "/scenario/read-bound"
	apiBoundApplicationHeader = "X-Tei-Diagnostic-Application"
	apiBoundBaselineHeader    = "X-Tei-Diagnostic-Baseline-Id"
	apiBoundRevisionHeader    = "X-Tei-Diagnostic-Baseline-Revision"
	apiBoundTargetHeader      = "X-Tei-Diagnostic-Target-Revision"
)

func exactlyOneExpectedHeader(r *http.Request, name, expected string) bool {
	values := r.Header.Values(name)
	return expected != "" && len(values) == 1 && values[0] == expected
}

func (s *webTargetServer) matchesDiagnosticBinding(r *http.Request) bool {
	return exactlyOneExpectedHeader(r, apiBoundApplicationHeader, s.projection.ApplicationID) &&
		exactlyOneExpectedHeader(r, apiBoundBaselineHeader, s.projection.HandoverBaselineID) &&
		exactlyOneExpectedHeader(r, apiBoundRevisionHeader, s.projection.BaselineRepositoryRevision) &&
		exactlyOneExpectedHeader(r, apiBoundTargetHeader, s.projection.TargetBuildRevision)
}

func (s *webTargetServer) handleBoundCommandScenario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.matchesDiagnosticBinding(r) {
		// A mismatched target is not a Business Result or an Authorization
		// decision. Refuse before decoding the request or calling the Runner.
		http.Error(w, "diagnostic API build binding mismatch", http.StatusConflict)
		return
	}
	s.handleCommandScenario(w, r)
}

// handleBoundReadScenario applies the same in-process B/R/T comparison to
// read-only Business Data dispatch. This is a diagnostic equality guard,
// never an authentication or authorization token. The existing direct
// /scenario/read route and its Runner remain untouched.
func (s *webTargetServer) handleBoundReadScenario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.matchesDiagnosticBinding(r) {
		http.Error(w, "diagnostic API build binding mismatch", http.StatusConflict)
		return
	}
	s.handleReadScenario(w, r)
}
