package composition

import (
	"errors"
	"fmt"
	"slices"
)

var (
	ErrProviderIneligible = errors.New("provider is not eligible")
	ErrSelectionMismatch  = errors.New("provider selection does not match contract or scope")
)

// Selection is a declaration snapshot, never an invocation authority or an
// artifact attestation. The host must separately validate current permission,
// leases, artifact provenance and runtime readiness before serving requests.
type Selection struct {
	Revision     uint64
	ProviderID   string
	Scope        string
	Contract     Contract
	Dependencies []Edge
}

// Select returns a defensive copy of an eligible provider selection. A caller
// cannot use another tenant's scope or silently choose an ambiguous provider.
func (r *Registry) Select(providerID, scope string) (Selection, error) {
	if r == nil {
		return Selection{}, ErrSelectionMismatch
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.providers[providerID]
	if !ok || scope == "" || p.Scope != scope {
		return Selection{}, ErrSelectionMismatch
	}
	a, ok := r.snapshot.Assessment(providerID)
	if !ok || !a.Eligible {
		return Selection{}, fmt.Errorf("%w: %s", ErrProviderIneligible, providerID)
	}
	return Selection{
		Revision:     r.snapshot.Revision,
		ProviderID:   p.ID,
		Scope:        p.Scope,
		Contract:     p.Offers,
		Dependencies: append([]Edge(nil), a.Bindings...),
	}, nil
}

// validateSelectionLocked must be called while holding r.mu. It rechecks all
// relevant contract and wiring identities, not just a caller-provided revision.
func (r *Registry) validateSelectionLocked(s Selection, b Binding) error {
	if r.snapshot.Revision != s.Revision {
		return ErrStaleRegistry
	}
	p, ok := r.providers[s.ProviderID]
	if !ok || p.Scope != s.Scope || p.Offers != s.Contract ||
		b.ProviderID != s.ProviderID || b.ContractID != s.Contract.ID ||
		b.ContractVersion != s.Contract.Version || !s.Contract.valid() {
		return ErrSelectionMismatch
	}
	a, ok := r.snapshot.Assessment(s.ProviderID)
	if !ok || !a.Eligible {
		return ErrProviderIneligible
	}
	if !slices.Equal(a.Bindings, s.Dependencies) {
		return ErrSelectionMismatch
	}
	return nil
}
