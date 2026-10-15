//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"net/http"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
)

// guardCurrentAcceptedSource prevents a newly arrived HTTP request from
// reaching a Runner/Command after the Host has revoked or replaced the
// Accepted Source used to construct this process's immutable World.
//
// This is a request-boundary Source identity check, NOT a Host effect grant.
// The Host owns the Reader's authenticity and revocation source. Revocation
// during an already-running Command and filesystem changes after startup are
// not closed by this reference middleware. The embedding Host cancels hostCtx
// when its runtime lifetime ends; this is independent of Reader revocation.
func guardCurrentAcceptedSource(hostCtx context.Context, next http.Handler, reader sourceacceptance.Reader, applicationID string, index sourceindex.Index, accepted sourceacceptance.Record) http.Handler {
	// Retain the exact startup Source membership. The runtime never upgrades
	// its World automatically when a new Host Decision appears.
	startup := sourceindex.Index{Artifacts: append([]sourceindex.Artifact(nil), index.Artifacts...)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The Host's lifetime is independent from the individual HTTP
		// request. A revoked/terminated Host must not dispatch new work even
		// if a previously accepted Reader still returns the same witness.
		if hostCtx.Err() != nil {
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE", http.StatusServiceUnavailable)
			return
		}
		current, err := sourceacceptance.VerifyCurrent(r.Context(), reader, applicationID, startup)
		if err != nil || hostCtx.Err() != nil ||
			current.DecisionRef != accepted.DecisionRef ||
			current.AuthorityRevision != accepted.AuthorityRevision ||
			current.ModelRevision != accepted.ModelRevision {
			// Avoid exposing Host provenance or the actual authorization error
			// in the HTTP body, including for a public health endpoint.
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}
