package composition

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

var ErrStaleRegistry = errors.New("composition registry revision changed")

// Contract is a transport- and package-independent service interface identity.
// Shape must be a fingerprint of the typed request/result/error contract.
type Contract struct {
	ID      string
	Version string
	Shape   string
}

func (c Contract) valid() bool { return c.ID != "" && c.Version != "" && c.Shape != "" }

// Dependency is a required coeffect. ProviderID selects a specific implementation;
// an empty ProviderID is valid only when exactly one provider matches.
type Dependency struct {
	Contract   Contract
	ProviderID string
}

// Provider describes available implementations, not a grant of runtime authority.
// Scope is an exact isolation boundary; cross-scope lookup is forbidden.
type Provider struct {
	ID       string
	Scope    string
	Offers   Contract
	Requires []Dependency
}

func (p Provider) validate() error {
	if p.ID == "" || p.Scope == "" || !p.Offers.valid() {
		return errors.New("provider ID, scope and complete contract are required")
	}
	seen := map[string]bool{}
	for _, req := range p.Requires {
		if !req.Contract.valid() {
			return fmt.Errorf("provider %s has invalid dependency contract", p.ID)
		}
		k := req.Contract.ID
		if seen[k] {
			return fmt.Errorf("provider %s repeats dependency %s", p.ID, k)
		}
		seen[k] = true
	}
	return nil
}

// Edge identifies an exact provider-to-provider dependency, never a package dependency.
type Edge struct {
	ContractID string
	ProviderID string
}

// Assessment is declarative dependency eligibility, NOT actual process readiness,
// artifact trust, permission, or an activated instance.
type Assessment struct {
	ProviderID string
	Eligible   bool
	Reason     string
	Bindings   []Edge
}

// Transition records a change of eligibility after an atomic registry edit.
// Owners may reconcile lifecycle changes outside the registry lock.
type Transition struct {
	ProviderID string
	From       string
	To         string
}

type Snapshot struct {
	Revision        uint64
	Providers       []Assessment
	ActivationOrder []string
	Transitions     []Transition
}

func (s Snapshot) Assessment(id string) (Assessment, bool) {
	for _, p := range s.Providers {
		if p.ProviderID == id {
			return p, true
		}
	}
	return Assessment{}, false
}

func cloneSnapshot(s Snapshot) Snapshot {
	out := Snapshot{Revision: s.Revision,
		ActivationOrder: append([]string(nil), s.ActivationOrder...),
		Transitions:     append([]Transition(nil), s.Transitions...)}
	for _, p := range s.Providers {
		p.Bindings = append([]Edge(nil), p.Bindings...)
		out.Providers = append(out.Providers, p)
	}
	return out
}

// Registry is an atomic registry of provider declarations. It has no background
// goroutines, callbacks under locks, or implicit lifecycle side effects.
type Registry struct {
	mu        sync.Mutex
	providers map[string]Provider
	snapshot  Snapshot
	epochs    map[string]uint64
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider), epochs: make(map[string]uint64)}
}

func (r *Registry) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneSnapshot(r.snapshot)
}

