package composition

import (
	"errors"
	"fmt"
	"slices"
	"sort"
)

var (
	ErrProviderIneligible = errors.New("provider is not eligible")
	ErrSelectionMismatch  = errors.New("provider selection does not match contract or scope")
)

// ProviderEpoch pins the declaration identity of a transitive dependency.
// It does not attest the corresponding runtime instance or artifact bytes.
type ProviderEpoch struct {
	ProviderID string
	Epoch      uint64
}

// Selection is a declaration snapshot, never an invocation authority or an
// artifact attestation. The host must separately validate current permission,
// leases, artifact provenance and runtime readiness before serving requests.
type Selection struct {
	Revision         uint64
	Epoch            uint64
	ProviderID       string
	Scope            string
	Contract         Contract
	Dependencies     []Edge
	DependencyEpochs []ProviderEpoch
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
	return r.selectionLocked(p, a), nil
}

// selectionLocked copies a structurally eligible candidate with the complete
// transitive dependency declaration closure. Caller must hold r.mu.
func (r *Registry) selectionLocked(p Provider, a Assessment) Selection {
	return Selection{
		Revision:         r.snapshot.Revision,
		Epoch:            r.epochs[p.ID],
		ProviderID:       p.ID,
		Scope:            p.Scope,
		Contract:         p.Offers,
		Dependencies:     append([]Edge(nil), a.Bindings...),
		DependencyEpochs: r.dependencyEpochsLocked(p.ID),
	}
}

// dependencyEpochsLocked captures every unique dependent provider, including
// indirect dependencies. evaluate() rejects cycles before these are exposed.
func (r *Registry) dependencyEpochsLocked(id string) []ProviderEpoch {
	// Build a call-local lookup from the Registry's authoritative snapshot.
	// Looking up an Assessment linearly at every dependency edge makes deep
	// selections quadratic. A call-local index avoids a second mutable cache
	// that would need synchronization with Registry.Apply.
	assessments := make(map[string]Assessment, len(r.snapshot.Providers))
	for _, assessment := range r.snapshot.Providers {
		assessments[assessment.ProviderID] = assessment
	}
	seen := map[string]bool{id: true}
	var walk func(string)
	walk = func(current string) {
		a, ok := assessments[current]
		if !ok || !a.Eligible {
			return
		}
		for _, edge := range a.Bindings {
			if seen[edge.ProviderID] {
				continue
			}
			seen[edge.ProviderID] = true
			walk(edge.ProviderID)
		}
	}
	walk(id)
	delete(seen, id)
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var deps []ProviderEpoch
	for _, id := range ids {
		deps = append(deps, ProviderEpoch{ProviderID: id, Epoch: r.epochs[id]})
	}
	return deps
}

// validateSelectionLocked must be called while holding r.mu. It rechecks all
// relevant contract and wiring identities, not just a caller-provided revision.
func (r *Registry) validateSelectionLocked(s Selection, b Binding) error {
	if s.Revision > r.snapshot.Revision || s.Epoch == 0 || r.epochs[s.ProviderID] != s.Epoch {
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
	if !slices.Equal(r.dependencyEpochsLocked(s.ProviderID), s.DependencyEpochs) {
		return ErrStaleRegistry
	}
	return nil
}
