package composition

// ProposalVerdict describes structural feasibility only. In particular,
// a candidate is not proof of conformance, artifact trust or host authority.
type ProposalVerdict string

const (
	ProposalCandidate ProposalVerdict = "candidate"
	ProposalConflict  ProposalVerdict = "conflict"
	ProposalUnknown   ProposalVerdict = "unknown"
)

// ProposalReason is stable machine-readable diagnostic information. A missing
// candidate in a partial registry is Unknown, never proof that code is needed.
type ProposalReason string

const (
	ReasonStructurallyEligible ProposalReason = "structurally-eligible"
	ReasonInvalidRequirement   ProposalReason = "invalid-requirement"
	ReasonNoKnownProvider      ProposalReason = "no-known-provider"
	ReasonAmbiguousProviders   ProposalReason = "ambiguous-providers"
	ReasonExplicitMismatch     ProposalReason = "explicit-provider-mismatch"
	ReasonDependencyIneligible ProposalReason = "dependency-ineligible"
)

// WiringRequirement is a design-time request, not a runtime invocation. When
// ProviderID is empty, an unambiguous match may be proposed by this registry.
type WiringRequirement struct {
	Contract   Contract
	Scope      string
	ProviderID string
}

// WiringFinding records a candidate derived from the current Registry or the
// reason why no candidate can be justified. A returned Selection is copied;
// it may become stale before actual Host admission and must be revalidated.
type WiringFinding struct {
	Verdict    ProposalVerdict
	Reason     ProposalReason
	Candidates []string
	Selected   *Selection
	Detail     string
}

// InspectWiring answers a design-time structural question without executing
// or authorizing a provider. It uses the SAME Registry declarations as runtime
// selection, rather than maintaining an independent AI-side resolver.
// It never asserts that a search over the known registry is exhaustive.
func (r *Registry) InspectWiring(req WiringRequirement) WiringFinding {
	if r == nil || req.Scope == "" || !req.Contract.valid() {
		return WiringFinding{Verdict: ProposalUnknown, Reason: ReasonInvalidRequirement}
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	var matches []string
	for _, id := range sortedKeys(r.providers) {
		p := r.providers[id]
		if p.Scope == req.Scope && p.Offers == req.Contract {
			matches = append(matches, id)
		}
	}
	finding := WiringFinding{Candidates: matches}
	chosen := req.ProviderID
	if chosen != "" {
		p, exists := r.providers[chosen]
		if exists && p.Scope != req.Scope {
			finding.Verdict, finding.Reason = ProposalUnknown, ReasonNoKnownProvider
			return finding
		}
		if exists && p.Offers != req.Contract {
			finding.Verdict, finding.Reason = ProposalConflict, ReasonExplicitMismatch
			return finding
		}
		if !exists {
			finding.Verdict, finding.Reason = ProposalUnknown, ReasonNoKnownProvider
			return finding
		}
	} else {
		switch len(matches) {
		case 0:
			finding.Verdict, finding.Reason = ProposalUnknown, ReasonNoKnownProvider
			return finding
		case 1:
			chosen = matches[0]
		default:
			finding.Verdict, finding.Reason = ProposalConflict, ReasonAmbiguousProviders
			return finding
		}
	}
	assessment, ok := r.snapshot.Assessment(chosen)
	if !ok || !assessment.Eligible {
		finding.Verdict, finding.Reason = ProposalUnknown, ReasonDependencyIneligible
		if ok {
			finding.Detail = assessment.Reason
		}
		return finding
	}
	selected := Selection{
		Revision:     r.snapshot.Revision,
		Epoch:        r.epochs[chosen],
		ProviderID:   chosen,
		Scope:        req.Scope,
		Contract:     req.Contract,
		Dependencies: append([]Edge(nil), assessment.Bindings...),
	}
	finding.Verdict, finding.Reason, finding.Selected = ProposalCandidate, ReasonStructurallyEligible, &selected
	return finding
}