// Apply atomically changes provider declarations if expected matches the revision.
// Missing or ambiguous dependencies remain ineligible and are never guessed.
// Cycles and invalid declarations abort the entire update without modifying state.
func (r *Registry) Apply(expected uint64, upserts []Provider, removes []string) (Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.snapshot.Revision != expected {
		return Snapshot{}, ErrStaleRegistry
	}
	candidate := make(map[string]Provider, len(r.providers)+len(upserts))
	for k, v := range r.providers {
		candidate[k] = v
	}
	removed := map[string]bool{}
	for _, id := range removes {
		if id == "" || removed[id] {
			return Snapshot{}, fmt.Errorf("invalid or duplicate remove %q", id)
		}
		removed[id] = true
		delete(candidate, id)
	}
	seen := map[string]bool{}
	for _, p := range upserts {
		if err := p.validate(); err != nil {
			return Snapshot{}, err
		}
		if seen[p.ID] || removed[p.ID] {
			return Snapshot{}, fmt.Errorf("provider %s has conflicting edits", p.ID)
		}
		seen[p.ID] = true
		p.Requires = append([]Dependency(nil), p.Requires...)
		candidate[p.ID] = p
	}
	if expected == ^uint64(0) {
		return Snapshot{}, errors.New("registry revision exhausted")
	}
	resolved, err := evaluate(candidate)
	if err != nil {
		return Snapshot{}, err
	}
	resolved.Revision = expected + 1
	old := map[string]string{}
	for _, p := range r.snapshot.Providers {
		old[p.ProviderID] = status(p, r.providers[p.ProviderID])
	}
	current := map[string]string{}
	for _, p := range resolved.Providers {
		current[p.ProviderID] = status(p, candidate[p.ProviderID])
	}
	all := map[string]bool{}
	for id := range old {
		all[id] = true
	}
	for id := range current {
		all[id] = true
	}
	ids := sortedKeys(all)
	for _, id := range ids {
		if old[id] != current[id] {
			resolved.Transitions = append(resolved.Transitions, Transition{id, old[id], current[id]})
		}
	}
	for _, change := range resolved.Transitions {
		if r.epochs[change.ProviderID] == ^uint64(0) {
			return Snapshot{}, fmt.Errorf("provider epoch exhausted: %s", change.ProviderID)
		}
	}
	for _, change := range resolved.Transitions {
		r.epochs[change.ProviderID]++
	}
	r.providers, r.snapshot = candidate, resolved
	return cloneSnapshot(resolved), nil
}

func status(p Assessment, definition Provider) string {
	return fmt.Sprintf("eligible=%t reason=%s bindings=%v definition=%v", p.Eligible, p.Reason, p.Bindings, definition)
}

func sortedKeys[V any](m map[string]V) []string {
	ids := make([]string, 0, len(m))
	for k := range m {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	return ids
}

func evaluate(providers map[string]Provider) (Snapshot, error) {
	var out Snapshot
	memo := map[string]Assessment{}
	visiting := map[string]bool{}
	var resolve func(string) (Assessment, error)
	resolve = func(id string) (Assessment, error) {
		if visiting[id] {
			return Assessment{}, fmt.Errorf("coeffect dependency cycle involving %s", id)
		}
		if existing, ok := memo[id]; ok {
			return existing, nil
		}
		p := providers[id]
		visiting[id] = true
		a := Assessment{ProviderID: id, Eligible: true}
		for _, req := range p.Requires {
			var matches []string
			for _, otherID := range sortedKeys(providers) {
				other := providers[otherID]
				if other.Scope == p.Scope && other.Offers == req.Contract && (req.ProviderID == "" || req.ProviderID == otherID) {
					matches = append(matches, otherID)
				}
			}
			if len(matches) == 0 {
				if a.Eligible {
					a.Eligible, a.Reason = false, fmt.Sprintf("missing %s in scope %s", req.Contract.ID, p.Scope)
				}
				continue
			}
			if len(matches) > 1 {
				if a.Eligible {
					a.Eligible, a.Reason = false, fmt.Sprintf("ambiguous %s in scope %s", req.Contract.ID, p.Scope)
				}
				continue
			}
			dependency, err := resolve(matches[0])
			if err != nil {
				return Assessment{}, err
			}
			if !dependency.Eligible {
				if a.Eligible {
					a.Eligible, a.Reason = false, fmt.Sprintf("dependency %s ineligible: %s", matches[0], dependency.Reason)
				}
				continue
			}
			a.Bindings = append(a.Bindings, Edge{req.Contract.ID, matches[0]})
		}
		visiting[id] = false
		memo[id] = a
		if a.Eligible {
			out.ActivationOrder = append(out.ActivationOrder, id)
		}
		return a, nil
	}
	for _, id := range sortedKeys(providers) {
		if _, err := resolve(id); err != nil {
			return Snapshot{}, err
		}
	}
	for _, id := range sortedKeys(memo) {
		out.Providers = append(out.Providers, memo[id])
	}
	return out, nil
}
