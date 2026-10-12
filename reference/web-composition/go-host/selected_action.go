package webcomposition

import (
	"errors"
	"net/http"

	"tei/internal/composition"
)

// SelectedActionProvider is a host-owned mapping from a selected logical
// service to its concrete invocation entry. The digest is an expected artifact
// identity, not evidence that bytes or signatures were verified.
type SelectedActionProvider struct {
	ProviderID     string
	ArtifactDigest string
	Invoke         http.Handler
}

// NewSelectedActionInvoker connects a previously protected Action endpoint to
// one selected composition generation. Place this handler INSIDE NewHost's
// Invoke callback; it neither authenticates callers nor authorizes an Action.
//
// Provider registrations and exact artifact identities belong to the trusted
// Host. Browser input or untrusted plugin declarations must not build them.
// Composition.Acquire applies the Host's current guard; this adapter does not
// replace the original Action's authorization or effect-time checks.
// The Host must also have authorized this exact Action in the current request.
// A selected handler registered directly on an unprotected HTTP route refuses
// dispatch even when a valid composition generation is available.
func NewSelectedActionInvoker(
	coordinator *composition.Coordinator,
	contractID, contractVersion string,
	providers []SelectedActionProvider,
) (http.Handler, error) {
	if coordinator == nil || contractID == "" || contractVersion == "" || len(providers) == 0 {
		return nil, errors.New("selected action needs coordinator, exact contract and providers")
	}
	registered := make(map[string]SelectedActionProvider, len(providers))
	for _, candidate := range providers {
		if candidate.ProviderID == "" || candidate.ArtifactDigest == "" || candidate.Invoke == nil {
			return nil, errors.New("selected action provider needs identity, artifact and handler")
		}
		if _, duplicate := registered[candidate.ProviderID]; duplicate {
			return nil, errors.New("selected action provider is duplicated")
		}
		registered[candidate.ProviderID] = candidate
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFromContext(r.Context()); !ok ||
			r.Context().Value(authorizedActionContextKey{}) != contractID {
			// A selected Action is not itself an authentication/authorization
			// boundary. Never acquire a Provider from a bare route or reuse an
			// approval for another Action, even when the caller forges headers.
			http.Error(w, "protected action authorization required", http.StatusForbidden)
			return
		}
		lease, err := coordinator.Acquire()
		if err != nil {
			http.Error(w, "action provider unavailable", http.StatusServiceUnavailable)
			return
		}
		defer lease.Release()
		bound := lease.Binding
		provider, ok := registered[bound.ProviderID]
		if !ok || bound.ContractID != contractID || bound.ContractVersion != contractVersion ||
			provider.ArtifactDigest != bound.ArtifactDigest {
			http.Error(w, "selected action binding unavailable", http.StatusServiceUnavailable)
			return
		}
		provider.Invoke.ServeHTTP(w, r)
	}), nil
}
